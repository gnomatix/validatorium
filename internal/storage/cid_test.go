package storage

import (
	"strings"
	"testing"
)

func TestContentIDIsStableRawSHA256CIDv1(t *testing.T) {
	data := []byte("hello validatorium")
	id, err := ContentID(data)
	if err != nil {
		t.Fatalf("ContentID: %v", err)
	}
	// CIDv1, raw codec, sha2-256, base32: the "bafkrei" prefix is fixed by
	// the multibase/version/codec/hash bytes.
	if !strings.HasPrefix(id, "bafkrei") {
		t.Fatalf("expected a base32 CIDv1 raw/sha2-256 identifier, got %q", id)
	}
	again, _ := ContentID(data)
	if again != id {
		t.Fatalf("ContentID not deterministic: %s != %s", again, id)
	}
	other, _ := ContentID([]byte("hello validatorium!"))
	if other == id {
		t.Fatal("different bytes produced the same CID")
	}
}

func TestContentIDMatchesKnownVector(t *testing.T) {
	// `ipfs block put --cid-codec raw` of the empty input yields this CID.
	id, err := ContentID(nil)
	if err != nil {
		t.Fatalf("ContentID: %v", err)
	}
	const want = "bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku"
	if id != want {
		t.Fatalf("empty-input CID = %s, want %s", id, want)
	}
}

func TestVerifyContentID(t *testing.T) {
	data := []byte("x")
	id, _ := ContentID(data)
	if err := VerifyContentID(id, data); err != nil {
		t.Fatalf("verify matching content: %v", err)
	}
	if err := VerifyContentID(id, []byte("y")); err == nil {
		t.Fatal("expected mismatch for different bytes")
	}
	if err := VerifyContentID("not-a-cid", data); err == nil {
		t.Fatal("expected error for undecodable cid")
	}
}
