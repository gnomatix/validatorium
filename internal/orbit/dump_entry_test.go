package orbit

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	orbitdb "berty.tech/go-orbit-db"
	"github.com/ipfs/boxo/path"
)

// TestDumpEntryForInterop writes one go-orbit-db event-log entry block to
// the file named by VALIDATORIUM_DUMP_ENTRY, for the Go/JavaScript
// comparison in docs/research/go-orbit-db-parity.md. Skipped otherwise.
func TestDumpEntryForInterop(t *testing.T) {
	out := os.Getenv("VALIDATORIUM_DUMP_ENTRY")
	if out == "" {
		t.Skip("set VALIDATORIUM_DUMP_ENTRY to a file path")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	odb, dir := newODB(ctx, t)
	lg, err := odb.Log(ctx, "interop", &orbitdb.CreateDBOptions{Directory: &dir})
	if err != nil {
		t.Fatal(err)
	}
	op, err := lg.Add(ctx, []byte("interop-payload"))
	if err != nil {
		t.Fatal(err)
	}
	blk, err := odb.IPFS().Block().Get(ctx, path.FromCid(op.GetEntry().GetHash()))
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	_, _ = b.ReadFrom(blk)
	if err := os.WriteFile(out, b.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %d bytes, cid %s", b.Len(), op.GetEntry().GetHash())
}
