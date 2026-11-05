package model

import (
	"strings"
	"testing"
	"time"
)

func TestNewAttestationIsWellFormed(t *testing.T) {
	at := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	a, err := NewAttestation("premise-uuid", "bafkreiabc", ResultTrue, "evaluator:key:abc", "ipfs://method", at)
	if err != nil {
		t.Fatalf("NewAttestation: %v", err)
	}
	if err := a.ValidateRecord(); err != nil {
		t.Fatalf("new attestation rejected: %v", err)
	}
	if len(a.UUID) != 36 {
		t.Fatalf("expected UUIDv4, got %q", a.UUID)
	}
	if a.EvaluatedAt != "2026-07-14T12:00:00Z" {
		t.Fatalf("evaluatedAt not RFC 3339 UTC: %q", a.EvaluatedAt)
	}
}

func TestAttestationValidateRecordRejectsMalformed(t *testing.T) {
	at := time.Now()
	base := func() *ValidityAttestation {
		a, _ := NewAttestation("p", "cid", ResultFalse, "e", "m", at)
		return a
	}
	cases := map[string]func(*ValidityAttestation){
		"missing uuid":        func(a *ValidityAttestation) { a.UUID = "" },
		"missing premise":     func(a *ValidityAttestation) { a.PremiseUUID = " " },
		"missing version cid": func(a *ValidityAttestation) { a.PremiseVersionCID = "" },
		"bad result":          func(a *ValidityAttestation) { a.Result = "probably" },
		"bad evaluatedAt":     func(a *ValidityAttestation) { a.EvaluatedAt = "yesterday" },
		"missing evaluator":   func(a *ValidityAttestation) { a.Evaluator = "" },
		"missing method":      func(a *ValidityAttestation) { a.Method = "" },
		"bad expiresAt":       func(a *ValidityAttestation) { a.ExpiresAt = "soon" },
	}
	for name, mutate := range cases {
		a := base()
		mutate(a)
		if err := a.ValidateRecord(); err == nil {
			t.Errorf("%s: expected rejection", name)
		}
	}
}

func TestEstablishesValidityAt(t *testing.T) {
	at := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	mk := func(r AttestationResult, expires string) *ValidityAttestation {
		a, _ := NewAttestation("p", "cid", r, "e", "m", at)
		a.ExpiresAt = expires
		return a
	}
	// Only a true result establishes validity.
	if mk(ResultFalse, "").EstablishesValidityAt(at) {
		t.Error("false result must not establish validity")
	}
	if mk(ResultIndeterminate, "").EstablishesValidityAt(at) {
		t.Error("indeterminate result must not establish validity")
	}
	if !mk(ResultTrue, "").EstablishesValidityAt(at.Add(1000 * time.Hour)) {
		t.Error("unexpiring true result must establish validity")
	}
	// Expiry ends validity without touching the premise.
	exp := at.Add(time.Hour).Format(time.RFC3339Nano)
	if !mk(ResultTrue, exp).EstablishesValidityAt(at.Add(30 * time.Minute)) {
		t.Error("true result before expiry must establish validity")
	}
	if mk(ResultTrue, exp).EstablishesValidityAt(at.Add(2 * time.Hour)) {
		t.Error("true result after expiry must not establish validity")
	}
	if mk(ResultTrue, "garbage").EstablishesValidityAt(at) {
		t.Error("unparseable expiry must not establish validity")
	}
}

func TestCanonicalBytesAreDeterministicAndVersionSensitive(t *testing.T) {
	m, _ := New("Pure water boils at 100 °C at 1 atm.", Attribution{Agent: "key:abc", AtTime: "2026-07-14T00:00:00Z"})
	m.Claim.Terms = []TermGrounding{{SurfaceForm: "water", OntologyURI: "http://www.wikidata.org/entity/Q283"}}
	m.SupportGraph = []Evidence{{Kind: EvidenceData, Reference: "doi:10.1000/example"}}

	b1, err := m.CanonicalBytes()
	if err != nil {
		t.Fatalf("CanonicalBytes: %v", err)
	}
	b2, _ := m.CanonicalBytes()
	if string(b1) != string(b2) {
		t.Fatal("canonical bytes differ between two encodings of the same record")
	}
	back, err := MicropublicationFromBytes(b1)
	if err != nil {
		t.Fatalf("MicropublicationFromBytes: %v", err)
	}
	b3, _ := back.CanonicalBytes()
	if string(b3) != string(b1) {
		t.Fatal("canonical bytes changed after a decode/encode round trip")
	}
	if !strings.Contains(string(b1), `"_id":"`+m.UUID+`"`) {
		t.Fatalf("canonical bytes do not carry the UUID: %s", b1)
	}

	// A lifecycle change is a new version.
	m.State = Lifecycle{Status: Obsolete, SupersededBy: []string{"successor"}}
	b4, _ := m.CanonicalBytes()
	if string(b4) == string(b1) {
		t.Fatal("lifecycle change did not change the canonical bytes")
	}

	a, _ := NewAttestation(m.UUID, "cid", ResultTrue, "e", "m", time.Unix(0, 0))
	ab, err := a.CanonicalBytes()
	if err != nil {
		t.Fatalf("attestation CanonicalBytes: %v", err)
	}
	aback, err := AttestationFromBytes(ab)
	if err != nil {
		t.Fatalf("AttestationFromBytes: %v", err)
	}
	if aback.UUID != a.UUID || aback.Result != ResultTrue {
		t.Fatalf("attestation round trip drift: %+v", aback)
	}
}
