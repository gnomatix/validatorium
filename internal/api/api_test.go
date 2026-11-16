package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"

	"gnomatix/validatorium/internal/bbolt"
)

func newServer(t *testing.T) (*httptest.Server, *bbolt.WorkingStore) {
	t.Helper()
	ws, err := bbolt.OpenWorkingStore(filepath.Join(t.TempDir(), "w.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ws.Close() })
	srv := httptest.NewServer(&Server{Store: ws})
	t.Cleanup(srv.Close)
	return srv, ws
}

func post(t *testing.T, u string, body interface{}) (*http.Response, map[string]string) {
	t.Helper()
	b, _ := json.Marshal(body)
	resp, err := http.Post(u, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	out := map[string]string{}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp, out
}

func verify(t *testing.T, base, subject string) (*http.Response, VerificationResponse) {
	t.Helper()
	resp, err := http.Get(base + "/api/verification?url=" + url.QueryEscape(subject))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var v VerificationResponse
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatal(err)
	}
	return resp, v
}

func TestRegisterVerifyObsolete(t *testing.T) {
	srv, ws := newServer(t)
	page := "https://example.org/article"

	// Nothing registered: not verified, still a well-formed ITVB response.
	resp, v := verify(t, srv.URL, page)
	if resp.StatusCode != 200 || v.Verified || v.VerificationURL != VerificationURL || v.Validatorium.Class != "self-certified" {
		t.Fatalf("empty registry: %d %+v", resp.StatusCode, v)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("missing CORS header for browser plugins")
	}

	r, reg := post(t, srv.URL+"/api/certifications", map[string]string{
		"subject": page, "method": "frontier-model-analysis", "date": "2026-11-16", "signer": "OPENPGP4FPR:C50B7193425E7E8D64AA825AE0FE10E03D83483A",
	})
	if r.StatusCode != 201 || reg["uuid"] == "" || reg["version"] == "" {
		t.Fatalf("register: %d %v", r.StatusCode, reg)
	}

	resp, v = verify(t, srv.URL, page)
	if !v.Verified || v.Standard != "seal-of-truth" || v.Method != "frontier-model-analysis" || v.Date != "2026-11-16" || v.BadgeURLs == nil {
		t.Fatalf("after registration: %+v", v)
	}
	if resp.Header.Get("X-Content-Verification") != "verified-true" || resp.Header.Get("X-Verification-Standard") != "seal-of-truth" {
		t.Fatalf("ITVB headers missing: %v", resp.Header)
	}
	if len(v.Validatorium.Current) != 1 || v.Validatorium.Current[0].UUID != reg["uuid"] || v.Validatorium.Current[0].SignerVerified {
		t.Fatalf("validatorium block: %+v", v.Validatorium)
	}

	// Anyone may obsolete it; the record stays on file.
	r, obs := post(t, srv.URL+"/api/certifications/"+reg["uuid"]+"/obsolete", map[string]string{"by": "reader:abc", "reason": "cited source retracted"})
	if r.StatusCode != 200 || obs["version"] == reg["version"] {
		t.Fatalf("obsolete: %d %v", r.StatusCode, obs)
	}
	_, v = verify(t, srv.URL, page)
	if v.Verified || v.Validatorium.Obsolete != 1 || len(v.Validatorium.Current) != 0 {
		t.Fatalf("after obsoleting: %+v", v)
	}
	// The original version is still addressable.
	orig, err := ws.GetCertificationVersion(t.Context(), reg["version"])
	if err != nil || orig.State.Status != "current" {
		t.Fatalf("original version lost: %+v %v", orig, err)
	}
	// Obsoleting twice is refused.
	if r, _ := post(t, srv.URL+"/api/certifications/"+reg["uuid"]+"/obsolete", map[string]string{"by": "x", "reason": "y"}); r.StatusCode != 409 {
		t.Fatalf("second obsolete: %d", r.StatusCode)
	}
	// The current record shows who obsoleted it and why.
	gr, err := http.Get(srv.URL + "/api/certifications/" + reg["uuid"])
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Version       string
		Certification struct {
			ObsoletedBy    string
			ObsoleteReason string
		}
	}
	_ = json.NewDecoder(gr.Body).Decode(&got)
	gr.Body.Close()
	if got.Version != obs["version"] || got.Certification.ObsoletedBy != "reader:abc" || got.Certification.ObsoleteReason != "cited source retracted" {
		t.Fatalf("get after obsolete: %+v", got)
	}
}

func TestRejectsMalformedRequests(t *testing.T) {
	srv, _ := newServer(t)
	cases := []map[string]string{
		{"subject": "not a url", "method": "m", "date": "2026-11-16", "signer": "s"},
		{"subject": "https://example.org", "method": "m", "date": "16/11/2026", "signer": "s"},
		{"subject": "https://example.org", "method": "m", "date": "2026-11-16"},
	}
	for _, c := range cases {
		if r, _ := post(t, srv.URL+"/api/certifications", c); r.StatusCode != 400 {
			t.Errorf("%v: status %d, want 400", c, r.StatusCode)
		}
	}
	if r, _ := post(t, srv.URL+"/api/certifications/nope/obsolete", map[string]string{"by": "x"}); r.StatusCode != 404 {
		t.Errorf("obsolete unknown: %d", r.StatusCode)
	}
	resp, err := http.Get(srv.URL + "/api/verification")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Errorf("verification without url: %d", resp.StatusCode)
	}
}
