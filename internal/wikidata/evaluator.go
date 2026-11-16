package wikidata

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"gnomatix/validatorium/internal/evaluate"
	"gnomatix/validatorium/internal/model"
	"gnomatix/validatorium/internal/storage"
)

// StatementMethod identifies the record-pointer method: the premise "Wikidata
// entity E, at revision R, holds statement P = V" is true when that exact
// revision holds a non-deprecated statement with that property and value.
// The claim is about what the record holds, not an endorsement of it.
const StatementMethod = "urn:validatorium:method:wikidata-statement"

// StatementEvaluator evaluates StatementMethod. Scope keys: "entity" (Q-id),
// "revision" (number), "property" (P-id), "value" (normalized value, e.g. a
// Q-id). The revision is required, so the pointer always names one immutable
// record version.
type StatementEvaluator struct {
	Client *Client
}

var _ evaluate.Evaluator = StatementEvaluator{}

func (StatementEvaluator) Method() string { return StatementMethod }

func (s StatementEvaluator) Evaluate(ctx context.Context, premise *model.Micropublication, scope map[string]interface{}, at time.Time) (evaluate.Outcome, error) {
	entity, _ := scope["entity"].(string)
	property, _ := scope["property"].(string)
	value, _ := scope["value"].(string)
	rev, err := revisionOf(scope["revision"])
	if err != nil {
		return evaluate.Outcome{}, err
	}
	if !qidRe.MatchString(entity) || !qidRe.MatchString(property) || value == "" || rev <= 0 {
		return evaluate.Outcome{}, errors.New("wikidata-statement: scope needs entity, revision, property and value")
	}
	c := s.Client
	if c == nil {
		c = &Client{}
	}
	e, raw, err := c.Fetch(ctx, entity, rev)
	if err != nil {
		return evaluate.Outcome{Result: model.ResultIndeterminate, Detail: "record could not be read: " + err.Error()}, nil
	}
	cid, err := storage.ContentID(raw)
	if err != nil {
		return evaluate.Outcome{}, err
	}
	out := evaluate.Outcome{Evidence: []string{c.SourceURI(entity, rev), cid}}
	if e.Holds(property, value) {
		out.Result = model.ResultTrue
		out.Detail = fmt.Sprintf("%s@%d holds %s = %s", entity, rev, property, value)
	} else {
		out.Result = model.ResultFalse
		out.Detail = fmt.Sprintf("%s@%d does not hold %s = %s", entity, rev, property, value)
	}
	return out, nil
}

func revisionOf(v interface{}) (int64, error) {
	switch r := v.(type) {
	case int64:
		return r, nil
	case int:
		return int64(r), nil
	case float64:
		return int64(r), nil
	case string:
		return strconv.ParseInt(r, 10, 64)
	case nil:
		return 0, nil
	}
	return 0, fmt.Errorf("wikidata-statement: revision has type %T", v)
}
