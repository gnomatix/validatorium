package bbolt_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"gnomatix/validatorium/internal/bbolt"
	"gnomatix/validatorium/internal/model"
	"gnomatix/validatorium/internal/storage"
)

func openWorking(t testing.TB) *bbolt.WorkingStore {
	t.Helper()
	ws, err := bbolt.OpenWorkingStore(filepath.Join(t.TempDir(), "working.db"))
	if err != nil {
		t.Fatalf("OpenWorkingStore: %v", err)
	}
	t.Cleanup(func() { ws.Close() })
	return ws
}

func samplePremise(t testing.TB, text string) *model.Micropublication {
	t.Helper()
	m, err := model.New(text, model.Attribution{Agent: "key:abc", AtTime: "2026-07-14T00:00:00Z"})
	if err != nil {
		t.Fatalf("model.New: %v", err)
	}
	m.SupportGraph = []model.Evidence{{Kind: model.EvidenceData, Reference: "doi:10.1000/example"}}
	return m
}

func TestCandidatesAreMutableDrafts(t *testing.T) {
	ws := openWorking(t)
	ctx := context.Background()

	if _, err := ws.GetCandidate(ctx, "missing"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	m := samplePremise(t, "The normal human diploid genome contains 46 chromosomes.")
	if err := ws.PutCandidate(ctx, m); err != nil {
		t.Fatalf("PutCandidate: %v", err)
	}
	got, err := ws.GetCandidate(ctx, m.UUID)
	if err != nil {
		t.Fatalf("GetCandidate: %v", err)
	}
	if got.Claim.Text != m.Claim.Text || len(got.SupportGraph) != 1 {
		t.Fatalf("candidate round trip drift: %+v", got)
	}

	// Drafts may be refined under the same UUID.
	m.Claim.Text = "The normal human diploid genome contains 46 chromosomes (23 pairs)."
	if err := ws.PutCandidate(ctx, m); err != nil {
		t.Fatalf("PutCandidate (update): %v", err)
	}
	got, _ = ws.GetCandidate(ctx, m.UUID)
	if got.Claim.Text != m.Claim.Text {
		t.Fatalf("candidate update not applied: %q", got.Claim.Text)
	}

	m2 := samplePremise(t, "Pure water boils at 100 °C at 1 atm.")
	_ = ws.PutCandidate(ctx, m2)
	ids, err := ws.ListCandidates(ctx)
	if err != nil {
		t.Fatalf("ListCandidates: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 candidates, got %v", ids)
	}
	if err := ws.DeleteCandidate(ctx, m.UUID); err != nil {
		t.Fatalf("DeleteCandidate: %v", err)
	}
	if err := ws.DeleteCandidate(ctx, m.UUID); err != nil {
		t.Fatalf("DeleteCandidate (absent) must not error: %v", err)
	}
	ids, _ = ws.ListCandidates(ctx)
	if len(ids) != 1 || ids[0] != m2.UUID {
		t.Fatalf("expected only %s to remain, got %v", m2.UUID, ids)
	}
}

func TestVersionsAreImmutableAndContentAddressed(t *testing.T) {
	ws := openWorking(t)
	ctx := context.Background()
	m := samplePremise(t, "DNA replication is semiconservative.")

	ref, err := ws.PutVersion(ctx, m)
	if err != nil {
		t.Fatalf("PutVersion: %v", err)
	}
	data, _ := m.CanonicalBytes()
	want, _ := storage.ContentID(data)
	if ref.UUID != m.UUID || ref.CID != want {
		t.Fatalf("version ref = %+v, want uuid %s cid %s", ref, m.UUID, want)
	}

	// Same bytes again: idempotent, same CID.
	ref2, err := ws.PutVersion(ctx, m)
	if err != nil || ref2 != ref {
		t.Fatalf("re-putting identical version: ref %+v err %v", ref2, err)
	}

	// A refined statement under the same UUID is a new version with a new CID,
	// and the old version is still retrievable unchanged.
	m.Claim.Text = "DNA replication is semiconservative (Meselson–Stahl, 1958)."
	ref3, err := ws.PutVersion(ctx, m)
	if err != nil {
		t.Fatalf("PutVersion v2: %v", err)
	}
	if ref3.CID == ref.CID {
		t.Fatal("changed record produced the same CID")
	}
	v1, err := ws.GetVersion(ctx, ref.CID)
	if err != nil {
		t.Fatalf("GetVersion v1: %v", err)
	}
	if v1.Claim.Text != "DNA replication is semiconservative." {
		t.Fatalf("v1 was altered: %q", v1.Claim.Text)
	}
	if _, err := ws.GetVersion(ctx, "bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for unknown version, got %v", err)
	}
}

func TestCurrentPointerRequiresStoredVersion(t *testing.T) {
	ws := openWorking(t)
	ctx := context.Background()
	m := samplePremise(t, "It is currently July.")

	if _, err := ws.GetCurrent(ctx, m.UUID); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound before any pointer, got %v", err)
	}
	err := ws.SetCurrent(ctx, storage.VersionRef{UUID: m.UUID, CID: "bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku"})
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("pointer to an unstored version must be refused with ErrNotFound, got %v", err)
	}
	ref, _ := ws.PutVersion(ctx, m)
	if err := ws.SetCurrent(ctx, ref); err != nil {
		t.Fatalf("SetCurrent: %v", err)
	}
	cur, err := ws.GetCurrent(ctx, m.UUID)
	if err != nil || cur != ref {
		t.Fatalf("GetCurrent = %+v, %v; want %+v", cur, err, ref)
	}
	// Moving the pointer to a later version keeps the earlier version.
	m.Claim.Text = "It is currently July (UTC)."
	ref2, _ := ws.PutVersion(ctx, m)
	_ = ws.SetCurrent(ctx, ref2)
	cur, _ = ws.GetCurrent(ctx, m.UUID)
	if cur.CID != ref2.CID {
		t.Fatalf("pointer not advanced: %+v", cur)
	}
	if _, err := ws.GetVersion(ctx, ref.CID); err != nil {
		t.Fatalf("earlier version lost after pointer move: %v", err)
	}
}

func TestAttestationsAppendOnlyAndIndexedByPremise(t *testing.T) {
	ws := openWorking(t)
	ctx := context.Background()
	at := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)

	a1, _ := model.NewAttestation("premise-A", "cid-1", model.ResultTrue, "evaluator", "method", at)
	a2, _ := model.NewAttestation("premise-A", "cid-1", model.ResultFalse, "evaluator", "method", at.Add(time.Hour))
	b1, _ := model.NewAttestation("premise-B", "cid-9", model.ResultIndeterminate, "evaluator", "method", at)
	for _, a := range []*model.ValidityAttestation{a1, a2, b1} {
		if err := ws.PutAttestation(ctx, a); err != nil {
			t.Fatalf("PutAttestation %s: %v", a.UUID, err)
		}
	}

	// Re-writing the identical record is a no-op; changing it is refused.
	if err := ws.PutAttestation(ctx, a1); err != nil {
		t.Fatalf("identical re-put must succeed: %v", err)
	}
	tampered := *a1
	tampered.Result = model.ResultFalse
	if err := ws.PutAttestation(ctx, &tampered); !errors.Is(err, storage.ErrImmutable) {
		t.Fatalf("expected ErrImmutable when altering an attestation, got %v", err)
	}
	got, err := ws.GetAttestation(ctx, a1.UUID)
	if err != nil || got.Result != model.ResultTrue {
		t.Fatalf("original attestation altered or lost: %+v %v", got, err)
	}

	listA, err := ws.ListAttestations(ctx, "premise-A")
	if err != nil {
		t.Fatalf("ListAttestations: %v", err)
	}
	if len(listA) != 2 {
		t.Fatalf("expected 2 attestations for premise-A, got %d", len(listA))
	}
	seen := map[string]bool{}
	for _, a := range listA {
		if a.PremiseUUID != "premise-A" {
			t.Fatalf("index returned attestation for %s", a.PremiseUUID)
		}
		seen[a.UUID] = true
	}
	if !seen[a1.UUID] || !seen[a2.UUID] {
		t.Fatalf("index missing attestations: %v", seen)
	}
	listB, _ := ws.ListAttestations(ctx, "premise-B")
	if len(listB) != 1 || listB[0].UUID != b1.UUID {
		t.Fatalf("premise-B list wrong: %+v", listB)
	}
	// A premise UUID that is a prefix of another must not match its index entries.
	listPrefix, _ := ws.ListAttestations(ctx, "premise")
	if len(listPrefix) != 0 {
		t.Fatalf("prefix UUID matched %d foreign attestations", len(listPrefix))
	}
	if _, err := ws.GetAttestation(ctx, "nope"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	// Malformed attestations are refused at the store boundary.
	bad := *a1
	bad.UUID = "other"
	bad.Result = "maybe"
	if err := ws.PutAttestation(ctx, &bad); err == nil {
		t.Fatal("malformed attestation accepted")
	}
}

func TestWorkingStorePersistsAcrossReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "working.db")
	ws, err := bbolt.OpenWorkingStore(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	ctx := context.Background()
	m := samplePremise(t, "Acetylsalicylic acid inhibits cyclooxygenase-1.")
	_ = ws.PutCandidate(ctx, m)
	ref, _ := ws.PutVersion(ctx, m)
	_ = ws.SetCurrent(ctx, ref)
	if err := ws.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	ws2, err := bbolt.OpenWorkingStore(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer ws2.Close()
	if _, err := ws2.GetCandidate(ctx, m.UUID); err != nil {
		t.Fatalf("candidate lost on reopen: %v", err)
	}
	cur, err := ws2.GetCurrent(ctx, m.UUID)
	if err != nil || cur != ref {
		t.Fatalf("pointer lost on reopen: %+v %v", cur, err)
	}
	if _, err := ws2.GetVersion(ctx, ref.CID); err != nil {
		t.Fatalf("version lost on reopen: %v", err)
	}
}

func BenchmarkPutCandidate(b *testing.B) {
	ws := openWorking(b)
	ctx := context.Background()
	m := samplePremise(b, "Benchmark premise.")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Claim.Text = fmt.Sprintf("Benchmark premise %d.", i)
		if err := ws.PutCandidate(ctx, m); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetCandidate(b *testing.B) {
	ws := openWorking(b)
	ctx := context.Background()
	m := samplePremise(b, "Benchmark premise.")
	_ = ws.PutCandidate(ctx, m)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ws.GetCandidate(ctx, m.UUID); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPutVersion(b *testing.B) {
	ws := openWorking(b)
	ctx := context.Background()
	m := samplePremise(b, "Benchmark premise.")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Claim.Text = fmt.Sprintf("Benchmark premise %d.", i)
		if _, err := ws.PutVersion(ctx, m); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkListAttestations(b *testing.B) {
	ws := openWorking(b)
	ctx := context.Background()
	at := time.Now()
	for i := 0; i < 100; i++ {
		a, _ := model.NewAttestation("p", "cid", model.ResultTrue, "e", "m", at)
		_ = ws.PutAttestation(ctx, a)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ws.ListAttestations(ctx, "p"); err != nil {
			b.Fatal(err)
		}
	}
}
