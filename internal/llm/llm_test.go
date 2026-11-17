package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewLocalRefusesRemoteHosts(t *testing.T) {
	for _, u := range []string{"https://api.example.com/v1/chat/completions", "http://10.0.0.5:11434/v1/chat/completions", "http://192.168.1.2/v1"} {
		if _, err := NewLocal(u, "m"); !errors.Is(err, ErrNotLocal) {
			t.Errorf("%s: err = %v, want ErrNotLocal", u, err)
		}
	}
	for _, u := range []string{"http://127.0.0.1:11434/v1/chat/completions", "http://localhost:8080/v1/chat/completions", "http://[::1]:8080/v1"} {
		if _, err := NewLocal(u, "m"); err != nil {
			t.Errorf("%s: %v", u, err)
		}
	}
}

func TestGenerate(t *testing.T) {
	var got map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"draft text"}}]}`))
	}))
	defer srv.Close()
	l, err := NewLocal(srv.URL+"/v1/chat/completions", "gemma4")
	if err != nil {
		t.Fatal(err)
	}
	out, err := l.Generate(context.Background(), "the prompt", GenerateOptions{Temperature: 0.3, SystemPrompt: "sys"})
	if err != nil || out != "draft text" {
		t.Fatalf("generate = %q, %v", out, err)
	}
	msgs, _ := got["messages"].([]interface{})
	if got["model"] != "gemma4" || got["temperature"] != 0.3 || len(msgs) != 2 {
		t.Fatalf("request: %v", got)
	}
}
