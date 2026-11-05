package model

import (
	"fmt"
	"strings"
	"time"
)

// AttestationResult is the outcome of one evaluation of a premise version
// against its evidence and applicable context (docs/specs/data-model.md §6).
type AttestationResult string

const (
	// ResultTrue means the premise evaluated true for the recorded versions,
	// scope, and evaluation time. Only a true attestation can admit a premise
	// version to the validated store.
	ResultTrue AttestationResult = "true"
	// ResultFalse means the premise did not evaluate true for that scope and
	// time. The attestation stays in the evaluation log; it is not a valid
	// premise state.
	ResultFalse AttestationResult = "false"
	// ResultIndeterminate means the available evidence or context cannot
	// establish a Boolean result.
	ResultIndeterminate AttestationResult = "indeterminate"
)

// ValidityAttestation is the immutable record of one evaluation. It never
// mutates the premise it describes; re-evaluation appends a new attestation.
//
// The attestation is produced by an evaluator that executed the premise's
// method against its evidence. This package does not evaluate anything; it
// only defines the record and checks that it is well-formed.
type ValidityAttestation struct {
	// UUID identifies this attestation record.
	UUID string `json:"_id"`
	// PremiseUUID is the stable identity of the evaluated premise.
	PremiseUUID string `json:"premiseUUID"`
	// PremiseVersionCID is the content identifier of the exact premise version
	// that was evaluated (see Micropublication.CanonicalBytes).
	PremiseVersionCID string `json:"premiseVersionCID"`
	// Result is true, false, or indeterminate.
	Result AttestationResult `json:"result"`
	// EvaluatedAt is the RFC 3339 time of the evaluation.
	EvaluatedAt string `json:"evaluatedAt"`
	// Evaluator identifies the evaluating agent or program (attribution only;
	// it carries no evidential weight).
	Evaluator string `json:"evaluator"`
	// Method is the URI or CID of the reproducible evaluation method.
	Method string `json:"method"`
	// EvidenceVersions lists the exact evidence versions (URI or CID) consulted.
	EvidenceVersions []string `json:"evidenceVersions,omitempty"`
	// TermRecordUUIDs lists the versioned term records in force at evaluation.
	TermRecordUUIDs []string `json:"termRecordUUIDs,omitempty"`
	// Scope records the applicable context (time, environment, conditions).
	Scope map[string]interface{} `json:"scope,omitempty"`
	// Dependencies lists premise UUIDs or version CIDs the evaluation relied on.
	Dependencies []string `json:"dependencies,omitempty"`
	// ExpiresAt is an optional RFC 3339 time after which this attestation no
	// longer establishes validity and re-evaluation is required.
	ExpiresAt string `json:"expiresAt,omitempty"`
}

// NewAttestation builds a well-formed attestation with a fresh UUID for the
// given premise version. The caller supplies the result its evaluator produced.
func NewAttestation(premiseUUID, premiseVersionCID string, result AttestationResult, evaluator, method string, evaluatedAt time.Time) (*ValidityAttestation, error) {
	id, err := newUUIDv4()
	if err != nil {
		return nil, fmt.Errorf("generate uuid: %w", err)
	}
	return &ValidityAttestation{
		UUID:              id,
		PremiseUUID:       premiseUUID,
		PremiseVersionCID: premiseVersionCID,
		Result:            result,
		EvaluatedAt:       evaluatedAt.UTC().Format(time.RFC3339Nano),
		Evaluator:         evaluator,
		Method:            method,
	}, nil
}

// ValidateRecord checks that the attestation is well-formed. It does not
// re-run the evaluation and says nothing about whether the result is correct.
func (a *ValidityAttestation) ValidateRecord() error {
	if strings.TrimSpace(a.UUID) == "" {
		return fmt.Errorf("attestation: missing UUID")
	}
	if strings.TrimSpace(a.PremiseUUID) == "" {
		return fmt.Errorf("attestation %s: missing premise UUID", a.UUID)
	}
	if strings.TrimSpace(a.PremiseVersionCID) == "" {
		return fmt.Errorf("attestation %s: missing premise version CID", a.UUID)
	}
	switch a.Result {
	case ResultTrue, ResultFalse, ResultIndeterminate:
	default:
		return fmt.Errorf("attestation %s: invalid result %q", a.UUID, a.Result)
	}
	if _, err := time.Parse(time.RFC3339Nano, a.EvaluatedAt); err != nil {
		return fmt.Errorf("attestation %s: evaluatedAt is not RFC 3339: %w", a.UUID, err)
	}
	if strings.TrimSpace(a.Evaluator) == "" {
		return fmt.Errorf("attestation %s: missing evaluator", a.UUID)
	}
	if strings.TrimSpace(a.Method) == "" {
		return fmt.Errorf("attestation %s: missing method", a.UUID)
	}
	if a.ExpiresAt != "" {
		if _, err := time.Parse(time.RFC3339Nano, a.ExpiresAt); err != nil {
			return fmt.Errorf("attestation %s: expiresAt is not RFC 3339: %w", a.UUID, err)
		}
	}
	return nil
}

// EstablishesValidityAt reports whether this attestation establishes that its
// premise version is valid at the given time: the result is true and the
// attestation has not expired. A false or indeterminate result never
// establishes validity.
func (a *ValidityAttestation) EstablishesValidityAt(at time.Time) bool {
	if a.Result != ResultTrue {
		return false
	}
	if a.ExpiresAt == "" {
		return true
	}
	exp, err := time.Parse(time.RFC3339Nano, a.ExpiresAt)
	if err != nil {
		return false
	}
	return at.Before(exp)
}
