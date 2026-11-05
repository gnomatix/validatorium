package bbolt_test

import (
	"context"
	"errors"
	"testing"

	"gnomatix/validatorium/internal/model"
	"gnomatix/validatorium/internal/storage"
)

func TestTermRecordsAreVersionedAndResolvable(t *testing.T) {
	ws := openWorking(t)
	ctx := context.Background()
	attr := model.Attribution{Agent: "key:abc", AtTime: "2026-07-14T00:00:00Z"}

	if _, err := ws.GetTermRecord(ctx, "missing"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	// Version 1 of a mapping for "water".
	water, _ := model.NewTermRecord("water", "http://www.wikidata.org/entity/Q283", attr)
	water.SourceVersion = "wikidata:rev:1"
	ref1, err := ws.PutTermRecord(ctx, water)
	if err != nil {
		t.Fatalf("PutTermRecord: %v", err)
	}
	got, err := ws.GetTermRecord(ctx, water.UUID)
	if err != nil || got.SourceVersion != "wikidata:rev:1" {
		t.Fatalf("GetTermRecord: %+v %v", got, err)
	}
	cur, err := ws.ResolveTerm(ctx, water.UUID)
	if err != nil || len(cur) != 1 || cur[0].UUID != water.UUID {
		t.Fatalf("current term must resolve to itself: %v %v", cur, err)
	}

	// The mapping is superseded by two more specific records (a split).
	liquid, _ := model.NewTermRecord("liquid water", "http://www.wikidata.org/entity/Q29053", attr)
	ice, _ := model.NewTermRecord("ice", "http://www.wikidata.org/entity/Q23392", attr)
	for _, r := range []*model.TermRecord{liquid, ice} {
		if _, err := ws.PutTermRecord(ctx, r); err != nil {
			t.Fatalf("PutTermRecord %s: %v", r.Label, err)
		}
	}
	water.State = model.Lifecycle{Status: model.Obsolete, SupersededBy: []string{liquid.UUID, ice.UUID}}
	ref2, err := ws.PutTermRecord(ctx, water)
	if err != nil {
		t.Fatalf("PutTermRecord (obsolete): %v", err)
	}
	if ref2.CID == ref1.CID {
		t.Fatal("obsoleting the record did not produce a new version")
	}

	// The current pointer names the obsolete version; the first version is
	// still addressable by CID and unchanged.
	got, _ = ws.GetTermRecord(ctx, water.UUID)
	if got.State.Status != model.Obsolete || len(got.State.SupersededBy) != 2 {
		t.Fatalf("current term version not obsolete: %+v", got.State)
	}
	v1, err := ws.GetTermRecordVersion(ctx, ref1.CID)
	if err != nil {
		t.Fatalf("GetTermRecordVersion v1: %v", err)
	}
	if v1.State.Status != model.Current {
		t.Fatalf("historical term version was altered: %+v", v1.State)
	}

	// Resolution follows the replacement pointers to the current records.
	resolved, err := ws.ResolveTerm(ctx, water.UUID)
	if err != nil {
		t.Fatalf("ResolveTerm: %v", err)
	}
	if len(resolved) != 2 || resolved[0].UUID != liquid.UUID || resolved[1].UUID != ice.UUID {
		t.Fatalf("expected [liquid ice], got %+v", resolved)
	}

	// Malformed records are refused at the store boundary.
	bad := *liquid
	bad.State = model.Lifecycle{Status: model.Obsolete}
	if _, err := ws.PutTermRecord(ctx, &bad); err == nil {
		t.Fatal("obsolete term without replacements accepted")
	}
	if _, err := ws.GetTermRecordVersion(ctx, "bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for unknown term version, got %v", err)
	}
}
