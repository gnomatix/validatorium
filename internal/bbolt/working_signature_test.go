package bbolt

import (
	"context"
	"path/filepath"
	"testing"

	"gnomatix/validatorium/internal/model"
)

func TestSignatureIndexes(t *testing.T) {
	ctx := context.Background()
	w, err := OpenWorkingStore(filepath.Join(t.TempDir(), "w.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	armored := "-----BEGIN PGP SIGNATURE-----\n\nx\n-----END PGP SIGNATURE-----\n"
	a := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	b := "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"
	sigs := []*model.Signature{
		{SubjectCID: "cid1", Signer: a, Action: "promote", ArmoredSignature: armored, SignedAt: "2026-11-16T12:00:00Z"},
		{SubjectCID: "cid2", Signer: a, Action: "promote", ArmoredSignature: armored, SignedAt: "2026-11-16T12:00:00Z"},
		// A second signer on the same version: kept separately, no merging.
		{SubjectCID: "cid1", Signer: b, Action: "promote", ArmoredSignature: armored, SignedAt: "2026-11-16T12:00:00Z"},
	}
	for _, s := range sigs {
		if _, err := w.PutSignature(ctx, s); err != nil {
			t.Fatal(err)
		}
	}
	byA, err := w.SignaturesBySigner(ctx, a)
	if err != nil || len(byA) != 2 {
		t.Fatalf("by signer a: %d %v", len(byA), err)
	}
	on1, err := w.SignaturesForSubject(ctx, "cid1")
	if err != nil || len(on1) != 2 {
		t.Fatalf("on cid1: %d %v", len(on1), err)
	}
	if none, _ := w.SignaturesBySigner(ctx, "CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC"); len(none) != 0 {
		t.Fatalf("unknown signer: %d", len(none))
	}
}
