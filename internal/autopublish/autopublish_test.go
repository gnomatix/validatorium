package autopublish

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gnomatix/validatorium/internal/bbolt"
	"gnomatix/validatorium/internal/evaluate"
	"gnomatix/validatorium/internal/model"
	"gnomatix/validatorium/internal/namespace"
	"gnomatix/validatorium/internal/promote"
	"gnomatix/validatorium/internal/storage"
	"gnomatix/validatorium/internal/storage/memory"
)

var at = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

func filePremise(path string) *model.Micropublication {
	m, _ := model.New("The file at "+path+" has the stated SHA-256 digest.", model.Attribution{Agent: "key:abc"})
	m.SupportGraph = []model.Evidence{
		{Kind: model.EvidenceMethod, Reference: evaluate.FileDigestMethod},
		{Kind: model.EvidenceData, Reference: "file://" + path},
	}
	return m
}

func TestRunPublishesOnlyValidPremises(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	ws, err := bbolt.OpenWorkingStore(filepath.Join(dir, "w.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()
	gs := memory.NewGlobalStore()
	reg, _ := evaluate.NewRegistry(evaluate.FileDigest{})

	good := filepath.Join(dir, "good.txt")
	_ = os.WriteFile(good, []byte("expected\n"), 0o600)
	bad := filepath.Join(dir, "bad.txt")
	_ = os.WriteFile(bad, []byte("other\n"), 0o600)
	missing := filepath.Join(dir, "missing.txt")

	pGood, pBad, pMissing := filePremise(good), filePremise(bad), filePremise(missing)
	for _, p := range []*model.Micropublication{pGood, pBad, pMissing} {
		if err := ws.PutCandidate(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	sum := sha256.Sum256([]byte("expected\n"))
	scope := map[string]interface{}{"sha256": hex.EncodeToString(sum[:])}

	pub := &Publisher{
		Promoter:   &promote.Promoter{Working: ws, Global: gs, Now: func() time.Time { return at }},
		Evaluators: reg,
		Evaluator:  "evaluator:autopublish-test",
		Namespaces: []namespace.Name{"validatorium:test"},
		Now:        func() time.Time { return at },
	}
	outs, err := pub.RunAll(ctx, []string{pGood.UUID, pBad.UUID, pMissing.UUID}, scope)
	if err != nil {
		t.Fatal(err)
	}

	// True: published without any separate push step.
	if !outs[0].Published || outs[0].Attestation.Result != model.ResultTrue {
		t.Fatalf("good: %+v", outs[0])
	}
	ptr, err := gs.GetPointer(ctx, pGood.UUID)
	if err != nil || ptr.CID != outs[0].Promotion.Version.CID {
		t.Fatalf("global pointer: %+v %v", ptr, err)
	}
	if _, err := gs.GetNamespacedPointer(ctx, "validatorium:test", pGood.UUID); err != nil {
		t.Fatalf("namespaced pointer: %v", err)
	}

	// False and indeterminate: not published, attestation recorded.
	for i, want := range []model.AttestationResult{model.ResultFalse, model.ResultIndeterminate} {
		o := outs[i+1]
		if o.Published || o.Attestation.Result != want || o.Reason == "" {
			t.Fatalf("outcome %d: %+v", i+1, o)
		}
		id := o.Attestation.PremiseUUID
		if _, err := gs.GetPointer(ctx, id); !errors.Is(err, storage.ErrNotFound) {
			t.Fatalf("%s reached the global store: %v", want, err)
		}
		atts, err := ws.ListAttestations(ctx, id)
		if err != nil || len(atts) != 1 || atts[0].Result != want {
			t.Fatalf("%s attestation not recorded: %v %v", want, atts, err)
		}
		if _, err := ws.GetCandidate(ctx, id); err != nil {
			t.Fatalf("%s candidate should stay a draft: %v", want, err)
		}
	}
}

func TestRunRequiresConfiguration(t *testing.T) {
	if _, err := (&Publisher{}).Run(context.Background(), "x", nil); err == nil {
		t.Fatal("unconfigured publisher ran")
	}
}
