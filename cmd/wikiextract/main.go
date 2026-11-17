// Command wikiextract streams a MediaWiki XML dump (plain or .bz2, including
// the multistream files) and writes one JSON object per article:
//
//	{"wiki":"enwiki","page_id":736,"title":"…","rev_id":125…,"text":"…","refs":[…]}
//
// Citations are kept: each <ref> becomes a "[ref:N]" marker in the text and an
// entry in refs with its URL, DOI, PMID, ISBN and title where present.
//
// Only main-namespace pages that are not redirects are written. The text is
// the article's wikitext with markup removed (see strip.go); it is input for
// decomposition workers, not a faithful rendering. Part of the Wikipedia
// ingestion feed (ingest/wikipedia); not part of the specification.
package main

import (
	"bufio"
	"compress/bzip2"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
)

type page struct {
	Title    string    `xml:"title"`
	NS       int       `xml:"ns"`
	ID       int64     `xml:"id"`
	Redirect *struct{} `xml:"redirect"`
	Revision struct {
		ID   int64  `xml:"id"`
		Text string `xml:"text"`
	} `xml:"revision"`
}

// titles, when not nil, limits extraction to these article titles.
var titles map[string]bool

type article struct {
	Wiki   string `json:"wiki"`
	PageID int64  `json:"page_id"`
	Title  string `json:"title"`
	RevID  int64  `json:"rev_id"`
	Text   string `json:"text"`
	Refs   []Ref  `json:"refs,omitempty"`
}

func main() {
	in := flag.String("in", "-", "dump file (.xml or .xml.bz2); '-' reads standard input")
	wiki := flag.String("wiki", "enwiki", "wiki identifier written to each record")
	max := flag.Int("max", 0, "stop after this many articles (0 = all)")
	minLen := flag.Int("min-chars", 200, "skip articles whose stripped text is shorter than this")
	titlesFile := flag.String("titles", "", "file of article titles, one per line; only these are extracted")
	index := flag.String("index", "", "multistream index (.txt or .txt.bz2) for -in; with it the bz2 streams are decoded in parallel")
	workers := flag.Int("workers", runtime.NumCPU(), "parallel stream decoders when -index is given")
	flag.Parse()
	if *titlesFile != "" {
		b, err := os.ReadFile(*titlesFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "wikiextract:", err)
			os.Exit(1)
		}
		titles = map[string]bool{}
		for _, t := range strings.Split(string(b), "\n") {
			if t = strings.TrimSpace(strings.ReplaceAll(t, "_", " ")); t != "" {
				titles[t] = true
			}
		}
	}
	var n int
	var err error
	if *index != "" {
		n, err = runParallel(*in, *index, *wiki, *max, *minLen, *workers, os.Stdout)
	} else {
		n, err = run(*in, *wiki, *max, *minLen, os.Stdout)
	}
	fmt.Fprintf(os.Stderr, "wikiextract: wrote %d articles\n", n)
	if err != nil {
		fmt.Fprintln(os.Stderr, "wikiextract:", err)
		os.Exit(1)
	}
}

func run(path, wiki string, max, minLen int, out io.Writer) (int, error) {
	var r io.Reader = os.Stdin
	if path != "-" {
		f, err := os.Open(path)
		if err != nil {
			return 0, err
		}
		defer f.Close()
		r = f
	}
	br := bufio.NewReaderSize(r, 1<<20)
	if strings.HasSuffix(path, ".bz2") {
		r = bzip2.NewReader(br) // reads concatenated (multistream) bz2 streams in sequence
	} else {
		r = br
	}
	return extract(r, wiki, max, minLen, out)
}

func extract(r io.Reader, wiki string, max, minLen int, out io.Writer) (int, error) {
	dec := xml.NewDecoder(r)
	w := bufio.NewWriterSize(out, 1<<20)
	defer w.Flush()
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	n := 0
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return n, nil
		}
		if err != nil {
			return n, err
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "page" {
			continue
		}
		var p page
		if err := dec.DecodeElement(&p, &se); err != nil {
			return n, err
		}
		a := convert(&p, wiki, minLen)
		if a == nil {
			continue
		}
		if err := enc.Encode(a); err != nil {
			return n, err
		}
		n++
		if max > 0 && n >= max {
			return n, nil
		}
	}
}

// convert turns one dump page into an article, or nil when the page is not a
// main-namespace article, is a redirect, is not in the title list, or is
// shorter than minLen after stripping.
func convert(p *page, wiki string, minLen int) *article {
	if p.NS != 0 || p.Redirect != nil || (titles != nil && !titles[p.Title]) {
		return nil
	}
	marked, refs := extractRefs(p.Revision.Text)
	text := Strip(marked)
	if len(text) < minLen {
		return nil
	}
	return &article{Wiki: wiki, PageID: p.ID, Title: p.Title, RevID: p.Revision.ID, Text: text, Refs: refs}
}
