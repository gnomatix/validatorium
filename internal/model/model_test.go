package model

import (
	"testing"
)

func TestNewAssignsUUIDAndCurrentState(t *testing.T) {
	m, err := New("Water can be a liquid.", Attribution{Agent: "key:abc"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if m.UUID == "" {
		t.Fatal("expected a UUID to be assigned")
	}
	if len(m.UUID) != 36 {
		t.Fatalf("expected 36-char UUIDv4, got %q", m.UUID)
	}
	if m.State.Status != Current {
		t.Fatalf("expected new record to be Current, got %q", m.State.Status)
	}
	if m.Claim.Text != "Water can be a liquid." {
		t.Fatalf("claim text not set: %q", m.Claim.Text)
	}
}

func TestUUIDsAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		m, err := New("apples can be red", Attribution{Agent: "key:abc"})
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		if seen[m.UUID] {
			t.Fatalf("duplicate UUID generated: %s", m.UUID)
		}
		seen[m.UUID] = true
	}
}

func TestValidateRecord(t *testing.T) {
	// A well-formed premise passes structural checks.
	good, _ := New("The speed of light in a vacuum is 299,792,458 m/s.", Attribution{Agent: "key:abc"})
	if err := good.ValidateRecord(); err != nil {
		t.Fatalf("well-formed premise rejected: %v", err)
	}

	// An empty statement is not a premise — rejected at the gate.
	empty, _ := New("   ", Attribution{Agent: "key:abc"})
	if err := empty.ValidateRecord(); err == nil {
		t.Fatal("expected empty-statement premise to be rejected")
	}

	// A missing UUID is rejected.
	noID := &Micropublication{Claim: Statement{Text: "x"}, State: Lifecycle{Status: Current}}
	if err := noID.ValidateRecord(); err == nil {
		t.Fatal("expected missing-UUID record to be rejected")
	}

	// An unknown evidence kind is rejected.
	badEv, _ := New("DNA replication is semiconservative.", Attribution{Agent: "key:abc"})
	badEv.SupportGraph = []Evidence{{Kind: "hearsay", Reference: "doi:10.0/x"}}
	if err := badEv.ValidateRecord(); err == nil {
		t.Fatal("expected unknown evidence kind to be rejected")
	}
}

func TestDocumentRoundTrip(t *testing.T) {
	m, _ := New("Acetylsalicylic acid inhibits cyclooxygenase-1 in human platelets.", Attribution{
		Agent:  "key:abc",
		AtTime: "2026-07-14T00:00:00Z",
	})
	m.Claim.Terms = []TermGrounding{
		{SurfaceForm: "Acetylsalicylic acid", OntologyURI: "http://www.wikidata.org/entity/Q18216"},
		{SurfaceForm: "human platelets", OntologyURI: "http://purl.obolibrary.org/obo/CL_0000233"},
	}
	m.SupportGraph = []Evidence{
		{Kind: EvidenceData, Reference: "doi:10.1038/171737a0", Note: "primary measurement"},
		{Kind: EvidenceMethod, Reference: "doi:10.1111/j.1601-5223.1956.tb03010.x"},
	}

	doc, err := m.Document()
	if err != nil {
		t.Fatalf("Document: %v", err)
	}
	if doc["_id"] != m.UUID {
		t.Fatalf("Documents-store key must be the UUID: got %v want %v", doc["_id"], m.UUID)
	}

	back, err := FromDocument(doc)
	if err != nil {
		t.Fatalf("FromDocument: %v", err)
	}
	if back.UUID != m.UUID {
		t.Fatalf("UUID drift: %q != %q", back.UUID, m.UUID)
	}
	if back.Claim.Text != m.Claim.Text {
		t.Fatalf("claim text drift")
	}
	if len(back.Claim.Terms) != 2 {
		t.Fatalf("term grounding lost in round-trip: got %d", len(back.Claim.Terms))
	}
	if len(back.SupportGraph) != 2 || back.SupportGraph[0].Kind != EvidenceData {
		t.Fatalf("support graph lost in round-trip: %+v", back.SupportGraph)
	}
	if err := back.ValidateRecord(); err != nil {
		t.Fatalf("round-tripped record is no longer structurally valid: %v", err)
	}
}
