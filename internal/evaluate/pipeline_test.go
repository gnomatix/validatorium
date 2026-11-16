package evaluate

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gnomatix/validatorium/internal/bbolt"
	"gnomatix/validatorium/internal/model"
	"gnomatix/validatorium/internal/promote"
	"gnomatix/validatorium/internal/release"
	"gnomatix/validatorium/internal/storage/memory"
)

// TestEvaluatePromoteRelease runs the whole path with no hand-made
// attestation: a runtime evaluation of a real file produces the attestation,
// promotion admits the premise on it, and a release tags and publishes it.
// A false evaluation of a second premise is refused at promotion and left out
// of the release.
func TestEvaluatePromoteRelease(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	ws, err := bbolt.OpenWorkingStore(filepath.Join(dir, "w.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()
	gs := memory.NewGlobalStore()
	reg, _ := NewRegistry(FileDigest{})

	good := filepath.Join(dir, "good.txt")
	_ = os.WriteFile(good, []byte("expected\n"), 0o600)
	bad := filepath.Join(dir, "bad.txt")
	_ = os.WriteFile(bad, []byte("not what the premise says\n"), 0o600)

	pGood := filePremise(t, good)
	pBad := filePremise(t, bad)
	for _, p := range []*model.Micropublication{pGood, pBad} {
		if err := ws.PutCandidate(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	scope := map[string]interface{}{"sha256": digest([]byte("expected\n"))}

	attGood, _, err := reg.Attest(ctx, pGood, scope, "evaluator:pipeline", at)
	if err != nil || attGood.Result != model.ResultTrue {
		t.Fatalf("good evaluation: %v %v", attGood, err)
	}
	attBad, _, err := reg.Attest(ctx, pBad, scope, "evaluator:pipeline", at)
	if err != nil || attBad.Result != model.ResultFalse {
		t.Fatalf("bad evaluation: %v %v", attBad, err)
	}

	pr := &promote.Promoter{Working: ws, Global: gs, Now: func() time.Time { return at.Add(time.Minute) }}
	if _, err := pr.Promote(ctx, pGood.UUID, attGood); err != nil {
		t.Fatalf("promote good: %v", err)
	}
	if _, err := pr.Promote(ctx, pBad.UUID, attBad); !errors.Is(err, promote.ErrNotValid) {
		t.Fatalf("promote bad: expected ErrNotValid, got %v", err)
	}
	// The false result is still part of the evaluation history.
	if err := ws.PutAttestation(ctx, attBad); err != nil {
		t.Fatal(err)
	}

	relAt := at.Add(time.Hour)
	res, err := release.Build(ctx, ws, "pipeline-1", relAt, []string{pGood.UUID, pBad.UUID})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(res.Manifest.Premises) != 1 || res.Manifest.Premises[0].UUID != pGood.UUID {
		t.Fatalf("release tagged %+v, want only the true premise", res.Manifest.Premises)
	}
	if len(res.Exclusions) != 1 || res.Exclusions[0].UUID != pBad.UUID {
		t.Fatalf("exclusions = %+v", res.Exclusions)
	}
	if _, err := release.Publish(ctx, gs, ws, res, relAt); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if _, err := gs.GetBlock(ctx, res.CID); err != nil {
		t.Fatalf("published manifest not in the global store: %v", err)
	}
}
