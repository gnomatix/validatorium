package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"gnomatix/validatorium/internal/bbolt"
	"gnomatix/validatorium/internal/llm"
	"gnomatix/validatorium/internal/storage"
)

// draftSystemPrompt instructs the local model. The model proposes text; it
// decides nothing about validity (llm-integration.md).
const draftSystemPrompt = `You write draft prose for a human author.
Use only the premises you are given, in the order given, keeping each one's meaning exactly.
Where the author gives glue text, use it as written to join the premises.
Add no facts, no examples, no hedging and no opinions. Reply with the prose only.`

// cmdDraft asks a local model to draft prose from premises (validatorium-3xe).
func cmdDraft(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("draft", flag.ContinueOnError)
	ws := openWS(fs)
	uuids := fs.String("uuid", "", "comma-separated premise UUIDs, in the order the text should use them (required)")
	glue := fs.String("glue", "", "optional file of author-written glue text")
	endpoint := fs.String("llm-url", "http://127.0.0.1:11434/v1/chat/completions", "OpenAI-compatible chat endpoint of a model on this machine (loopback only)")
	model := fs.String("model", "gemma4", "model name the endpoint serves")
	if err := fs.Parse(args); err != nil {
		return err
	}
	ids := splitList(*uuids)
	if len(ids) == 0 {
		return errors.New("draft: -uuid is required")
	}
	provider, err := llm.NewLocal(*endpoint, *model)
	if err != nil {
		return err
	}
	st, err := bbolt.OpenWorkingStore(*ws)
	if err != nil {
		return err
	}
	defer st.Close()
	ctx := context.Background()
	var b strings.Builder
	b.WriteString("Premises:\n")
	for i, id := range ids {
		text, err := premiseStatement(ctx, st, id)
		if err != nil {
			return fmt.Errorf("draft: premise %s: %w", id, err)
		}
		fmt.Fprintf(&b, "%d. %s\n", i+1, text)
	}
	if *glue != "" {
		g, err := os.ReadFile(*glue)
		if err != nil {
			return err
		}
		fmt.Fprintf(&b, "\nAuthor's glue text:\n%s\n", strings.TrimSpace(string(g)))
	}
	text, err := provider.Generate(ctx, b.String(), llm.GenerateOptions{Temperature: 0.3, SystemPrompt: draftSystemPrompt})
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "DRAFT by local model %s from %d premises (not a premise; not evaluated)\n\n%s\n", *model, len(ids), strings.TrimSpace(text))
	return nil
}

// premiseStatement returns the statement of the current published version,
// or of the candidate draft when none is published.
func premiseStatement(ctx context.Context, st *bbolt.WorkingStore, id string) (string, error) {
	if ref, err := st.GetCurrent(ctx, id); err == nil {
		m, err := st.GetVersion(ctx, ref.CID)
		if err != nil {
			return "", err
		}
		return m.Claim.Text, nil
	} else if !errors.Is(err, storage.ErrNotFound) {
		return "", err
	}
	m, err := st.GetCandidate(ctx, id)
	if err != nil {
		return "", err
	}
	return m.Claim.Text, nil
}
