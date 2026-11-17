package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPremiseViewPage(t *testing.T) {
	srv := httptest.NewServer(&Server{})
	defer srv.Close()
	get := func(path string) (int, string, string) {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, resp.Header.Get("Content-Type"), string(b)
	}
	code, ct, body := get("/view/premises/abc-123")
	if code != 200 || !strings.HasPrefix(ct, "text/html") || !strings.Contains(body, `const id = "abc-123";`) || !strings.Contains(body, `src="/static/cytoscape.min.js"`) {
		t.Fatalf("page: %d %s\n%s", code, ct, body[:200])
	}
	// The UUID is escaped for the script context, so markup in it cannot run.
	_, _, body = get("/view/premises/" + "x%3C%2Fscript%3E%3Cscript%3Ealert(1)%3C%2Fscript%3E")
	if strings.Contains(body, "<script>alert(1)") {
		t.Fatal("unescaped markup in the page")
	}
	code, ct, body = get("/static/cytoscape.min.js")
	if code != 200 || !strings.HasPrefix(ct, "text/javascript") || len(body) != len(cytoscapeJS) || !strings.Contains(body[:200], "Cytoscape Consortium") {
		t.Fatalf("library: %d %s %d bytes", code, ct, len(body))
	}
}
