package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gnomatix/validatorium/internal/autopublish"
	"gnomatix/validatorium/internal/evaluate"
	"gnomatix/validatorium/internal/model"
	"gnomatix/validatorium/internal/promote"
	"gnomatix/validatorium/internal/storage"
)

type fakeValidator struct{ gotScope map[string]interface{} }

func (f *fakeValidator) Run(_ context.Context, id string, scope map[string]interface{}) (autopublish.Outcome, error) {
	f.gotScope = scope
	switch id {
	case "missing":
		return autopublish.Outcome{}, fmt.Errorf("autopublish %s: load candidate: %w", id, storage.ErrNotFound)
	case "nomethod":
		return autopublish.Outcome{}, fmt.Errorf("autopublish %s: %w", id, evaluate.ErrNoMethod)
	case "false":
		return autopublish.Outcome{Attestation: &model.ValidityAttestation{Result: model.ResultFalse, Method: "urn:m"}, Reason: "evaluation result false"}, nil
	}
	return autopublish.Outcome{
		Attestation: &model.ValidityAttestation{Result: model.ResultTrue, Method: "urn:m"},
		Published:   true,
		Promotion:   promote.Result{Version: storage.VersionRef{CID: "bafyversion"}, AttestationCID: "bafyatt"},
	}, nil
}

func postValidate(t *testing.T, base, id, body string) (int, map[string]interface{}) {
	t.Helper()
	resp, err := http.Post(base+"/api/premises/"+id+"/validate", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	out := map[string]interface{}{}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func TestValidateRoute(t *testing.T) {
	plain := httptest.NewServer(&Server{})
	defer plain.Close()
	if code, _ := postValidate(t, plain.URL, "u1", ""); code != http.StatusNotImplemented {
		t.Fatalf("without a validator: %d, want 501", code)
	}

	fv := &fakeValidator{}
	srv := httptest.NewServer(&Server{Validator: fv})
	defer srv.Close()

	code, out := postValidate(t, srv.URL, "u1", `{"scope":{"sha256":"abc"}}`)
	if code != 200 || out["published"] != true || out["result"] != "true" || out["version"] != "bafyversion" || out["attestation_cid"] != "bafyatt" {
		t.Fatalf("published: %d %v", code, out)
	}
	if fv.gotScope["sha256"] != "abc" {
		t.Fatalf("scope not passed: %v", fv.gotScope)
	}
	if code, out := postValidate(t, srv.URL, "u1", ""); code != 200 || fv.gotScope != nil {
		t.Fatalf("empty body: %d %v scope %v", code, out, fv.gotScope)
	}
	if code, out := postValidate(t, srv.URL, "false", ""); code != 200 || out["published"] != false || out["result"] != "false" || out["reason"] == nil {
		t.Fatalf("false result: %d %v", code, out)
	}
	if code, _ := postValidate(t, srv.URL, "missing", ""); code != http.StatusNotFound {
		t.Fatalf("missing: %d, want 404", code)
	}
	if code, _ := postValidate(t, srv.URL, "nomethod", ""); code != http.StatusUnprocessableEntity {
		t.Fatalf("no method: %d, want 422", code)
	}
	if code, _ := postValidate(t, srv.URL, "u1", "{"); code != http.StatusBadRequest {
		t.Fatalf("bad JSON: %d, want 400", code)
	}
}
