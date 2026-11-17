package model

import (
	"testing"
	"time"
)

func TestTemporalPolicyIntervals(t *testing.T) {
	start := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	// Open-ended interval starting at a given time.
	openEnded := &TemporalPolicy{ValidTime: &Interval{Start: start.Format(time.RFC3339)}}
	// Closed interval entirely in the past.
	closedPast := &TemporalPolicy{ValidTime: &Interval{Start: "1990-01-01T00:00:00Z", End: "2005-01-01T00:00:00Z"}}
	for _, p := range []*TemporalPolicy{openEnded, closedPast, nil} {
		if err := p.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	if openEnded.ActiveAt(start.Add(-time.Hour)) || !openEnded.ActiveAt(start.Add(time.Hour)) {
		t.Fatal("open-ended interval wrong")
	}
	if !closedPast.ActiveAt(time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)) || closedPast.ActiveAt(start) {
		t.Fatal("closed past interval wrong")
	}
	var none *TemporalPolicy
	if !none.ActiveAt(start) {
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
