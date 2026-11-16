// Package wikidata resolves Wikidata entities into Validatorium records
// (docs/specs/wikidata-integration.md §§1, 3, 5.2).
//
// An external record is cited as a pointer, not as an authority: Validatorium
// keeps a structured form of exactly what the source holds at one revision,
// together with the revision-pinned source URI and the content identifier of
// the bytes read, and can re-check that the structured form is equivalent to
// the source. A claim backed this way is a claim about what the record holds;
// Validatorium adds no editorial judgment to it.
package wikidata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gnomatix/validatorium/internal/model"
	"gnomatix/validatorium/internal/storage"
)

// EntityURIPrefix is the canonical URI prefix of Wikidata items.
const EntityURIPrefix = "http://www.wikidata.org/entity/"

var qidRe = regexp.MustCompile(`^[QP][1-9][0-9]*$`)

// ErrNotFound is returned when Wikidata has no such entity or revision.
var ErrNotFound = errors.New("wikidata: entity or revision not found")

// Client reads entity data from a Wikibase instance.
type Client struct {
	// BaseURL is the wiki root, default https://www.wikidata.org.
	BaseURL string
	// HTTP is the client used; default has a 30 s timeout.
	HTTP *http.Client
	// UserAgent identifies the caller to Wikidata, as its API policy asks.
	UserAgent string
}

func (c *Client) base() string {
	if c.BaseURL != "" {
		return strings.TrimRight(c.BaseURL, "/")
	}
	return "https://www.wikidata.org"
}

// SourceURI returns the revision-pinned address of an entity's data.
func (c *Client) SourceURI(qid string, revision int64) string {
	u := c.base() + "/wiki/Special:EntityData/" + url.PathEscape(qid) + ".json"
	if revision > 0 {
		u += "?revision=" + strconv.FormatInt(revision, 10)
	}
	return u
}

// Fetch reads an entity, at revision when revision > 0, else its latest
// revision. It returns the parsed entity and the exact bytes read.
func (c *Client) Fetch(ctx context.Context, qid string, revision int64) (*Entity, []byte, error) {
	if !qidRe.MatchString(qid) {
		return nil, nil, fmt.Errorf("wikidata: %q is not an entity id", qid)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.SourceURI(qid, revision), nil)
	if err != nil {
		return nil, nil, err
	}
	ua := c.UserAgent
	if ua == "" {
		ua = "validatorium/0 (https://github.com/gnomatix/validatorium)"
	}
	req.Header.Set("User-Agent", ua)
	hc := c.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("wikidata: fetch %s: %w", qid, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusBadRequest {
		return nil, nil, fmt.Errorf("%w: %s revision %d (HTTP %d)", ErrNotFound, qid, revision, resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("wikidata: fetch %s: HTTP %d", qid, resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, nil, fmt.Errorf("wikidata: read %s: %w", qid, err)
	}
	e, err := ParseEntity(raw, qid)
	if err != nil {
		return nil, nil, err
	}
	if revision > 0 && e.Revision != revision {
		return nil, nil, fmt.Errorf("wikidata: asked for %s revision %d, got %d", qid, revision, e.Revision)
	}
	return e, raw, nil
}

// Statement is one main-snak value of an entity, in normalized string form.
type Statement struct {
	Property string `json:"property"`
	Value    string `json:"value"`
	Datatype string `json:"datatype,omitempty"`
	Rank     string `json:"rank"`
}

// Entity is the structured form of one entity revision.
type Entity struct {
	ID           string            `json:"id"`
	Revision     int64             `json:"revision"`
	Modified     string            `json:"modified,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
	Descriptions map[string]string `json:"descriptions,omitempty"`
	Statements   []Statement       `json:"statements"`
}

// ParseEntity parses Special:EntityData JSON for one entity into its
// structured form. Statements are sorted, so equal revisions give equal forms.
func ParseEntity(raw []byte, qid string) (*Entity, error) {
	var doc struct {
		Entities map[string]struct {
			ID           string                            `json:"id"`
			LastRevID    int64                             `json:"lastrevid"`
			Modified     string                            `json:"modified"`
			Labels       map[string]struct{ Value string } `json:"labels"`
			Descriptions map[string]struct{ Value string } `json:"descriptions"`
			Claims       map[string][]struct {
				Rank     string `json:"rank"`
				Mainsnak struct {
					Snaktype  string `json:"snaktype"`
					Property  string `json:"property"`
					Datatype  string `json:"datatype"`
					Datavalue struct {
						Type  string          `json:"type"`
						Value json.RawMessage `json:"value"`
					} `json:"datavalue"`
				} `json:"mainsnak"`
			} `json:"claims"`
		} `json:"entities"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("wikidata: parse %s: %w", qid, err)
	}
	src, ok := doc.Entities[qid]
	if !ok {
		// A redirected id is returned under its target id.
		if len(doc.Entities) != 1 {
			return nil, fmt.Errorf("%w: %s not in response", ErrNotFound, qid)
		}
		for _, v := range doc.Entities {
			src = v
		}
	}
	e := &Entity{ID: src.ID, Revision: src.LastRevID, Modified: src.Modified, Labels: map[string]string{}, Descriptions: map[string]string{}}
	for lang, l := range src.Labels {
		e.Labels[lang] = l.Value
	}
	for lang, d := range src.Descriptions {
		e.Descriptions[lang] = d.Value
	}
	for prop, claims := range src.Claims {
		for _, c := range claims {
			var v string
			switch c.Mainsnak.Snaktype {
			case "value":
				v = normalizeValue(c.Mainsnak.Datavalue.Type, c.Mainsnak.Datavalue.Value)
			default:
				v = "<" + c.Mainsnak.Snaktype + ">" // somevalue / novalue
			}
			e.Statements = append(e.Statements, Statement{Property: prop, Value: v, Datatype: c.Mainsnak.Datatype, Rank: c.Rank})
		}
	}
	sort.Slice(e.Statements, func(i, j int) bool {
		a, b := e.Statements[i], e.Statements[j]
		if a.Property != b.Property {
			return a.Property < b.Property
		}
		if a.Value != b.Value {
			return a.Value < b.Value
		}
		return a.Rank < b.Rank
	})
	return e, nil
}

func normalizeValue(typ string, raw json.RawMessage) string {
	switch typ {
	case "wikibase-entityid":
		var v struct{ ID string }
		if json.Unmarshal(raw, &v) == nil && v.ID != "" {
			return v.ID
		}
	case "string":
		var s string
		if json.Unmarshal(raw, &s) == nil {
			return s
		}
	case "time":
		var v struct {
			Time      string `json:"time"`
			Precision int    `json:"precision"`
		}
		if json.Unmarshal(raw, &v) == nil {
			return v.Time + "/" + strconv.Itoa(v.Precision)
		}
	case "quantity":
		var v struct{ Amount, Unit string }
		if json.Unmarshal(raw, &v) == nil {
			return v.Amount + " " + strings.TrimPrefix(v.Unit, EntityURIPrefix)
		}
	case "monolingualtext":
		var v struct{ Text, Language string }
		if json.Unmarshal(raw, &v) == nil {
			return v.Text + "@" + v.Language
		}
	case "globecoordinate":
		var v struct{ Latitude, Longitude float64 }
		if json.Unmarshal(raw, &v) == nil {
			return strconv.FormatFloat(v.Latitude, 'f', -1, 64) + "," + strconv.FormatFloat(v.Longitude, 'f', -1, 64)
		}
	}
	return string(raw)
}

// Holds reports whether the entity has a statement with this property and
// value whose rank is not deprecated.
func (e *Entity) Holds(property, value string) bool {
	for _, s := range e.Statements {
		if s.Property == property && s.Value == value && s.Rank != "deprecated" {
			return true
		}
	}
	return false
}

// Snapshot is Validatorium's side of the pointer: the structured form of an
// entity at one revision, the revision-pinned source URI, and the content
// identifier of the bytes read.
type Snapshot struct {
	Entity      Entity `json:"entity"`
	SourceURI   string `json:"sourceURI"`
	RawCID      string `json:"rawCID"`
	RetrievedAt string `json:"retrievedAt"`
}

// TakeSnapshot fetches an entity at revision (latest when 0) and returns its
// snapshot.
func (c *Client) TakeSnapshot(ctx context.Context, qid string, revision int64, at time.Time) (*Snapshot, error) {
	e, raw, err := c.Fetch(ctx, qid, revision)
	if err != nil {
		return nil, err
	}
	cid, err := storage.ContentID(raw)
	if err != nil {
		return nil, err
	}
	return &Snapshot{Entity: *e, SourceURI: c.SourceURI(e.ID, e.Revision), RawCID: cid, RetrievedAt: at.UTC().Format(time.RFC3339Nano)}, nil
}

// CanonicalBytes returns the snapshot's encoding for content addressing.
func (s *Snapshot) CanonicalBytes() ([]byte, error) { return json.Marshal(s) }

// Equivalent re-reads the snapshot's exact revision and reports whether the
// structured form taken now equals the one recorded.
func (c *Client) Equivalent(ctx context.Context, s *Snapshot) (bool, error) {
	e, _, err := c.Fetch(ctx, s.Entity.ID, s.Entity.Revision)
	if err != nil {
		return false, err
	}
	a, _ := json.Marshal(e)
	b, _ := json.Marshal(s.Entity)
	return string(a) == string(b), nil
}

// TermRecord turns an entity into a versioned term record pinned to the
// exact revision read.
func TermRecord(e *Entity, lang string, attribution model.Attribution) (*model.TermRecord, error) {
	label := e.Labels[lang]
	if label == "" {
		return nil, fmt.Errorf("wikidata: %s has no %q label", e.ID, lang)
	}
	t, err := model.NewTermRecord(label, EntityURIPrefix+e.ID, attribution)
	if err != nil {
		return nil, err
	}
	t.SourceVersion = "wikidata:" + e.ID + "@" + strconv.FormatInt(e.Revision, 10)
	t.Definition = e.Descriptions[lang]
	return t, t.ValidateRecord()
}
