package model

import (
	"errors"
	"testing"
)

func TestTermRecordValidateRecord(t *testing.T) {
	good, err := NewTermRecord("water", "http://www.wikidata.org/entity/Q283", Attribution{Agent: "key:abc"})
	if err != nil {
		t.Fatalf("NewTermRecord: %v", err)
	}
	if err := good.ValidateRecord(); err != nil {
		t.Fatalf("well-formed term record rejected: %v", err)
	}

	noLabel := *good
	noLabel.Label = ""
	if err := noLabel.ValidateRecord(); err == nil {
		t.Error("missing label accepted")
	}
	noURI := *good
	noURI.OntologyURI = " "
	if err := noURI.ValidateRecord(); err == nil {
		t.Error("missing ontology URI accepted")
	}
	currentWithReplacement := *good
	currentWithReplacement.State = Lifecycle{Status: Current, SupersededBy: []string{"x"}}
	if err := currentWithReplacement.ValidateRecord(); err == nil {
		t.Error("current record naming replacements accepted")
	}
	obsoleteNoReplacement := *good
	obsoleteNoReplacement.State = Lifecycle{Status: Obsolete}
	if err := obsoleteNoReplacement.ValidateRecord(); err == nil {
		t.Error("obsolete record without replacements accepted")
	}
	selfReplace := *good
	selfReplace.State = Lifecycle{Status: Obsolete, SupersededBy: []string{good.UUID}}
	if err := selfReplace.ValidateRecord(); err == nil {
		t.Error("self-replacement accepted")
	}
	badStatus := *good
	badStatus.State = Lifecycle{Status: "retired"}
	if err := badStatus.ValidateRecord(); err == nil {
		t.Error("unknown status accepted")
	}

	b, err := good.CanonicalBytes()
	if err != nil {
		t.Fatalf("CanonicalBytes: %v", err)
	}
	back, err := TermRecordFromBytes(b)
	if err != nil {
		t.Fatalf("TermRecordFromBytes: %v", err)
	}
	if back.UUID != good.UUID || back.OntologyURI != good.OntologyURI {
		t.Fatalf("term record round-trip drift: %+v", back)
	}
}

func termGraph(t *testing.T) (map[string]*TermRecord, TermLookup) {
	t.Helper()
	recs := map[string]*TermRecord{}
	add := func(id string, status LifecycleStatus, replacements ...string) {
		recs[id] = &TermRecord{
			UUID:        id,
			Label:       id,
			OntologyURI: "http://example.org/" + id,
			State:       Lifecycle{Status: status, SupersededBy: replacements},
		}
	}
	// t1 was split into t2 and t3; t3 was later refined into t4.
	add("t1", Obsolete, "t2", "t3")
	add("t2", Current)
	add("t3", Obsolete, "t4")
	add("t4", Current)
	// t5 and t6 both resolve to t4 (merge).
	add("t5", Obsolete, "t4")
	add("t6", Obsolete, "t4", "t5")
	// Cycle: c1 -> c2 -> c1.
	add("c1", Obsolete, "c2")
	add("c2", Obsolete, "c1")
	// Dangling pointer.
	add("d1", Obsolete, "missing")
	lookup := func(id string) (*TermRecord, error) {
		r, ok := recs[id]
		if !ok {
			return nil, errors.New("not found")
		}
		return r, nil
	}
	return recs, lookup
}

func ids(recs []*TermRecord) []string {
	out := make([]string, len(recs))
	for i, r := range recs {
		out[i] = r.UUID
	}
	return out
}

func TestResolveTermReplacements(t *testing.T) {
	recs, lookup := termGraph(t)

	cur, err := ResolveTermReplacements(lookup, "t2")
	if err != nil || len(cur) != 1 || cur[0].UUID != "t2" {
		t.Fatalf("current record must resolve to itself: %v %v", ids(cur), err)
	}

	split, err := ResolveTermReplacements(lookup, "t1")
	if err != nil {
		t.Fatalf("resolve t1: %v", err)
	}
	if got := ids(split); len(got) != 2 || got[0] != "t2" || got[1] != "t4" {
		t.Fatalf("t1 should resolve to [t2 t4], got %v", got)
	}
	// Resolution did not touch the obsolete records.
	if recs["t1"].State.Status != Obsolete || len(recs["t1"].State.SupersededBy) != 2 {
		t.Fatal("obsolete record was altered by resolution")
	}

	merged, err := ResolveTermReplacements(lookup, "t6")
	if err != nil {
		t.Fatalf("resolve t6: %v", err)
	}
	if got := ids(merged); len(got) != 1 || got[0] != "t4" {
		t.Fatalf("t6 should resolve to [t4] once, got %v", got)
	}

	if _, err := ResolveTermReplacements(lookup, "c1"); !errors.Is(err, ErrReplacementCycle) {
		t.Fatalf("expected ErrReplacementCycle, got %v", err)
	}
	if _, err := ResolveTermReplacements(lookup, "d1"); err == nil {
		t.Fatal("dangling replacement pointer resolved without error")
	}
	if _, err := ResolveTermReplacements(lookup, "nope"); err == nil {
		t.Fatal("unknown starting record resolved without error")
	}
}
