package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"gnomatix/validatorium/internal/bbolt"
	"gnomatix/validatorium/internal/model"
	"gnomatix/validatorium/internal/promote"
	"gnomatix/validatorium/internal/storage"
	"gnomatix/validatorium/internal/storage/memory"
)

func getJSON(t *testing.T, u string, v interface{}) int {
	t.Helper()
	resp, err := http.Get(u)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_ = json.NewDecoder(resp.Body).Decode(v)
	return resp.StatusCode
}

func TestPremiseLifecycleThroughAPI(t *testing.T) {
	ctx := context.Background()
	ws, err := bbolt.OpenWorkingStore(filepath.Join(t.TempDir(), "w.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()
	at := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	clock := at
	srv := httptest.NewServer(&Server{Store: ws, Now: func() time.Time { return clock }})
	defer srv.Close()

	// Anyone may state a premise; it starts as a candidate.
	r, out := post(t, srv.URL+"/api/premises", map[string]interface{}{"statement": "Jeff's favorite food is strawberry Pop-Tarts.", "agent": "key:jeff"})
	if r.StatusCode != 201 || out["status"] != StatusCandidate {
		t.Fatalf("create: %d %v", r.StatusCode, out)
	}
	id := out["uuid"]
	var pv PremiseView
	if code := getJSON(t, srv.URL+"/api/premises/"+id, &pv); code != 200 || pv.Status != StatusCandidate {
		t.Fatalf("candidate view: %d %+v", code, pv)
	}

	// Evaluated true and published: valid.
	m, _ := ws.GetCandidate(ctx, id)
	data, _ := m.CanonicalBytes()
	cid, _ := storage.ContentID(data)
	att, err := model.NewAttestation(id, cid, model.ResultTrue, "evaluator:test", "urn:test:method", at)
	if err != nil {
		t.Fatal(err)
	}
	att.ExpiresAt = at.Add(24 * time.Hour).Format(time.RFC3339)
	pr := &promote.Promoter{Working: ws, Global: memory.NewGlobalStore(), Now: func() time.Time { return at }}
	if _, err := pr.Promote(ctx, id, att); err != nil {
		t.Fatal(err)
	}
	if code := getJSON(t, srv.URL+"/api/premises/"+id, &pv); code != 200 || pv.Status != StatusValid || pv.Version != cid || len(pv.Attestations) != 1 {
		t.Fatalf("valid view: %d %+v", code, pv)
	}

	// Search finds it, valid first.
	var res struct {
		Results []PremiseView `json:"results"`
	}
	_, _ = post(t, srv.URL+"/api/premises", map[string]interface{}{"statement": "Strawberry is a flavor.", "agent": "key:other"})
	if code := getJSON(t, srv.URL+"/api/premises?q="+url.QueryEscape("strawberry"), &res); code != 200 || len(res.Results) != 2 || res.Results[0].UUID != id {
		t.Fatalf("search: %d %+v", code, res)
	}

	// After the attestation expires, still published but not currently valid.
	clock = at.Add(48 * time.Hour)
	if code := getJSON(t, srv.URL+"/api/premises/"+id, &pv); code != 200 || pv.Status != StatusPublishedNotValid {
		t.Fatalf("expired view: %d %+v", code, pv)
	}
	if code := getJSON(t, srv.URL+"/api/premises/no-such-uuid", &pv); code != 404 {
		t.Fatalf("missing: %d", code)
	}
}
