// Package llm talks to a local language model through the OpenAI-compatible
// chat endpoint that Ollama and llama.cpp serve (docs/specs/llm-integration.md
// §2). The privacy invariant of that spec, "No data leaves the local
// machine. Ever.", is enforced here: the client refuses any endpoint whose
// host is not a loopback address.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// GenerateOptions follows the option set of llm-integration.md §2.3.
type GenerateOptions struct {
	Temperature  float64
	MaxTokens    int
	Stop         []string
	SystemPrompt string
}

// Provider generates text from a prompt.
type Provider interface {
	Generate(ctx context.Context, prompt string, opts GenerateOptions) (string, error)
}

// ErrNotLocal is returned for an endpoint outside the local machine.
var ErrNotLocal = errors.New("llm: endpoint is not on the local machine")

// Local is a Provider for an OpenAI-compatible chat endpoint on this machine.
type Local struct {
	URL   string // e.g. http://127.0.0.1:11434/v1/chat/completions
	Model string
	HTTP  *http.Client
}

// NewLocal checks that the endpoint is a loopback address and returns a client.
func NewLocal(endpoint, model string) (*Local, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("llm: endpoint: %w", err)
	}
	host := u.Hostname()
	ip := net.ParseIP(host)
	if !(strings.EqualFold(host, "localhost") || (ip != nil && ip.IsLoopback())) {
		return nil, fmt.Errorf("%w: %s", ErrNotLocal, host)
	}
	if model == "" {
		return nil, errors.New("llm: model name is required")
	}
	return &Local{URL: endpoint, Model: model, HTTP: &http.Client{Timeout: 10 * time.Minute}}, nil
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Generate sends one chat request and returns the reply text.
func (l *Local) Generate(ctx context.Context, prompt string, opts GenerateOptions) (string, error) {
	msgs := []chatMessage{}
	if opts.SystemPrompt != "" {
		msgs = append(msgs, chatMessage{"system", opts.SystemPrompt})
	}
	msgs = append(msgs, chatMessage{"user", prompt})
	req := map[string]interface{}{"model": l.Model, "messages": msgs, "temperature": opts.Temperature, "stream": false}
	if opts.MaxTokens > 0 {
		req["max_tokens"] = opts.MaxTokens
	}
	if len(opts.Stop) > 0 {
		req["stop"] = opts.Stop
	}
	body, _ := json.Marshal(req)
	hr, err := http.NewRequestWithContext(ctx, http.MethodPost, l.URL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	hr.Header.Set("Content-Type", "application/json")
	resp, err := l.HTTP.Do(hr)
	if err != nil {
		return "", fmt.Errorf("llm: request: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llm: status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var out struct {
		Choices []struct {
			Message chatMessage `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &out); err != nil || len(out.Choices) == 0 {
		return "", fmt.Errorf("llm: unexpected reply: %.200s", raw)
	}
	return out.Choices[0].Message.Content, nil
}
