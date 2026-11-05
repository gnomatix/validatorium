package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// TermRecord is one versioned ontology-term mapping record
// (docs/specs/data-model.md §4). A premise grounds each significant term to a
// TermRecord UUID; the record names the ontology entry and the ontology
// version it came from. When a mapping is superseded the old record is marked
// obsolete and points to its replacement record UUIDs; it is never deleted,
// so historical premise versions keep resolving their exact term references.
//
// Ontologies define what a term means. They do not determine whether a
// premise is true.
type TermRecord struct {
	// UUID identifies this term record across its versions.
	UUID string `json:"_id"`
	// Label is the preferred human-readable label of the term.
	Label string `json:"label"`
	// OntologyURI is the entry this record maps to (for example a Wikidata
	// entity URI or an OBO PURL).
	OntologyURI string `json:"ontologyURI"`
	// OntologyRecordUUID identifies the versioned ontology record (name plus
	// version) this mapping was taken from.
	OntologyRecordUUID string `json:"ontologyRecordUUID,omitempty"`
	// SourceVersion is the upstream ontology version or revision identifier
	// (for example a Wikidata revision ID or an OBO release date) the mapping
	// was read from.
	SourceVersion string `json:"sourceVersion,omitempty"`
	// Definition is the term's definition as given by the ontology at that
	// version.
	Definition string `json:"definition,omitempty"`
	// Attribution records who created the mapping record.
	Attribution Attribution `json:"attribution"`
	// State is current until superseded, then obsolete with replacement UUIDs.
	State Lifecycle `json:"state"`
}

// NewTermRecord builds a current term record with a fresh UUID.
func NewTermRecord(label, ontologyURI string, attribution Attribution) (*TermRecord, error) {
	id, err := newUUIDv4()
	if err != nil {
		return nil, fmt.Errorf("generate uuid: %w", err)
	}
	return &TermRecord{
		UUID:        id,
		Label:       label,
		OntologyURI: ontologyURI,
		Attribution: attribution,
		State:       Lifecycle{Status: Current},
	}, nil
}

// ValidateRecord checks that the term record is well-formed. An obsolete
// record must name at least one replacement; a current record must name none.
func (t *TermRecord) ValidateRecord() error {
	if strings.TrimSpace(t.UUID) == "" {
		return errors.New("term record: missing UUID")
	}
	if strings.TrimSpace(t.Label) == "" {
		return fmt.Errorf("term record %s: missing label", t.UUID)
	}
	if strings.TrimSpace(t.OntologyURI) == "" {
		return fmt.Errorf("term record %s: missing ontology URI", t.UUID)
	}
	switch t.State.Status {
	case Current:
		if len(t.State.SupersededBy) != 0 {
			return fmt.Errorf("term record %s: current record must not name replacements", t.UUID)
		}
	case Obsolete:
		if len(t.State.SupersededBy) == 0 {
			return fmt.Errorf("term record %s: obsolete record must name its replacement UUIDs", t.UUID)
		}
		for _, r := range t.State.SupersededBy {
			if r == t.UUID {
				return fmt.Errorf("term record %s: record cannot replace itself", t.UUID)
			}
		}
	default:
		return fmt.Errorf("term record %s: invalid lifecycle status %q", t.UUID, t.State.Status)
	}
	return nil
}

// CanonicalBytes returns the byte encoding of one immutable version of the
// term record for content addressing.
func (t *TermRecord) CanonicalBytes() ([]byte, error) {
	b, err := json.Marshal(t)
	if err != nil {
		return nil, fmt.Errorf("canonical term record: %w", err)
	}
	return b, nil
}

// TermRecordFromBytes decodes bytes produced by CanonicalBytes.
func TermRecordFromBytes(b []byte) (*TermRecord, error) {
	var t TermRecord
	if err := json.Unmarshal(b, &t); err != nil {
		return nil, fmt.Errorf("decode term record: %w", err)
	}
	return &t, nil
}

// TermLookup returns the term record for a UUID. It is the only dependency
// ResolveTermReplacements has on storage.
type TermLookup func(uuid string) (*TermRecord, error)

// ErrReplacementCycle is returned when replacement pointers form a cycle.
var ErrReplacementCycle = errors.New("term replacement pointers form a cycle")

// ResolveTermReplacements follows replacement pointers from the given term
// record UUID and returns the current records that replace it. A current
// record resolves to itself. A record obsoleted by a split resolves to every
// current leaf of the replacement graph, in first-seen order. The starting
// record and every record on the path are returned by the lookup unchanged;
// resolution never rewrites history.
func ResolveTermReplacements(lookup TermLookup, uuid string) ([]*TermRecord, error) {
	var (
		out     []*TermRecord
		visited = map[string]bool{}
		onPath  = map[string]bool{}
	)
	var walk func(id string) error
	walk = func(id string) error {
		if onPath[id] {
			return fmt.Errorf("%w: at %s", ErrReplacementCycle, id)
		}
		if visited[id] {
			return nil
		}
		rec, err := lookup(id)
		if err != nil {
			return fmt.Errorf("resolve term %s: %w", id, err)
		}
		if rec.State.Status == Current {
			visited[id] = true
			out = append(out, rec)
			return nil
		}
		onPath[id] = true
		for _, next := range rec.State.SupersededBy {
			if err := walk(next); err != nil {
				return err
			}
		}
		onPath[id] = false
		visited[id] = true
		return nil
	}
	if err := walk(uuid); err != nil {
		return nil, err
	}
	return out, nil
}
