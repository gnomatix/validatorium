package main

import (
	"bufio"
	"bytes"
	"compress/bzip2"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// streamOffsets reads a multistream index ("offset:page_id:title" per line,
// plain or bz2) and returns the distinct stream start offsets in order.
func streamOffsets(path string) ([]int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var r io.Reader = bufio.NewReaderSize(f, 1<<20)
	if strings.HasSuffix(path, ".bz2") {
		r = bzip2.NewReader(r)
	}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<16), 1<<20)
	seen := map[int64]bool{}
	var offs []int64
	for sc.Scan() {
		line := sc.Text()
		i := strings.IndexByte(line, ':')
		if i <= 0 {
			continue
		}
		o, err := strconv.ParseInt(line[:i], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("index line %q: %v", line, err)
		}
		if !seen[o] {
			seen[o] = true
			offs = append(offs, o)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	sort.Slice(offs, func(i, j int) bool { return offs[i] < offs[j] })
	return offs, nil
}

// pagesInStream decodes one bz2 stream of a multistream dump and returns its
// articles in dump order. A stream holds a run of <page> elements; the last
// stream also holds the closing </mediawiki> tag, which ends decoding.
func pagesInStream(f *os.File, start, end int64, wiki string, minLen int) ([]*article, error) {
	dec := xml.NewDecoder(bzip2.NewReader(bufio.NewReaderSize(io.NewSectionReader(f, start, end-start), 1<<20)))
	var out []*article
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			var se *xml.SyntaxError
			if errors.As(err, &se) && strings.Contains(se.Msg, "</mediawiki>") {
				return out, nil
			}
			return out, fmt.Errorf("stream at byte %d: %w", start, err)
		}
		el, ok := tok.(xml.StartElement)
		if !ok || el.Name.Local != "page" {
			continue
		}
		var p page
		if err := dec.DecodeElement(&p, &el); err != nil {
			return out, fmt.Errorf("stream at byte %d: %w", start, err)
		}
		if a := convert(&p, wiki, minLen); a != nil {
			out = append(out, a)
		}
	}
}

// runParallel decodes the streams named by the index with the given number
// of workers and writes the articles in dump order, the same output as run.
func runParallel(path, index, wiki string, max, minLen, workers int, out io.Writer) (int, error) {
	if workers < 1 {
		workers = 1
	}
	offs, err := streamOffsets(index)
	if err != nil {
		return 0, err
	}
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return 0, err
	}
	type result struct {
		arts []*article
		err  error
	}
	results := make([]chan result, len(offs))
	for i := range results {
		results[i] = make(chan result, 1)
	}
	jobs := make(chan int)
	done := make(chan struct{})
	// slots bounds how many decoded streams wait in memory for the writer.
	slots := make(chan struct{}, 4*workers)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				end := st.Size()
				if i+1 < len(offs) {
					end = offs[i+1]
				}
				arts, err := pagesInStream(f, offs[i], end, wiki, minLen)
				results[i] <- result{arts, err}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for i := range offs {
			select {
			case slots <- struct{}{}:
			case <-done:
				return
			}
			select {
			case jobs <- i:
			case <-done:
				return
			}
		}
	}()
	defer func() { close(done); wg.Wait() }()

	w := bufio.NewWriterSize(out, 1<<20)
	defer w.Flush()
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	n := 0
	for i := range offs {
		r := <-results[i]
		<-slots
		if r.err != nil {
			return n, r.err
		}
		for _, a := range r.arts {
			buf.Reset()
			if err := enc.Encode(a); err != nil {
				return n, err
			}
			if _, err := w.Write(buf.Bytes()); err != nil {
				return n, err
			}
			n++
			if max > 0 && n >= max {
				return n, nil
			}
		}
	}
	return n, nil
}
