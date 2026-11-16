package evaluate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gnomatix/validatorium/internal/model"
	"gnomatix/validatorium/internal/storage"
)

var at = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

func filePremise(t *testing.T, path string) *model.Micropublication {
	t.Helper()
	m, _ := model.New("The file at "+path+" has the stated SHA-256 digest.", model.Attribution{Agent: "key:abc"})
	m.SupportGraph = []model.Evidence{
		{Kind: model.EvidenceMethod, Reference: FileDigestMethod},
		{Kind: model.EvidenceData, Reference: "file://" + path},
	}
	return m
}

func digest(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

func TestFileDigestResults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.txt")
	content := []byte("observable state\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	reg, err := NewRegistry(FileDigest{})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	prem := filePremise(t, path)
	before, _ := prem.CanonicalBytes()

	// Match: true, with the observed bytes' CID as evidence.
	att, out, err := reg.Attest(ctx, prem, map[string]interface{}{"sha256": digest(content)}, "evaluator:test", at)
	if err != nil {
		t.Fatalf("Attest: %v", err)
	}
	if att.Result != model.ResultTrue || out.Result != model.ResultTrue {
		t.Fatalf("result = %s, want true", att.Result)
	}
	wantCID, _ := storage.ContentID(content)
	if len(att.EvidenceVersions) != 2 || att.EvidenceVersions[1] != wantCID {
		t.Fatalf("evidence = %v, want the observed content CID %s", att.EvidenceVersions, wantCID)
	}
	premCID, _ := storage.ContentID(before)
	if att.PremiseUUID != prem.UUID || att.PremiseVersionCID != premCID || att.Method != FileDigestMethod {
		t.Fatalf("attestation does not name the exact premise version and method: %+v", att)
	}
	if !att.EstablishesValidityAt(at) {
		t.Fatal("true attestation does not establish validity")
	}

	// The premise was not modified by evaluation.
	after, _ := prem.CanonicalBytes()
	if string(after) != string(before) {
		t.Fatal("evaluation modified the premise")
	}

	// Changed file: false, and the new observation is recorded.
	_ = os.WriteFile(path, []byte("changed\n"), 0o600)
	att2, _, err := reg.Attest(ctx, prem, map[string]interface{}{"sha256": digest(content)}, "evaluator:test", at.Add(time.Minute))
	if err != nil || att2.Result != model.ResultFalse {
		t.Fatalf("changed file: %v %v", att2, err)
	}
	if att2.UUID == att.UUID {
		t.Fatal("re-evaluation reused the earlier attestation")
	}

	// Missing file: indeterminate, not false.
	_ = os.Remove(path)
	att3, out3, err := reg.Attest(ctx, prem, map[string]interface{}{"sha256": digest(content)}, "evaluator:test", at)
	if err != nil || att3.Result != model.ResultIndeterminate {
		t.Fatalf("missing file: %v %v", att3, err)
	}
	if out3.Detail == "" {
		t.Fatal("indeterminate outcome has no detail")
	}
}

func TestFileDigestRejectsMalformedInput(t *testing.T) {
	reg, _ := NewRegistry(FileDigest{})
	ctx := context.Background()
	prem := filePremise(t, "/nonexistent")
	for name, scope := range map[string]map[string]interface{}{
		"no digest":    {},
		"short digest": {"sha256": "abc"},
		"not hex":      {"sha256": "zz" + digest(nil)[2:]},
	} {
		if _, _, err := reg.Attest(ctx, prem, scope, "e", at); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	noFile, _ := model.New("No file reference.", model.Attribution{Agent: "key:abc"})
	noFile.SupportGraph = []model.Evidence{{Kind: model.EvidenceMethod, Reference: FileDigestMethod}}
	if _, _, err := reg.Attest(ctx, noFile, map[string]interface{}{"sha256": digest(nil)}, "e", at); err == nil {
		t.Error("premise without a file:// data element accepted")
	}
	// Size limit gives indeterminate, not an error or a false.
	dir := t.TempDir()
	big := filepath.Join(dir, "big")
	_ = os.WriteFile(big, make([]byte, 64), 0o600)
	small, _ := NewRegistry(FileDigest{MaxBytes: 16})
	att, _, err := small.Attest(ctx, filePremise(t, big), map[string]interface{}{"sha256": digest(make([]byte, 64))}, "e", at)
	if err != nil || att.Result != model.ResultIndeterminate {
		t.Fatalf("over-limit file: %v %v", att, err)
	}
}

func TestRegistrySelectsByMethod(t *testing.T) {
	if _, err := NewRegistry(FileDigest{}, FileDigest{}); err == nil {
		t.Fatal("duplicate method accepted")
	}
	reg, _ := NewRegistry(FileDigest{})
	if m := reg.Methods(); len(m) != 1 || m[0] != FileDigestMethod {
		t.Fatalf("Methods = %v", m)
	}
	other, _ := model.New("Uses an unregistered method.", model.Attribution{Agent: "key:abc"})
	other.SupportGraph = []model.Evidence{{Kind: model.EvidenceMethod, Reference: "urn:example:unknown"}}
	if _, _, err := reg.Attest(context.Background(), other, nil, "e", at); !errors.Is(err, ErrNoMethod) {
		t.Fatalf("expected ErrNoMethod, got %v", err)
	}
	none, _ := model.New("No method element.", model.Attribution{Agent: "key:abc"})
	if _, _, err := reg.Attest(context.Background(), none, nil, "e", at); !errors.Is(err, ErrNoMethod) {
		t.Fatalf("expected ErrNoMethod for missing method, got %v", err)
	}
}
