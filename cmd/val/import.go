package main

import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"gnomatix/validatorium/internal/bbolt"
	"gnomatix/validatorium/internal/model"
	"gnomatix/validatorium/internal/storage"
)

// ImportSource identifies the exact source revision a premise was decomposed
// from. It is cited as a pointer to that revision, never as authority.
type ImportSource struct {
	Wiki   string `json:"wiki"`
	PageID int64  `json:"page_id"`
	Title  string `json:"title"`
	RevID  int64  `json:"rev_id"`
	URL    string `json:"url"`
}

// ImportTerm is one grounded term as the worker proposes it.
type ImportTerm struct {
	SurfaceForm string `json:"surface_form"`
	OntologyURI string `json:"ontology_uri"`
	Sense       string `json:"sense"`
}

// ImportRecord is one line of worker output (see ingest/wikipedia/WORKER_PROMPT.md).
type ImportRecord struct {
	Source    ImportSource           `json:"source"`
	LocalID   string                 `json:"local_id"`
	Kind      string                 `json:"kind"` // "premise" or "conclusion"
	Statement string                 `json:"statement"`
	Terms     []ImportTerm           `json:"terms"`
	Scope     map[string]interface{} `json:"scope"`
	Quote     string                 `json:"quote"`
	DependsOn []string               `json:"depends_on"`
}

// premiseRefPrefix marks a support-graph entry that names another premise.
const premiseRefPrefix = "urn:validatorium:premise:"

// importNamespace is the UUIDv5 namespace for imported premises, so the same
// source revision and local id always yield the same premise UUID and a
// re-run import never duplicates records.
var importNamespace = [16]byte{0x6b, 0x2f, 0x0d, 0x4e, 0x8a, 0x51, 0x4c, 0x7e, 0x9b, 0x13, 0x55, 0x0a, 0x21, 0xc4, 0x6e, 0x90}

func importUUID(r *ImportRecord) string {
	key := fmt.Sprintf("%s|%d|%d|%s", r.Source.Wiki, r.Source.PageID, r.Source.RevID, r.LocalID)
	h := sha1.New()
	h.Write(importNamespace[:])
	h.Write([]byte(key))
	s := h.Sum(nil)
	s[6] = (s[6] & 0x0f) | 0x50
	s[8] = (s[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", s[0:4], s[4:6], s[6:8], s[8:10], s[10:16])
}

func (r *ImportRecord) check() error {
	switch {
	case strings.TrimSpace(r.Statement) == "":
		return errors.New("empty statement")
	case r.Source.RevID <= 0 || strings.TrimSpace(r.Source.Wiki) == "":
		return errors.New("source must name the wiki and a revision id")
	case strings.TrimSpace(r.LocalID) == "":
		return errors.New("missing local_id")
	case r.Kind != "premise" && r.Kind != "conclusion":
		return fmt.Errorf("kind %q is neither premise nor conclusion", r.Kind)
	}
	return nil
}

// ImportStats counts what one import did.
type ImportStats struct {
	Created, Existing, Conclusions, Rejected int
}

// importJSONL loads worker output into the working store as candidate
// premises. Conclusions are counted and not stored as premises
// (docs/specs/premise-decomposition.md §3.3). Malformed lines are rejected
// and reported; they do not stop the import.
func importJSONL(ctx context.Context, st *bbolt.WorkingStore, in io.Reader, agent string, at time.Time, rejects io.Writer) (ImportStats, error) {
	var stats ImportStats
	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 0, 1<<20), 16<<20)
	var batch []ImportRecord
	flush := func() error {
		// Resolve depends_on within the same source revision.
		ids := map[string]string{}
		for i := range batch {
			ids[batch[i].LocalID] = importUUID(&batch[i])
		}
		for i := range batch {
			r := &batch[i]
			if r.Kind == "conclusion" {
				stats.Conclusions++
				continue
			}
			id := ids[r.LocalID]
			if _, err := st.GetCandidate(ctx, id); err == nil {
				stats.Existing++
				continue
			} else if !errors.Is(err, storage.ErrNotFound) {
				return err
			}
			m, err := model.New(r.Statement, model.Attribution{Agent: agent, AtTime: at.UTC().Format(time.RFC3339)})
			if err != nil {
				return err
			}
			m.UUID = id
			for _, t := range r.Terms {
				if strings.TrimSpace(t.SurfaceForm) == "" {
					continue
				}
				m.Claim.Terms = append(m.Claim.Terms, model.TermGrounding{SurfaceForm: t.SurfaceForm, OntologyURI: t.OntologyURI, Sense: t.Sense})
			}
			m.SupportGraph = append(m.SupportGraph, model.Evidence{Kind: model.EvidenceData, Reference: r.Source.URL, Note: r.Quote})
			for _, dep := range r.DependsOn {
				if did, ok := ids[dep]; ok {
					m.SupportGraph = append(m.SupportGraph, model.Evidence{Kind: model.EvidenceData, Reference: premiseRefPrefix + did, Note: "depends on premise " + dep})
				}
			}
			if err := st.PutCandidate(ctx, m); err != nil {
				fmt.Fprintf(rejects, "%s/%d/%s: %v\n", r.Source.Wiki, r.Source.RevID, r.LocalID, err)
				stats.Rejected++
				continue
			}
			stats.Created++
		}
		batch = batch[:0]
		return nil
	}
	line := 0
	for sc.Scan() {
		line++
		text := strings.TrimSpace(sc.Text())
		if text == "" {
			continue
		}
		var r ImportRecord
		if err := json.Unmarshal([]byte(text), &r); err != nil {
			fmt.Fprintf(rejects, "line %d: invalid JSON: %v\n", line, err)
			stats.Rejected++
			continue
		}
		if err := r.check(); err != nil {
			fmt.Fprintf(rejects, "line %d: %v\n", line, err)
			stats.Rejected++
			continue
		}
		if len(batch) > 0 && (batch[0].Source.Wiki != r.Source.Wiki || batch[0].Source.RevID != r.Source.RevID) {
			if err := flush(); err != nil {
				return stats, err
			}
		}
		batch = append(batch, r)
	}
	if err := sc.Err(); err != nil {
		return stats, err
	}
	return stats, flush()
}

// cmdImport loads a JSONL file of decomposed premises (worker output) into
// the working store as candidates. Candidates are drafts: they are not valid
// until evaluated, and nothing is published by importing.
func cmdImport(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	ws := openWS(fs)
	file := fs.String("file", "", "JSONL file of worker output; '-' reads standard input (required)")
	agent := fs.String("agent", "", "agent recorded on each candidate, e.g. the worker identity (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *file == "" || *agent == "" {
		return errors.New("import: -file and -agent are required")
	}
	in := os.Stdin
	if *file != "-" {
		f, err := os.Open(*file)
		if err != nil {
			return err
		}
		defer f.Close()
		in = f
	}
	st, err := bbolt.OpenWorkingStore(*ws)
	if err != nil {
		return err
	}
	defer st.Close()
	stats, err := importJSONL(context.Background(), st, in, *agent, time.Now(), os.Stderr)
	fmt.Fprintf(out, "created %d, already present %d, conclusions skipped %d, rejected %d\n", stats.Created, stats.Existing, stats.Conclusions, stats.Rejected)
	return err
}
