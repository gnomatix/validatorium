package release

import (
	"context"
	"errors"
	"testing"
	"time"

	"gnomatix/validatorium/internal/model"
	"gnomatix/validatorium/internal/promote"
	"gnomatix/validatorium/internal/storage"
	"gnomatix/validatorium/internal/storage/memory"
)

// promoted creates a candidate, promotes it under a true attestation that
// relies on term, and returns its UUID.
func promoted(t *testing.T, e env, gs *memory.GlobalStore, text string, term *model.TermRecord) string {
	t.Helper()
	ctx := context.Background()
	m, _ := model.New(text, model.Attribution{Agent: "key:abc"})
	if err := e.ws.PutCandidate(ctx, m); err != nil {
		t.Fatal(err)
	}
	data, _ := m.CanonicalBytes()
	cid, _ := storage.ContentID(data)
	a, _ := model.NewAttestation(m.UUID, cid, model.ResultTrue, "evaluator:test", "ipfs://method", relAt.Add(-time.Hour))
	if term != nil {
		a.TermRecordUUIDs = []string{term.UUID}
	}
	p := &promote.Promoter{Working: e.ws, Global: gs, Now: func() time.Time { return relAt.Add(-30 * time.Minute) }}
	if _, err := p.Promote(ctx, m.UUID, a); err != nil {
		t.Fatalf("promote: %v", err)
	}
	return m.UUID
}

func TestPublishStoresManifestTermsAndPointer(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	gs := memory.NewGlobalStore()
	term, _ := model.NewTermRecord("water", "http://www.wikidata.org/entity/Q283", model.Attribution{Agent: "key:abc"})
	termRef, _ := e.ws.PutTermRecord(ctx, term)

	id := promoted(t, e, gs, "Pure water boils at 100 °C at 1 atm.", term)
	res, err := Build(ctx, e.ws, "2026.07", relAt, []string{id})
	if err != nil || len(res.Manifest.Premises) != 1 {
		t.Fatalf("Build: %+v %v", res, err)
	}

	ptr, err := Publish(ctx, gs, e.ws, res, relAt)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if ptr.Namespace != Namespace || ptr.UUID != "2026.07" || ptr.CID != res.CID {
		t.Fatalf("pointer = %+v", ptr)
	}
	blk, err := gs.GetBlock(ctx, res.CID)
	if err != nil {
		t.Fatalf("manifest block: %v", err)
	}
	want, _ := res.Manifest.CanonicalBytes()
	if string(blk) != string(want) {
		t.Fatal("stored manifest differs from built manifest")
	}
	if _, err := gs.GetBlock(ctx, termRef.CID); err != nil {
		t.Fatalf("term record version not published with the release: %v", err)
	}
	listed, _ := gs.ListNamespace(ctx, Namespace)
	if len(listed) != 1 || listed[0].CID != res.CID {
		t.Fatalf("releases namespace = %+v", listed)
	}

	// Same manifest, same label: no-op.
	if _, err := Publish(ctx, gs, e.ws, res, relAt.Add(time.Hour)); err != nil {
		t.Fatalf("republishing the identical release: %v", err)
	}
	again, _ := gs.GetNamespacedPointer(ctx, Namespace, "2026.07")
	if again.UpdatedAt != ptr.UpdatedAt {
		t.Fatal("identical republish rewrote the pointer")
	}
}

func TestPublishRefusesRelabelAndUnpromotedVersions(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	gs := memory.NewGlobalStore()

	a := promoted(t, e, gs, "First premise.", nil)
	b := promoted(t, e, gs, "Second premise.", nil)
	r1, _ := Build(ctx, e.ws, "v1", relAt, []string{a})
	if _, err := Publish(ctx, gs, e.ws, r1, relAt); err != nil {
		t.Fatalf("Publish r1: %v", err)
	}
	// A different manifest under the same label is refused; history is fixed.
	r1b, _ := Build(ctx, e.ws, "v1", relAt, []string{a, b})
	if _, err := Publish(ctx, gs, e.ws, r1b, relAt); !errors.Is(err, ErrLabelTaken) {
		t.Fatalf("expected ErrLabelTaken, got %v", err)
	}
	ptr, _ := gs.GetNamespacedPointer(ctx, Namespace, "v1")
	if ptr.CID != r1.CID {
		t.Fatal("published release label was changed")
	}

	// A premise attested locally but never promoted is not in the global
	// store, so a release tagging it cannot be published.
	local, localCID := e.premise("Attested locally, never promoted.")
	e.attest(local, localCID, model.ResultTrue, relAt.Add(-time.Hour), nil)
	r2, _ := Build(ctx, e.ws, "v2", relAt, []string{local.UUID})
	if len(r2.Manifest.Premises) != 1 {
		t.Fatalf("build should tag the locally valid premise: %+v", r2)
	}
	if _, err := Publish(ctx, gs, e.ws, r2, relAt); !errors.Is(err, ErrMissingVersion) {
		t.Fatalf("expected ErrMissingVersion, got %v", err)
	}
	if _, err := gs.GetNamespacedPointer(ctx, Namespace, "v2"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("refused release was listed: %v", err)
	}

	// A manifest altered after building no longer matches its CID.
	r3, _ := Build(ctx, e.ws, "v3", relAt, []string{a})
	r3.Manifest.Version = "v3-altered"
	if _, err := Publish(ctx, gs, e.ws, r3, relAt); err == nil {
		t.Fatal("altered manifest published")
	}
}
