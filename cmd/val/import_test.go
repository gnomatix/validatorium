package main

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gnomatix/validatorium/internal/bbolt"
)

const sampleJSONL = `{"source":{"wiki":"enwiki","page_id":736,"title":"Albert Einstein","rev_id":1250000001,"url":"https://en.wikipedia.org/w/index.php?oldid=1250000001"},"local_id":"p1","kind":"premise","statement":"Albert Einstein was born on 14 March 1879.","terms":[{"surface_form":"Albert Einstein","ontology_uri":"http://www.wikidata.org/entity/Q937"}],"quote":"Albert Einstein (14 March 1879 – 18 April 1955)"}
{"source":{"wiki":"enwiki","page_id":736,"title":"Albert Einstein","rev_id":1250000001,"url":"https://en.wikipedia.org/w/index.php?oldid=1250000001"},"local_id":"p2","kind":"premise","statement":"Albert Einstein was born in Ulm.","depends_on":["p1"],"quote":"born in Ulm","citations":[{"doi":"10.1000/xyz","title":"Einstein"}]}
{"source":{"wiki":"enwiki","page_id":736,"title":"Albert Einstein","rev_id":1250000001,"url":"https://en.wikipedia.org/w/index.php?oldid=1250000001"},"local_id":"c1","kind":"conclusion","statement":"Albert Einstein was German-born."}
not json
{"source":{"wiki":"enwiki","rev_id":0},"local_id":"x","kind":"premise","statement":"no revision"}
`

func TestImportJSONL(t *testing.T) {
	ctx := context.Background()
	st, err := bbolt.OpenWorkingStore(filepath.Join(t.TempDir(), "w.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	var rejects bytes.Buffer
	at := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	stats, err := importJSONL(ctx, st, strings.NewReader(sampleJSONL), "worker:test", at, &rejects, nil)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Created != 2 || stats.Conclusions != 1 || stats.Rejected != 2 {
		t.Fatalf("stats %+v, rejects:\n%s", stats, rejects.String())
	}
	ids, _ := st.ListCandidates(ctx)
	if len(ids) != 2 {
		t.Fatalf("candidates %v", ids)
	}
	var withDep int
	for _, id := range ids {
		c, err := st.GetCandidate(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if c.SupportGraph[0].Reference != "https://en.wikipedia.org/w/index.php?oldid=1250000001" {
			t.Fatalf("source pointer missing: %+v", c.SupportGraph)
		}
		for _, e := range c.SupportGraph {
			if strings.HasPrefix(e.Reference, premiseRefPrefix) {
				withDep++
			}
		}
	}
	var cited int
	for _, id := range ids {
		c, _ := st.GetCandidate(ctx, id)
		for _, e := range c.SupportGraph {
			if e.Reference == "https://doi.org/10.1000/xyz" {
				cited++
			}
		}
	}
	if cited != 1 {
		t.Fatalf("citation pointers: %d", cited)
	}
	if withDep != 1 {
		t.Fatalf("dependency links: %d", withDep)
	}
	// Re-running the same file creates nothing new.
	again, err := importJSONL(ctx, st, strings.NewReader(sampleJSONL), "worker:test", at, &rejects, nil)
	if err != nil || again.Created != 0 || again.Existing != 2 {
		t.Fatalf("re-import: %+v %v", again, err)
	}
	// A second worker decomposing the same article keeps its own records.
	other, err := importJSONL(ctx, st, strings.NewReader(sampleJSONL), "worker:other", at, &rejects, nil)
	if err != nil || other.Created != 2 {
		t.Fatalf("second worker: %+v %v", other, err)
	}
}

func TestImportQuoteCheck(t *testing.T) {
	ctx := context.Background()
	st, err := bbolt.OpenWorkingStore(filepath.Join(t.TempDir(), "w.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	article := `{"wiki":"enwiki","page_id":736,"title":"Albert Einstein","rev_id":1250000001,"text":"Albert Einstein (14 March 1879 – 18 April 1955) was a physicist. [ref:1]\nHe was born   in Ulm."}`
	articles, err := loadArticles(strings.NewReader(article))
	if err != nil {
		t.Fatal(err)
	}
	src := `{"wiki":"enwiki","page_id":736,"title":"Albert Einstein","rev_id":1250000001,"url":"https://en.wikipedia.org/w/index.php?oldid=1250000001"}`
	line := func(id, statement, quote string) string {
		return fmt.Sprintf(`{"source":%s,"local_id":%q,"kind":"premise","statement":%q,"quote":%q}`, src, id, statement, quote)
	}
	in := strings.Join([]string{
		line("p1", "Albert Einstein was a physicist.", "was a physicist."),
		line("p2", "Albert Einstein was born in Ulm.", "He was born in Ulm."),
		line("p3", "Albert Einstein was a chemist.", "was a chemist"),
		`{"source":{"wiki":"enwiki","page_id":1,"rev_id":99,"url":"x"},"local_id":"p1","kind":"premise","statement":"Unrelated.","quote":"Unrelated."}`,
	}, "\n")
	var rejects bytes.Buffer
	stats, err := importJSONL(ctx, st, strings.NewReader(in), "worker:test", time.Now(), &rejects, articles)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Created != 2 || stats.QuoteMismatch != 1 || stats.Rejected != 2 {
		t.Fatalf("stats %+v\n%s", stats, rejects.String())
	}
}
