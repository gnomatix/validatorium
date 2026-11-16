package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func run1(t *testing.T, name string, args ...string) string {
	t.Helper()
	var out bytes.Buffer
	if err := subcommands[name](args, &out); err != nil {
		t.Fatalf("val %s %v: %v", name, args, err)
	}
	return out.String()
}

func TestLocalWorkflow(t *testing.T) {
	dir := t.TempDir()
	ws := filepath.Join(dir, "w.db")
	file := filepath.Join(dir, "state.txt")
	content := []byte("observed\n")
	if err := os.WriteFile(file, content, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	digest := hex.EncodeToString(sum[:])

	uuid := strings.TrimSpace(run1(t, "create", "-working-store", ws, "-text", "The file has the stated digest.", "-file", file))
	if len(uuid) != 36 {
		t.Fatalf("create printed %q, want a UUID", uuid)
	}

	out := run1(t, "evaluate", "-working-store", ws, "-uuid", uuid, "-sha256", digest)
	if !strings.HasPrefix(out, "true ") {
		t.Fatalf("evaluate printed %q, want a true result", out)
	}
	wrong := strings.Repeat("0", 64)
	out = run1(t, "evaluate", "-working-store", ws, "-uuid", uuid, "-sha256", wrong)
	if !strings.HasPrefix(out, "false ") {
		t.Fatalf("evaluate with the wrong digest printed %q, want false", out)
	}

	lines := strings.Split(strings.TrimSpace(run1(t, "attestations", "-working-store", ws, "-uuid", uuid)), "\n")
	if len(lines) != 2 {
		t.Fatalf("attestations printed %d lines, want 2", len(lines))
	}

	// The premise is still a draft (never promoted), so a release has no
	// current version to tag and reports it.
	var rel struct {
		CID        string
		Manifest   struct{ Premises []json.RawMessage }
		Exclusions []struct{ UUID, Reason string }
	}
	if err := json.Unmarshal([]byte(run1(t, "release", "-working-store", ws, "-label", "local-1", "-uuids", uuid)), &rel); err != nil {
		t.Fatalf("release output is not JSON: %v", err)
	}
	if rel.CID == "" || len(rel.Manifest.Premises) != 0 || len(rel.Exclusions) != 1 || rel.Exclusions[0].Reason != "no current version" {
		t.Fatalf("release = %+v", rel)
	}
}

func TestWorkflowRequiresArguments(t *testing.T) {
	ws := filepath.Join(t.TempDir(), "w.db")
	var out bytes.Buffer
	for name, args := range map[string][]string{
		"create":       {"-working-store", ws},
		"evaluate":     {"-working-store", ws},
		"attestations": {"-working-store", ws},
		"release":      {"-working-store", ws, "-label", "x"},
	} {
		if err := subcommands[name](args, &out); err == nil {
			t.Errorf("%s with missing arguments succeeded", name)
		}
	}
}
