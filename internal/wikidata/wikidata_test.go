package wikidata

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gnomatix/validatorium/internal/evaluate"
	"gnomatix/validatorium/internal/model"
	"gnomatix/validatorium/internal/storage"
)

// Two revisions of a small entity, in Special:EntityData shape.
const rev1 = `{"entities":{"Q7186":{"id":"Q7186","lastrevid":100,"modified":"2026-01-01T00:00:00Z",
 "labels":{"en":{"language":"en","value":"Marie Curie"}},
 "descriptions":{"en":{"language":"en","value":"physicist and chemist"}},
 "claims":{
  "P166":[{"rank":"normal","mainsnak":{"snaktype":"value","property":"P166","datatype":"wikibase-item","datavalue":{"type":"wikibase-entityid","value":{"entity-type":"item","id":"Q38104"}}}},
          {"rank":"deprecated","mainsnak":{"snaktype":"value","property":"P166","datatype":"wikibase-item","datavalue":{"type":"wikibase-entityid","value":{"entity-type":"item","id":"Q999"}}}}],
  "P569":[{"rank":"normal","mainsnak":{"snaktype":"value","property":"P569","datatype":"time","datavalue":{"type":"time","value":{"time":"+1867-11-07T00:00:00Z","precision":11}}}}],
  "P1477":[{"rank":"normal","mainsnak":{"snaktype":"value","property":"P1477","datatype":"monolingualtext","datavalue":{"type":"monolingualtext","value":{"text":"Maria Salomea Skłodowska","language":"pl"}}}}],
  "P570":[{"rank":"normal","mainsnak":{"snaktype":"somevalue","property":"P570","datatype":"time"}}]
 }}}}`

const rev2 = `{"entities":{"Q7186":{"id":"Q7186","lastrevid":200,"modified":"2026-02-01T00:00:00Z",
 "labels":{"en":{"language":"en","value":"Marie Curie"}},
 "descriptions":{"en":{"language":"en","value":"physicist and chemist"}},
 "claims":{}}}}`

func fakeWiki(t *testing.T, bodies map[string]string) (*Client, *int) {
	t.Helper()
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if !strings.HasPrefix(r.URL.Path, "/wiki/Special:EntityData/") || r.Header.Get("User-Agent") == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		key := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/wiki/Special:EntityData/"), ".json") + "@" + r.URL.Query().Get("revision")
		body, ok := bodies[key]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return &Client{BaseURL: srv.URL}, &hits
}

func TestFetchParsesAndNormalizes(t *testing.T) {
	c, _ := fakeWiki(t, map[string]string{"Q7186@100": rev1, "Q7186@": rev2})
	ctx := context.Background()
	e, raw, err := c.Fetch(ctx, "Q7186", 100)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if e.ID != "Q7186" || e.Revision != 100 || e.Labels["en"] != "Marie Curie" || string(raw) != rev1 {
		t.Fatalf("entity = %+v", e)
	}
	want := map[string]bool{
		"P166=Q38104/normal":                       true,
		"P166=Q999/deprecated":                     true,
		"P569=+1867-11-07T00:00:00Z/11/normal":     true,
		"P1477=Maria Salomea Skłodowska@pl/normal": true,
		"P570=<somevalue>/normal":                  true,
	}
	for _, s := range e.Statements {
		k := s.Property + "=" + s.Value + "/" + s.Rank
		if !want[k] {
			t.Errorf("unexpected statement %s", k)
		}
		delete(want, k)
	}
	if len(want) != 0 {
		t.Errorf("missing statements: %v", want)
	}
	if !e.Holds("P166", "Q38104") || e.Holds("P166", "Q999") {
		t.Fatal("Holds must accept normal rank and ignore deprecated rank")
	}

	latest, _, err := c.Fetch(ctx, "Q7186", 0)
	if err != nil || latest.Revision != 200 {
		t.Fatalf("latest revision: %+v %v", latest, err)
	}
	if _, _, err := c.Fetch(ctx, "Q7186", 150); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing revision: expected ErrNotFound, got %v", err)
	}
	if _, _, err := c.Fetch(ctx, "not-an-id", 0); err == nil {
		t.Fatal("malformed id accepted")
	}
}

func TestSnapshotIsAPointerWithEquivalenceCheck(t *testing.T) {
	bodies := map[string]string{"Q7186@100": rev1}
	c, _ := fakeWiki(t, bodies)
	ctx := context.Background()
	at := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)

	snap, err := c.TakeSnapshot(ctx, "Q7186", 100, at)
	if err != nil {
		t.Fatalf("TakeSnapshot: %v", err)
	}
	wantCID, _ := storage.ContentID([]byte(rev1))
	if snap.RawCID != wantCID || !strings.HasSuffix(snap.SourceURI, "/wiki/Special:EntityData/Q7186.json?revision=100") {
		t.Fatalf("snapshot pointer = %+v", snap)
	}
	if ok, err := c.Equivalent(ctx, snap); err != nil || !ok {
		t.Fatalf("unchanged revision not equivalent: %v %v", ok, err)
	}
	// If what the source serves for that revision differs from the recorded
	// structured form, equivalence fails.
	bodies["Q7186@100"] = strings.Replace(rev1, `"Q38104"`, `"Q11111"`, 1)
	if ok, err := c.Equivalent(ctx, snap); err != nil || ok {
		t.Fatalf("changed content reported equivalent: %v %v", ok, err)
	}
	// Canonical bytes are stable for content addressing.
	a, _ := snap.CanonicalBytes()
	b, _ := snap.CanonicalBytes()
	if string(a) != string(b) {
		t.Fatal("snapshot canonical bytes not stable")
	}
}

func TestTermRecordPinsRevision(t *testing.T) {
	e, err := ParseEntity([]byte(rev1), "Q7186")
	if err != nil {
		t.Fatal(err)
	}
	tr, err := TermRecord(e, "en", model.Attribution{Agent: "gemini-cli"})
	if err != nil {
		t.Fatalf("TermRecord: %v", err)
	}
	if tr.Label != "Marie Curie" || tr.OntologyURI != "http://www.wikidata.org/entity/Q7186" || tr.SourceVersion != "wikidata:Q7186@100" || tr.Definition != "physicist and chemist" {
		t.Fatalf("term record = %+v", tr)
	}
	if _, err := TermRecord(e, "de", model.Attribution{Agent: "x"}); err == nil {
		t.Fatal("missing label language accepted")
	}
}

func TestStatementEvaluator(t *testing.T) {
	c, _ := fakeWiki(t, map[string]string{"Q7186@100": rev1})
	reg, err := evaluate.NewRegistry(StatementEvaluator{Client: c})
	if err != nil {
		t.Fatal(err)
	}
	prem, _ := model.New("Wikidata Q7186 at revision 100 holds P166 = Q38104.", model.Attribution{Agent: "gemini-cli"})
	prem.SupportGraph = []model.Evidence{
		{Kind: model.EvidenceMethod, Reference: StatementMethod},
		{Kind: model.EvidenceData, Reference: c.SourceURI("Q7186", 100)},
	}
	ctx := context.Background()
	at := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	scope := func(prop, val string, rev interface{}) map[string]interface{} {
		return map[string]interface{}{"entity": "Q7186", "revision": rev, "property": prop, "value": val}
	}

	att, _, err := reg.Attest(ctx, prem, scope("P166", "Q38104", float64(100)), "gemini-cli", at)
	if err != nil || att.Result != model.ResultTrue {
		t.Fatalf("held statement: %v %v", att, err)
	}
	rawCID, _ := storage.ContentID([]byte(rev1))
	if len(att.EvidenceVersions) != 2 || att.EvidenceVersions[1] != rawCID {
		t.Fatalf("evidence = %v, want the read bytes' CID", att.EvidenceVersions)
	}
	if att, _, _ := reg.Attest(ctx, prem, scope("P166", "Q999", "100"), "gemini-cli", at); att.Result != model.ResultFalse {
		t.Fatalf("deprecated statement: got %s, want false", att.Result)
	}
	if att, _, _ := reg.Attest(ctx, prem, scope("P166", "Q38104", 150), "gemini-cli", at); att.Result != model.ResultIndeterminate {
		t.Fatalf("unreadable revision: got %s, want indeterminate", att.Result)
	}
	if _, _, err := reg.Attest(ctx, prem, scope("P166", "Q38104", nil), "gemini-cli", at); err == nil {
		t.Fatal("evaluation without a pinned revision accepted")
	}
}
