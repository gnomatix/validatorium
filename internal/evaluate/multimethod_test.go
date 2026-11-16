package evaluate

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gnomatix/validatorium/internal/model"
)

// sizeMethod is an independent check of the same file premise: it observes
// the file's length instead of its digest.
type sizeMethod struct{ want int64 }

func (sizeMethod) Method() string { return "urn:test:method:file-size" }

func (s sizeMethod) Evaluate(_ context.Context, p *model.Micropublication, _ map[string]interface{}, _ time.Time) (Outcome, error) {
	for _, e := range p.SupportGraph {
		if e.Kind == model.EvidenceData {
			fi, err := os.Stat(e.Reference[len("file://"):])
			if err != nil {
				return Outcome{Result: model.ResultIndeterminate, Detail: err.Error()}, nil
			}
			if fi.Size() == s.want {
				return Outcome{Result: model.ResultTrue, Evidence: []string{e.Reference}}, nil
			}
			return Outcome{Result: model.ResultFalse, Evidence: []string{e.Reference}}, nil
		}
	}
	return Outcome{Result: model.ResultIndeterminate}, nil
}

func TestSamePremiseVersionAnyNumberOfMethods(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "f.txt")
	content := []byte("observable state\n")
	_ = os.WriteFile(path, content, 0o600)
	prem := filePremise(t, path)

	reg, err := NewRegistry(FileDigest{}, sizeMethod{want: int64(len(content))})
	if err != nil {
		t.Fatal(err)
	}
	scope := map[string]interface{}{"sha256": digest(content)}

	a1, _, err := reg.Attest(ctx, prem, scope, "evaluator:one", at)
	if err != nil || a1.Result != model.ResultTrue || a1.Method != FileDigestMethod {
		t.Fatalf("first method: %+v %v", a1, err)
	}
	// A method the premise record does not name.
	a2, _, err := reg.AttestWith(ctx, prem, "urn:test:method:file-size", nil, "evaluator:two", at)
	if err != nil || a2.Result != model.ResultTrue || a2.Method != "urn:test:method:file-size" {
		t.Fatalf("second method: %+v %v", a2, err)
	}
	if a1.PremiseVersionCID != a2.PremiseVersionCID || a1.UUID == a2.UUID {
		t.Fatalf("expected two attestations of one version: %s/%s %s/%s", a1.UUID, a1.PremiseVersionCID, a2.UUID, a2.PremiseVersionCID)
	}
	// The same method may run again, any number of times.
	a3, _, err := reg.Attest(ctx, prem, scope, "evaluator:three", at.Add(time.Hour))
	if err != nil || a3.UUID == a1.UUID {
		t.Fatalf("repeat evaluation: %+v %v", a3, err)
	}
	if _, _, err := reg.AttestWith(ctx, prem, "urn:test:unregistered", nil, "e", at); !errors.Is(err, ErrNoMethod) {
		t.Fatalf("unregistered method: %v", err)
	}
}
