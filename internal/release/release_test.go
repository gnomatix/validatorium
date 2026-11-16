package release

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gnomatix/validatorium/internal/bbolt"
	"gnomatix/validatorium/internal/model"
	"gnomatix/validatorium/internal/storage"
)

var relAt = time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)

type env struct {
	t  *testing.T
	ws *bbolt.WorkingStore
}

func newEnv(t *testing.T) env {
	t.Helper()
	ws, err := bbolt.OpenWorkingStore(filepath.Join(t.TempDir(), "w.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ws.Close() })
	return env{t: t, ws: ws}
}

// premise stores a current version of a new premise and returns it with its CID.
func (e env) premise(text string) (*model.Micropublication, string) {
	e.t.Helper()
	m, _ := model.New(text, model.Attribution{Agent: "key:abc"})
	ref, err := e.ws.PutVersion(context.Background(), m)
	if err != nil {
		e.t.Fatal(err)
	}
	if err := e.ws.SetCurrent(context.Background(), ref); err != nil {
		e.t.Fatal(err)
	}
	return m, ref.CID
}

func (e env) attest(m *model.Micropublication, cid string, r model.AttestationResult, evaluated time.Time, mutate func(*model.ValidityAttestation)) *model.ValidityAttestation {
	e.t.Helper()
	a, _ := model.NewAttestation(m.UUID, cid, r, "evaluator:test", "ipfs://method", evaluated)
	if mutate != nil {
		mutate(a)
	}
	if err := e.ws.PutAttestation(context.Background(), a); err != nil {
		e.t.Fatal(err)
	}
	return a
}

func reasons(res Result) map[string]string {
	out := map[string]string{}
	for _, x := range res.Exclusions {
		out[x.UUID] = x.Reason
	}
	return out
}

func TestBuildTagsOnlyEligiblePremises(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	before := relAt.Add(-time.Hour)

	valid, validCID := e.premise("Pure water boils at 100 °C at 1 atm.")
	goodAtt := e.attest(valid, validCID, model.ResultTrue, before, nil)

	falseP, falseCID := e.premise("The Moon is made of cheese.")
	e.attest(falseP, falseCID, model.ResultFalse, before, nil)

	expired, expiredCID := e.premise("It is currently July.")
	e.attest(expired, expiredCID, model.ResultTrue, before, func(a *model.ValidityAttestation) {
		a.ExpiresAt = relAt.Add(-time.Minute).Format(time.RFC3339Nano)
	})

	future, futureCID := e.premise("Evaluated after the release time.")
	e.attest(future, futureCID, model.ResultTrue, relAt.Add(time.Hour), nil)

	stale, staleCID := e.premise("Attested at an earlier version.")
	e.attest(stale, staleCID, model.ResultTrue, before, nil)
	stale.Claim.Text = "Attested at an earlier version, then refined."
	ref2, _ := e.ws.PutVersion(ctx, stale)
	_ = e.ws.SetCurrent(ctx, ref2)

	obsolete, _ := model.New("Superseded premise.", model.Attribution{Agent: "key:abc"})
	obsolete.State = model.Lifecycle{Status: model.Obsolete, SupersededBy: []string{valid.UUID}}
	oref, _ := e.ws.PutVersion(ctx, obsolete)
	_ = e.ws.SetCurrent(ctx, oref)
	e.attest(obsolete, oref.CID, model.ResultTrue, before, nil)

	cands := []string{valid.UUID, falseP.UUID, expired.UUID, future.UUID, stale.UUID, obsolete.UUID, "missing-uuid", valid.UUID}
	res, err := Build(ctx, e.ws, "2026.07", relAt, cands)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(res.Manifest.Premises) != 1 || res.Manifest.Premises[0].UUID != valid.UUID || res.Manifest.Premises[0].CID != validCID {
		t.Fatalf("tagged premises = %+v, want only %s@%s", res.Manifest.Premises, valid.UUID, validCID)
	}
	if len(res.Manifest.Attestations) != 1 || res.Manifest.Attestations[0] != goodAtt.UUID {
		t.Fatalf("attestations = %v, want [%s]", res.Manifest.Attestations, goodAtt.UUID)
	}
	r := reasons(res)
	want := map[string]string{
		falseP.UUID:    "no true, unexpired attestation for the current version",
		expired.UUID:   "no true, unexpired attestation for the current version",
		future.UUID:    "no true, unexpired attestation for the current version",
		stale.UUID:     "no true, unexpired attestation for the current version",
		obsolete.UUID:  "premise is obsolete",
		"missing-uuid": "no current version",
	}
	for id, w := range want {
		if r[id] != w {
			t.Errorf("exclusion for %s = %q, want %q", id, r[id], w)
		}
	}
	if len(res.Exclusions) != len(want) {
		t.Errorf("got %d exclusions, want %d: %+v", len(res.Exclusions), len(want), res.Exclusions)
	}
}

func TestBuildIsDeterministicAndContentAddressed(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	var ids []string
	for _, s := range []string{"A premise.", "B premise.", "C premise."} {
		m, cid := e.premise(s)
		e.attest(m, cid, model.ResultTrue, relAt.Add(-time.Hour), nil)
		ids = append(ids, m.UUID)
	}
	r1, err := Build(ctx, e.ws, "v1", relAt, ids)
	if err != nil {
		t.Fatal(err)
	}
	// Candidates in a different order must give the same manifest.
	rev := []string{ids[2], ids[1], ids[0]}
	r2, err := Build(ctx, e.ws, "v1", relAt, rev)
	if err != nil {
		t.Fatal(err)
	}
	if r1.CID != r2.CID {
		t.Fatalf("candidate order changed the manifest CID: %s vs %s", r1.CID, r2.CID)
	}
	if len(r1.Manifest.Premises) != 3 {
		t.Fatalf("expected 3 tagged premises, got %d", len(r1.Manifest.Premises))
	}
	for i := 1; i < len(r1.Manifest.Premises); i++ {
		if r1.Manifest.Premises[i-1].UUID >= r1.Manifest.Premises[i].UUID {
			t.Fatal("premises not in UUID order")
		}
	}
	data, _ := r1.Manifest.CanonicalBytes()
	if err := storage.VerifyContentID(r1.CID, data); err != nil {
		t.Fatalf("CID does not match canonical bytes: %v", err)
	}
	r3, _ := Build(ctx, e.ws, "v2", relAt, ids)
	if r3.CID == r1.CID {
		t.Fatal("a different release label produced the same CID")
	}
	if _, err := Build(ctx, e.ws, " ", relAt, nil); err == nil {
		t.Fatal("empty release label accepted")
	}
}

func TestBuildTermAndDependencyRules(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	before := relAt.Add(-time.Hour)

	liveTerm, _ := model.NewTermRecord("water", "http://www.wikidata.org/entity/Q283", model.Attribution{Agent: "key:abc"})
	if _, err := e.ws.PutTermRecord(ctx, liveTerm); err != nil {
		t.Fatal(err)
	}
	oldTerm, _ := model.NewTermRecord("ice", "http://www.wikidata.org/entity/Q23392", model.Attribution{Agent: "key:abc"})
	_, _ = e.ws.PutTermRecord(ctx, oldTerm)
	oldTerm.State = model.Lifecycle{Status: model.Obsolete, SupersededBy: []string{liveTerm.UUID}}
	_, _ = e.ws.PutTermRecord(ctx, oldTerm)

	base, baseCID := e.premise("Base premise.")
	e.attest(base, baseCID, model.ResultTrue, before, func(a *model.ValidityAttestation) {
		a.TermRecordUUIDs = []string{liveTerm.UUID}
	})
	derived, derivedCID := e.premise("Derived from the base premise.")
	e.attest(derived, derivedCID, model.ResultTrue, before, func(a *model.ValidityAttestation) {
		a.Dependencies = []string{base.UUID, baseCID}
	})
	usesOld, usesOldCID := e.premise("Grounded on an obsolete term.")
	e.attest(usesOld, usesOldCID, model.ResultTrue, before, func(a *model.ValidityAttestation) {
		a.TermRecordUUIDs = []string{oldTerm.UUID}
	})
	chained, chainedCID := e.premise("Depends on the obsolete-term premise.")
	e.attest(chained, chainedCID, model.ResultTrue, before, func(a *model.ValidityAttestation) {
		a.Dependencies = []string{usesOld.UUID}
	})

	// With the base included, the derived premise is tagged; the obsolete
	// term excludes usesOld, which in turn strands chained.
	res, err := Build(ctx, e.ws, "r1", relAt, []string{chained.UUID, derived.UUID, usesOld.UUID, base.UUID})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	tagged := map[string]bool{}
	for _, p := range res.Manifest.Premises {
		tagged[p.UUID] = true
	}
	if !tagged[base.UUID] || !tagged[derived.UUID] || len(tagged) != 2 {
		t.Fatalf("tagged = %v, want base and derived only", tagged)
	}
	r := reasons(res)
	if r[usesOld.UUID] != "term record "+oldTerm.UUID+" is obsolete" {
		t.Errorf("usesOld exclusion = %q", r[usesOld.UUID])
	}
	if r[chained.UUID] != "depends on "+usesOld.UUID+", which is not in this release" {
		t.Errorf("chained exclusion = %q", r[chained.UUID])
	}
	liveRef, _ := e.ws.PutTermRecord(ctx, liveTerm) // identical bytes: returns the stored version
	if len(res.Manifest.Terms) != 1 || res.Manifest.Terms[0] != liveRef {
		t.Fatalf("terms = %+v, want the live term's exact version %+v", res.Manifest.Terms, liveRef)
	}

	// Without the base as a candidate, the derived premise is excluded too.
	res2, _ := Build(ctx, e.ws, "r2", relAt, []string{derived.UUID})
	if len(res2.Manifest.Premises) != 0 || reasons(res2)[derived.UUID] != "depends on "+base.UUID+", which is not in this release" {
		t.Fatalf("derived without base: premises %+v exclusions %+v", res2.Manifest.Premises, res2.Exclusions)
	}
}
