package model

import (
	"testing"
	"time"
)

func TestTemporalPolicyJeffScopes(t *testing.T) {
	said := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	// "My favorite food is Pop-Tarts": holds as of the date stated, open-ended.
	asOf := &TemporalPolicy{ValidTime: &Interval{Start: said.Format(time.RFC3339)}}
	// "The kind they used to make": a closed period that has ended.
	usedTo := &TemporalPolicy{ValidTime: &Interval{Start: "1990-01-01T00:00:00Z", End: "2005-01-01T00:00:00Z"}}
	for _, p := range []*TemporalPolicy{asOf, usedTo, nil} {
		if err := p.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	if asOf.ActiveAt(said.Add(-time.Hour)) || !asOf.ActiveAt(said.Add(time.Hour)) {
		t.Fatal("as-of scope wrong")
	}
	if !usedTo.ActiveAt(time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)) || usedTo.ActiveAt(said) {
		t.Fatal("used-to scope wrong")
	}
	var none *TemporalPolicy
	if !none.ActiveAt(said) {
		t.Fatal("nil policy must be active")
	}
	bad := []*TemporalPolicy{
		{ValidTime: &Interval{Start: "yesterday"}},
		{ValidTime: &Interval{Start: "2005-01-01T00:00:00Z", End: "1990-01-01T00:00:00Z"}},
		{TTL: "-5h"},
		{Timezone: "Not/AZone"},
	}
	for _, b := range bad {
		if b.Validate() == nil {
			t.Fatalf("accepted %+v", b)
		}
	}
}

func TestTemporalPolicyKeepsExistingCIDs(t *testing.T) {
	m, _ := New("Iron has atomic number 26.", Attribution{Agent: "key:x"})
	before, _ := m.CanonicalBytes()
	m.Temporal = nil
	after, _ := m.CanonicalBytes()
	if string(before) != string(after) {
		t.Fatal("a nil temporal policy changed the canonical bytes")
	}
	m.Temporal = &TemporalPolicy{ValidTime: &Interval{Start: "yesterday"}}
	if m.ValidateRecord() == nil {
		t.Fatal("invalid temporal policy accepted by ValidateRecord")
	}
}
