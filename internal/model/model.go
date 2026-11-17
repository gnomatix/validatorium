// Package model defines the Validatorium micropublication record: a factual
// premise together with the evidence that grounds it.
//
// The record is built from the Micropublications ontology (MP),
// http://purl.org/mp, as vendored in docs/specs/mp.owl (v1.18). Every field
// below maps to an MP class or property, named in its doc comment. Two aspects
// are NOT in MP and are the project's own layer, marked "project layer":
//
//   - UUID identity. MP has no identifier scheme. The UUID is the software's
//     handle for a premise (and the go-orbit-db Documents-store "_id"). Humans
//     read the Statement string; software addresses records by UUID.
//   - current/obsolete versioning. MP models support/challenge structurally and
//     has no lifecycle status. Versioning belongs to the IPFS/store layer.
//
// MP represents a premise's standing structurally, not as a status label: a
// Claim is grounded by its support graph (MP:hasSupportGraphElement, whose
// elements are MP:Data / MP:Method / MP:Material linked via supportedByData /
// supportedByMethod / supportedByEvidence) and disputed by its challenge graph
// (MP:hasChallengeGraphElement). There is no mutable validity-status field:
// the system stores only valid premises, so presence means valid.
package model

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"
)

// Micropublication is a factual premise plus its grounding — MP:Micropublication,
// "a set of Representations, having supports and/or challenges relations to one
// another," rooted at a Claim.
type Micropublication struct {
	// UUID — project layer (not MP). Stable software identity; also the
	// go-orbit-db Documents-store key ("_id").
	UUID string `json:"_id"`

	// Claim — MP:Claim, "the single principal Statement arguedBy a
	// Micropublication."
	Claim Statement `json:"claim"`

	// SupportGraph — MP:hasSupportGraphElement. Empirical evidence grounding the
	// claim. An empty support graph means the claim is not yet grounded.
	SupportGraph []Evidence `json:"supportGraph,omitempty"`

	// ChallengeGraph — MP:hasChallengeGraphElement. Evidence that challenges the
	// claim. Evidence-based per MP; never argument or opinion.
	ChallengeGraph []Evidence `json:"challengeGraph,omitempty"`

	// Attribution — MP:Attribution, "the minimal level of support for any
	// Statement is its Attribution to some Agent."
	Attribution Attribution `json:"attribution"`

	// State — project layer (not MP). Lifecycle/versioning.
	State Lifecycle `json:"state"`
}

// Statement is MP:Statement, "a declarative Sentence" that is "truth-bearing."
type Statement struct {
	// Text — MP:statement (datatype property). The premise's minimal string
	// representation: its canonical, shortest human-readable form. This string
	// is for humans; software identity is the parent UUID.
	Text string `json:"statement"`

	// Terms — grounding of the statement's words, via MP:SemanticQualifier
	// applied with MP:semtags. Each significant word resolves to an ontology
	// entry or standard-language sense.
	Terms []TermGrounding `json:"terms,omitempty"`
}

// EvidenceKind distinguishes the three MP empirical-evidence classes.
type EvidenceKind string

const (
	// EvidenceData — MP:Data, linked via MP:supportedByData / MP:dataSupports.
	EvidenceData EvidenceKind = "data"
	// EvidenceMethod — MP:Method ("a reusable recipe showing how the Data were
	// obtained"), linked via MP:supportedByMethod / MP:methodSupports.
	EvidenceMethod EvidenceKind = "method"
	// EvidenceMaterial — MP:Material, a component of a Method.
	EvidenceMaterial EvidenceKind = "material"
)

// Evidence is one element of a support or challenge graph — an MP:Data,
// MP:Method, or MP:Material, with its citation.
type Evidence struct {
	// Kind — which MP evidence class this element is.
	Kind EvidenceKind `json:"kind"`

	// Reference — MP:Reference identified by MP:citation. A resolvable pointer to
	// the source (DOI, URL, or IPFS CID). Evidence is linked, never embedded.
	Reference string `json:"reference,omitempty"`

	// Note — an optional MP:Statement describing this evidence element.
	Note string `json:"note,omitempty"`
}

// TermGrounding records how one word/phrase of a Statement is grounded — an
// MP:SemanticQualifier applied via MP:semtags.
type TermGrounding struct {
	// SurfaceForm — the word/phrase as it appears in the Statement text.
	SurfaceForm string `json:"surfaceForm"`
	// OntologyURI — the ontology entry the term resolves to (e.g. a Wikidata QID
	// or OBO URI), or empty when grounded to a standard-language sense.
	OntologyURI string `json:"ontologyURI,omitempty"`
	// Sense — a standard-language definition, used when there is no ontology
	// entry for an ordinary word.
	Sense string `json:"sense,omitempty"`
}

// Attribution is MP:Attribution: who asserted or curated a record, and when.
// Attribution is for accountability, never validity; it carries no epistemic
// weight and confers no standing.
type Attribution struct {
	// Agent — MP:attributionOfAgent. An anonymized signer identity (e.g. a key
	// fingerprint), sufficient to trace actions to one actor, not to a person.
	Agent string `json:"agent"`
	// AtTime — MP:atTime. RFC 3339 timestamp of the assertion.
	AtTime string `json:"atTime,omitempty"`
	// Curator — MP:attributionAsCurator / MP:curatedBy, when a reviewer curated
	// this record. Optional.
	Curator string `json:"curator,omitempty"`
}

// LifecycleStatus is the project-layer versioning state (not MP).
type LifecycleStatus string

const (
	// Current — the live premise.
	Current LifecycleStatus = "current"
	// Obsolete — superseded by a refinement or split. Obsolete records are
	// RETAINED as redirects to their successor(s); this is a state, not a
	// judgment about the premise.
	Obsolete LifecycleStatus = "obsolete"
)

// Lifecycle is project layer (not MP): a premise is Current until superseded,
// then Obsolete with a redirect to its successor UUID(s).
type Lifecycle struct {
	Status LifecycleStatus `json:"status"`
	// SupersededBy — successor UUID(s) an obsolete record redirects to.
	SupersededBy []string `json:"supersededBy,omitempty"`
}

// New builds a Current micropublication for the given statement text and
// attribution, assigning a fresh UUID. It does not validate; call Validate
// before evidence evaluation and admission processing.
func New(statementText string, attribution Attribution) (*Micropublication, error) {
	id, err := newUUIDv4()
	if err != nil {
		return nil, fmt.Errorf("generate uuid: %w", err)
	}
	return &Micropublication{
		UUID:        id,
		Claim:       Statement{Text: statementText},
		Attribution: attribution,
		State:       Lifecycle{Status: Current},
	}, nil
}

// ValidateRecord checks whether the record is well-formed for evaluation.
// It does not determine whether the premise is true; semantic validity requires
// evidence evaluation against the applicable context.
func (m *Micropublication) ValidateRecord() error {
	if strings.TrimSpace(m.UUID) == "" {
		return fmt.Errorf("micropublication: missing UUID")
	}
	if strings.TrimSpace(m.Claim.Text) == "" {
		return fmt.Errorf("micropublication %s: claim has no statement text", m.UUID)
	}
	if m.State.Status != Current && m.State.Status != Obsolete {
		return fmt.Errorf("micropublication %s: invalid lifecycle status %q", m.UUID, m.State.Status)
	}
	for _, e := range m.SupportGraph {
		if err := e.validate(); err != nil {
			return fmt.Errorf("micropublication %s: support graph: %w", m.UUID, err)
		}
	}
	for _, e := range m.ChallengeGraph {
		if err := e.validate(); err != nil {
			return fmt.Errorf("micropublication %s: challenge graph: %w", m.UUID, err)
		}
	}
	return nil
}

func (e Evidence) validate() error {
	switch e.Kind {
	case EvidenceData, EvidenceMethod, EvidenceMaterial:
	default:
		return fmt.Errorf("invalid evidence kind %q", e.Kind)
	}
	return nil
}

// Document renders the record as the map a go-orbit-db Documents store persists,
// keyed by "_id" (the UUID). It round-trips through the JSON tags so the stored
// document and the Go struct never drift.
func (m *Micropublication) Document() (map[string]interface{}, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("marshal micropublication: %w", err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("unmarshal to document: %w", err)
	}
	return doc, nil
}

// FromDocument parses a stored Documents-store record back into a
// Micropublication.
func FromDocument(doc map[string]interface{}) (*Micropublication, error) {
	b, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("marshal document: %w", err)
	}
	var m Micropublication
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("unmarshal micropublication: %w", err)
	}
	return &m, nil
}

// newUUIDv4 returns a random RFC 4122 version-4 UUID using crypto/rand. No
// external dependency: the store key is a string and this is standard library.
func newUUIDv4() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
