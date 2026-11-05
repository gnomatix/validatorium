package promote

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"gnomatix/validatorium/internal/bbolt"
	"gnomatix/validatorium/internal/model"
	"gnomatix/validatorium/internal/storage"
	"gnomatix/validatorium/internal/storage/memory"
)

var fixedNow = time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)

type fixture struct {
	ws   *bbolt.WorkingStore
	gs   *memory.GlobalStore
	p    *Promoter
	cand *model.Micropublication
	cid  string
}

func setup(t *testing.T) fixture {
	t.Helper()
	ws, err := bbolt.OpenWorkingStore(filepath.Join(t.TempDir(), "working.db"))
	if err != nil {
		t.Fatalf("open working store: %v", err)
	}
	t.Cleanup(func() { ws.Close() })
	gs := memory.NewGlobalStore()

	cand, _ := model.New("Pure water boils at 100 °C at 1 atm.", model.Attribution{Agent: "key:abc", AtTime: "2026-07-14T00:00:00Z"})
	cand.Claim.Terms = []model.TermGrounding{{SurfaceForm: "water", OntologyURI: "http://www.wikidata.org/entity/Q283"}}
	cand.SupportGraph = []model.Evidence{{Kind: model.EvidenceData, Reference: "doi:10.1000/example"}}
	if err := ws.PutCandidate(context.Background(), cand); err != nil {
		t.Fatalf("put candidate: %v", err)
	}
	data, _ := cand.CanonicalBytes()
	cid, _ := storage.ContentID(data)
	return fixture{ws: ws, gs: gs, p: &Promoter{Working: ws, Global: gs, Now: func() time.Time { return fixedNow }}, cand: cand, cid: cid}
}

func (f fixture) attestation(t *testing.T, result model.AttestationResult) *model.ValidityAttestation {
	t.Helper()
	a, err := model.NewAttestation(f.cand.UUID, f.cid, result, "evaluator:test", "ipfs://method", fixedNow.Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	a.EvidenceVersions = []string{"doi:10.1000/example"}
	return a
}

// assertUnchanged checks that a refused promotion left both stores untouched.
func (f fixture) assertUnchanged(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	if _, err := f.ws.GetCandidate(ctx, f.cand.UUID); err != nil {
		t.Fatalf("draft removed after refusal: %v", err)
	}
	if _, err := f.ws.GetCurrent(ctx, f.cand.UUID); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("current pointer set after refusal: %v", err)
	}
	if _, err := f.ws.GetVersion(ctx, f.cid); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("version stored after refusal: %v", err)
	}
	if atts, _ := f.ws.ListAttestations(ctx, f.cand.UUID); len(atts) != 0 {
		t.Fatalf("attestation recorded after refusal: %d", len(atts))
	}
	if f.gs.BlockCount() != 0 {
		t.Fatalf("global blocks written after refusal: %d", f.gs.BlockCount())
	}
	if _, err := f.gs.GetPointer(ctx, f.cand.UUID); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("global pointer written after refusal: %v", err)
	}
}

func TestPromoteAdmitsTrueAttestationForExactVersion(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	att := f.attestation(t, model.ResultTrue)

	res, err := f.p.Promote(ctx, f.cand.UUID, att)
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if res.Version.UUID != f.cand.UUID || res.Version.CID != f.cid {
		t.Fatalf("promoted version = %+v, want %s@%s", res.Version, f.cand.UUID, f.cid)
	}

	// Tier 1: version stored, pointer set, attestation logged, draft removed.
	v, err := f.ws.GetVersion(ctx, f.cid)
	if err != nil || v.Claim.Text != f.cand.Claim.Text {
		t.Fatalf("local version missing or wrong: %+v %v", v, err)
	}
	cur, err := f.ws.GetCurrent(ctx, f.cand.UUID)
	if err != nil || cur.CID != f.cid {
		t.Fatalf("local current pointer wrong: %+v %v", cur, err)
	}
	atts, _ := f.ws.ListAttestations(ctx, f.cand.UUID)
	if len(atts) != 1 || atts[0].UUID != att.UUID {
		t.Fatalf("attestation log wrong: %+v", atts)
	}
	if _, err := f.ws.GetCandidate(ctx, f.cand.UUID); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("draft should be removed after promotion, got %v", err)
	}

	// Tier 2: version and attestation blocks addressable by CID; pointer names both.
	blk, err := f.gs.GetBlock(ctx, f.cid)
	if err != nil {
		t.Fatalf("global version block: %v", err)
	}
	back, _ := model.MicropublicationFromBytes(blk)
	if back.UUID != f.cand.UUID {
		t.Fatalf("global version block decodes to %s", back.UUID)
	}
	attBlk, err := f.gs.GetBlock(ctx, res.AttestationCID)
	if err != nil {
		t.Fatalf("global attestation block: %v", err)
	}
	attBack, _ := model.AttestationFromBytes(attBlk)
	if attBack.UUID != att.UUID || attBack.PremiseVersionCID != f.cid {
		t.Fatalf("global attestation drift: %+v", attBack)
	}
	ptr, err := f.gs.GetPointer(ctx, f.cand.UUID)
	if err != nil {
		t.Fatalf("global pointer: %v", err)
	}
	if ptr.CID != f.cid || ptr.AttestationCID != res.AttestationCID || ptr.Status != model.Current {
		t.Fatalf("global pointer wrong: %+v", ptr)
	}
	if ptr.UpdatedAt != fixedNow.Format(time.RFC3339Nano) {
		t.Fatalf("pointer UpdatedAt = %q", ptr.UpdatedAt)
	}
}

func TestPromoteRefusesFalseAndIndeterminate(t *testing.T) {
	for _, r := range []model.AttestationResult{model.ResultFalse, model.ResultIndeterminate} {
		f := setup(t)
		_, err := f.p.Promote(context.Background(), f.cand.UUID, f.attestation(t, r))
		if !errors.Is(err, ErrNotValid) {
			t.Fatalf("result %q: expected ErrNotValid, got %v", r, err)
		}
		f.assertUnchanged(t)
	}
}

func TestPromoteRefusesExpiredAttestation(t *testing.T) {
	f := setup(t)
	att := f.attestation(t, model.ResultTrue)
	att.ExpiresAt = fixedNow.Add(-time.Second).Format(time.RFC3339Nano)
	_, err := f.p.Promote(context.Background(), f.cand.UUID, att)
	if !errors.Is(err, ErrNotValid) {
		t.Fatalf("expected ErrNotValid for expired attestation, got %v", err)
	}
	f.assertUnchanged(t)

	// The same attestation is accepted while still in force.
	att.ExpiresAt = fixedNow.Add(time.Hour).Format(time.RFC3339Nano)
	if _, err := f.p.Promote(context.Background(), f.cand.UUID, att); err != nil {
		t.Fatalf("unexpired attestation refused: %v", err)
	}
}

func TestPromoteRefusesAttestationForOtherVersionOrPremise(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	// Attestation for a different version of the same premise (draft edited
	// after evaluation).
	att := f.attestation(t, model.ResultTrue)
	f.cand.Claim.Text = "Pure water boils at 100 °C at 101.325 kPa."
	if err := f.ws.PutCandidate(ctx, f.cand); err != nil {
		t.Fatal(err)
	}
	_, err := f.p.Promote(ctx, f.cand.UUID, att)
	if !errors.Is(err, ErrVersionMismatch) {
		t.Fatalf("expected ErrVersionMismatch after draft edit, got %v", err)
	}
	if f.gs.BlockCount() != 0 {
		t.Fatal("global store written despite version mismatch")
	}

	// Attestation for a different premise UUID.
	data, _ := f.cand.CanonicalBytes()
	newCID, _ := storage.ContentID(data)
	other, _ := model.NewAttestation("some-other-premise", newCID, model.ResultTrue, "e", "m", fixedNow)
	_, err = f.p.Promote(ctx, f.cand.UUID, other)
	if !errors.Is(err, ErrVersionMismatch) {
		t.Fatalf("expected ErrVersionMismatch for foreign premise UUID, got %v", err)
	}
}

func TestPromoteRefusesObsoleteCandidateAndMalformedInput(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	f.cand.State = model.Lifecycle{Status: model.Obsolete, SupersededBy: []string{"successor"}}
	_ = f.ws.PutCandidate(ctx, f.cand)
	data, _ := f.cand.CanonicalBytes()
	cid, _ := storage.ContentID(data)
	att, _ := model.NewAttestation(f.cand.UUID, cid, model.ResultTrue, "e", "m", fixedNow)
	if _, err := f.p.Promote(ctx, f.cand.UUID, att); !errors.Is(err, ErrNotCurrent) {
		t.Fatalf("expected ErrNotCurrent, got %v", err)
	}

	if _, err := f.p.Promote(ctx, "unknown-uuid", att); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for unknown candidate, got %v", err)
	}
	if _, err := f.p.Promote(ctx, f.cand.UUID, nil); err == nil {
		t.Fatal("nil attestation accepted")
	}
	bad := *att
	bad.Method = ""
	if _, err := f.p.Promote(ctx, f.cand.UUID, &bad); err == nil {
		t.Fatal("malformed attestation accepted")
	}
	if f.gs.BlockCount() != 0 {
		t.Fatal("global store written despite refusals")
	}
}

func TestRepromotionOfRefinedVersionKeepsHistory(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	if _, err := f.p.Promote(ctx, f.cand.UUID, f.attestation(t, model.ResultTrue)); err != nil {
		t.Fatalf("first promotion: %v", err)
	}

	// Refine the statement under the same UUID, evaluate again, promote again.
	v2 := *f.cand
	v2.Claim.Text = "Pure water boils at 100 °C at 101.325 kPa."
	_ = f.ws.PutCandidate(ctx, &v2)
	data, _ := v2.CanonicalBytes()
	cid2, _ := storage.ContentID(data)
	att2, _ := model.NewAttestation(v2.UUID, cid2, model.ResultTrue, "evaluator:test", "ipfs://method", fixedNow)
	res, err := f.p.Promote(ctx, v2.UUID, att2)
	if err != nil {
		t.Fatalf("second promotion: %v", err)
	}
	if res.Version.CID != cid2 || cid2 == f.cid {
		t.Fatalf("second version CID = %s (first %s)", res.Version.CID, f.cid)
	}
	ptr, _ := f.gs.GetPointer(ctx, v2.UUID)
	if ptr.CID != cid2 {
		t.Fatalf("pointer did not advance: %+v", ptr)
	}
	// Both versions and both attestations remain addressable.
	if _, err := f.gs.GetBlock(ctx, f.cid); err != nil {
		t.Fatalf("first version block gone: %v", err)
	}
	if _, err := f.ws.GetVersion(ctx, f.cid); err != nil {
		t.Fatalf("first local version gone: %v", err)
	}
	atts, _ := f.ws.ListAttestations(ctx, v2.UUID)
	if len(atts) != 2 {
		t.Fatalf("expected 2 attestations in history, got %d", len(atts))
	}
}
