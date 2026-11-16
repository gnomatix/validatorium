// Package autopublish enacts automatic publication on validity (Beads
// decision validatorium-o2u): a local candidate is evaluated by the evaluator
// registered for its method, the attestation is recorded whatever its result,
// and the candidate is published to the global store when, and only when, the
// attestation establishes validity. There is no separate push step.
//
// The package adds no admission rule of its own. Evaluation is
// evaluate.Registry.Attest, and admission is promote.Promoter.PromoteInto, so
// the rules for both are defined in exactly one place.
package autopublish

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gnomatix/validatorium/internal/evaluate"
	"gnomatix/validatorium/internal/model"
	"gnomatix/validatorium/internal/namespace"
	"gnomatix/validatorium/internal/promote"
)

// Publisher evaluates working-store candidates and publishes the valid ones.
type Publisher struct {
	Promoter   *promote.Promoter
	Evaluators *evaluate.Registry
	// Evaluator is the agent recorded on every attestation (attribution only).
	Evaluator string
	// Namespaces receive a pointer to every published version, in both tiers.
	Namespaces []namespace.Name
	// Now supplies the evaluation time; nil means time.Now. Promotion uses
	// the Promoter's own clock.
	Now func() time.Time
}

// Outcome reports one evaluation and what followed from it.
type Outcome struct {
	Attestation *model.ValidityAttestation
	Evaluation  evaluate.Outcome
	// Published is true when the candidate was written to the global store.
	Published bool
	// Promotion is set when Published is true.
	Promotion promote.Result
	// Reason states why the candidate was not published, when it was not.
	Reason string
}

// Run evaluates the candidate with this UUID and publishes it if the result
// establishes validity. A false or indeterminate result is recorded in the
// working store and is not an error. An error means nothing was decided:
// the candidate could not be loaded or evaluated, or a store write failed.
func (p *Publisher) Run(ctx context.Context, premiseUUID string, scope map[string]interface{}) (Outcome, error) {
	if p.Promoter == nil || p.Promoter.Working == nil || p.Promoter.Global == nil || p.Evaluators == nil {
		return Outcome{}, errors.New("autopublish: promoter with both stores and an evaluator registry are required")
	}
	if p.Evaluator == "" {
		return Outcome{}, errors.New("autopublish: evaluator identity is required")
	}
	now := time.Now
	if p.Now != nil {
		now = p.Now
	}
	cand, err := p.Promoter.Working.GetCandidate(ctx, premiseUUID)
	if err != nil {
		return Outcome{}, fmt.Errorf("autopublish %s: load candidate: %w", premiseUUID, err)
	}
	att, ev, err := p.Evaluators.Attest(ctx, cand, scope, p.Evaluator, now())
	if err != nil {
		return Outcome{}, fmt.Errorf("autopublish %s: %w", premiseUUID, err)
	}
	out := Outcome{Attestation: att, Evaluation: ev}

	if att.Result != model.ResultTrue {
		if err := p.Promoter.Working.PutAttestation(ctx, att); err != nil {
			return out, fmt.Errorf("autopublish %s: record attestation: %w", premiseUUID, err)
		}
		out.Reason = fmt.Sprintf("evaluation result %s", att.Result)
		return out, nil
	}
	res, err := p.Promoter.PromoteInto(ctx, premiseUUID, att, p.Namespaces)
	switch {
	case err == nil:
		out.Published, out.Promotion = true, res
		return out, nil
	case errors.Is(err, promote.ErrNotValid), errors.Is(err, promote.ErrNotCurrent), errors.Is(err, promote.ErrVersionMismatch):
		// Refused at admission (for example the attestation expired between
		// evaluation and promotion). The attestation stays on record.
		if perr := p.Promoter.Working.PutAttestation(ctx, att); perr != nil {
			return out, fmt.Errorf("autopublish %s: record attestation: %w", premiseUUID, perr)
		}
		out.Reason = err.Error()
		return out, nil
	default:
		return out, err
	}
}

// RunAll runs every listed candidate in order and returns one outcome per
// UUID. It stops at the first error, returning the outcomes gathered so far.
func (p *Publisher) RunAll(ctx context.Context, uuids []string, scope map[string]interface{}) ([]Outcome, error) {
	outs := make([]Outcome, 0, len(uuids))
	for _, id := range uuids {
		o, err := p.Run(ctx, id, scope)
		if err != nil {
			return outs, err
		}
		outs = append(outs, o)
	}
	return outs, nil
}
