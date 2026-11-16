// Package evaluate executes a premise's evaluation method against evidence and
// scope and produces an immutable validity attestation
// (docs/specs/evidence-verification.md §4, docs/design/two-tier-storage.md §7.1).
//
// Evaluators never write. The caller stores the attestation in the working
// store and, for a true result, may hand it to promotion. A false or
// indeterminate result is a legitimate outcome and is recorded the same way.
//
// Evaluators are selected by the method URI recorded on the premise's support
// graph (an MP:Method evidence element). The method decides validity; who runs
// it, and how many times, carries no weight.
package evaluate

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"gnomatix/validatorium/internal/model"
	"gnomatix/validatorium/internal/storage"
)

// Evaluator executes one evaluation method.
type Evaluator interface {
	// Method is the URI of the method this evaluator implements.
	Method() string
	// Evaluate runs the method for the given premise version and scope at
	// time at. It returns the result and the exact evidence versions it
	// examined. It must not modify the premise or write to any store.
	Evaluate(ctx context.Context, premise *model.Micropublication, scope map[string]interface{}, at time.Time) (Outcome, error)
}

// Outcome is what an evaluator observed.
type Outcome struct {
	Result model.AttestationResult
	// Evidence lists the exact evidence versions examined (URIs or CIDs).
	Evidence []string
	// Detail is a short, factual statement of what was observed.
	Detail string
}

// Registry maps method URIs to evaluators.
type Registry struct {
	byMethod map[string]Evaluator
}

// NewRegistry returns a registry holding the given evaluators.
func NewRegistry(evs ...Evaluator) (*Registry, error) {
	r := &Registry{byMethod: map[string]Evaluator{}}
	for _, e := range evs {
		m := e.Method()
		if strings.TrimSpace(m) == "" {
			return nil, errors.New("evaluate: evaluator with empty method URI")
		}
		if _, dup := r.byMethod[m]; dup {
			return nil, fmt.Errorf("evaluate: two evaluators for method %s", m)
		}
		r.byMethod[m] = e
	}
	return r, nil
}

// Methods lists the registered method URIs in order.
func (r *Registry) Methods() []string {
	out := make([]string, 0, len(r.byMethod))
	for m := range r.byMethod {
		out = append(out, m)
	}
	sort.Strings(out)
	return out
}

// ErrNoMethod is returned when the premise names no method element, or names
// one no registered evaluator implements.
var ErrNoMethod = errors.New("evaluate: no registered evaluator for the premise's method")

// Attest evaluates the premise with the evaluator registered for its method
// and returns a well-formed attestation for its exact current version. The
// evaluator identity is recorded for attribution only.
func (r *Registry) Attest(ctx context.Context, premise *model.Micropublication, scope map[string]interface{}, evaluator string, at time.Time) (*model.ValidityAttestation, Outcome, error) {
	if premise == nil {
		return nil, Outcome{}, errors.New("evaluate: nil premise")
	}
	if err := premise.ValidateRecord(); err != nil {
		return nil, Outcome{}, err
	}
	method := methodOf(premise)
	ev, ok := r.byMethod[method]
	if !ok {
		return nil, Outcome{}, fmt.Errorf("%w (method %q)", ErrNoMethod, method)
	}
	out, err := ev.Evaluate(ctx, premise, scope, at)
	if err != nil {
		return nil, Outcome{}, fmt.Errorf("evaluate %s with %s: %w", premise.UUID, method, err)
	}
	switch out.Result {
	case model.ResultTrue, model.ResultFalse, model.ResultIndeterminate:
	default:
		return nil, Outcome{}, fmt.Errorf("evaluate %s: evaluator returned invalid result %q", premise.UUID, out.Result)
	}

	data, err := premise.CanonicalBytes()
	if err != nil {
		return nil, Outcome{}, err
	}
	cid, err := storage.ContentID(data)
	if err != nil {
		return nil, Outcome{}, err
	}
	att, err := model.NewAttestation(premise.UUID, cid, out.Result, evaluator, method, at)
	if err != nil {
		return nil, Outcome{}, err
	}
	att.EvidenceVersions = append([]string(nil), out.Evidence...)
	if len(scope) > 0 {
		att.Scope = scope
	}
	if err := att.ValidateRecord(); err != nil {
		return nil, Outcome{}, err
	}
	return att, out, nil
}

// methodOf returns the reference of the premise's first MP:Method element in
// its support graph, or "".
func methodOf(m *model.Micropublication) string {
	for _, e := range m.SupportGraph {
		if e.Kind == model.EvidenceMethod {
			return e.Reference
		}
	}
	return ""
}
