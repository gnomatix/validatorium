package evaluate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"gnomatix/validatorium/internal/model"
	"gnomatix/validatorium/internal/storage"
)

// FileDigestMethod identifies the runtime file-digest evaluation method: the
// premise "the file at P has SHA-256 digest D" is true when the bytes at P,
// read at evaluation time, hash to D.
const FileDigestMethod = "urn:validatorium:method:file-sha256"

// FileDigest evaluates FileDigestMethod against the local filesystem. The
// premise's support graph carries the file as an MP:Data element whose
// reference is a file:// URI; the expected digest is the scope value
// "sha256" (64 lowercase hex characters).
//
// Results: true when the digest matches; false when the file is read and the
// digest differs; indeterminate when the file cannot be read. The evidence
// recorded is the content identifier of the bytes actually read, so the exact
// observation is reproducible.
type FileDigest struct {
	// MaxBytes caps how much is read; 0 means 1 GiB.
	MaxBytes int64
}

func (FileDigest) Method() string { return FileDigestMethod }

func (f FileDigest) Evaluate(ctx context.Context, premise *model.Micropublication, scope map[string]interface{}, at time.Time) (Outcome, error) {
	want, _ := scope["sha256"].(string)
	want = strings.ToLower(strings.TrimSpace(want))
	if len(want) != 64 {
		return Outcome{}, errors.New("file-sha256: scope must give \"sha256\" as 64 hex characters")
	}
	if _, err := hex.DecodeString(want); err != nil {
		return Outcome{}, fmt.Errorf("file-sha256: scope sha256 is not hex: %w", err)
	}
	path := ""
	for _, e := range premise.SupportGraph {
		if e.Kind == model.EvidenceData && strings.HasPrefix(e.Reference, "file://") {
			path = strings.TrimPrefix(e.Reference, "file://")
			break
		}
	}
	if path == "" {
		return Outcome{}, errors.New("file-sha256: premise support graph has no file:// data element")
	}

	limit := f.MaxBytes
	if limit <= 0 {
		limit = 1 << 30
	}
	fh, err := os.Open(path)
	if err != nil {
		return Outcome{Result: model.ResultIndeterminate, Detail: "file could not be read: " + err.Error()}, nil
	}
	defer fh.Close()
	data, err := io.ReadAll(io.LimitReader(fh, limit+1))
	if err != nil {
		return Outcome{Result: model.ResultIndeterminate, Detail: "file read failed: " + err.Error()}, nil
	}
	if int64(len(data)) > limit {
		return Outcome{Result: model.ResultIndeterminate, Detail: fmt.Sprintf("file exceeds the %d-byte evaluation limit", limit)}, nil
	}
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	observed, err := storage.ContentID(data)
	if err != nil {
		return Outcome{}, err
	}
	out := Outcome{Evidence: []string{"file://" + path, observed}}
	if got == want {
		out.Result = model.ResultTrue
		out.Detail = "sha256 matches"
	} else {
		out.Result = model.ResultFalse
		out.Detail = "sha256 is " + got
	}
	return out, nil
}
