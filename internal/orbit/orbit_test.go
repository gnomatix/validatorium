package orbit

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"berty.tech/go-orbit-db/stores"
	"github.com/ipfs/go-cid"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	mh "github.com/multiformats/go-multihash"

	"gnomatix/validatorium/internal/ipfsnode"
	"gnomatix/validatorium/internal/storage"
)

func newMockNode(ctx context.Context, t *testing.T, mn mocknet.Mocknet) *ipfsnode.Node {
	t.Helper()
	n, err := ipfsnode.NewMockNode(ctx, mn)
	if err != nil {
		t.Fatalf("new mock node: %v", err)
	}
	t.Cleanup(func() { _ = n.Close() })
	return n
}

func openStore(ctx context.Context, t *testing.T, node *ipfsnode.Node, name string, opts *Options) *GlobalStore {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "orbitdb")
	gs, err := Open(ctx, node, dir, name, opts)
	if err != nil {
		t.Fatalf("open global store: %v", err)
	}
	t.Cleanup(func() { _ = gs.Close() })
	return gs
}

func randomBytes(t *testing.T, n int) []byte {
	t.Helper()
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("random bytes: %v", err)
	}
	return b
}

func TestSingleNodeBlocks(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	mn := mocknet.New()
	defer mn.Close()
	node := newMockNode(ctx, t, mn)
	gs := openStore(ctx, t, node, "blocks-test", &Options{BlockFetchTimeout: 2 * time.Second})

	data := append([]byte("validatorium block "), randomBytes(t, 64)...)

	id, err := gs.PutBlock(ctx, data)
	if err != nil {
		t.Fatalf("PutBlock: %v", err)
	}
	want, err := storage.ContentID(data)
	if err != nil {
		t.Fatalf("ContentID: %v", err)
	}
	if id != want {
		t.Fatalf("PutBlock cid = %s, want storage.ContentID = %s", id, want)
	}

	got, err := gs.GetBlock(ctx, id)
	if err != nil {
		t.Fatalf("GetBlock: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("GetBlock bytes mismatch: got %d bytes, want %d", len(got), len(data))
	}

	// Putting the same bytes again must be a no-op that yields the same CID.
	id2, err := gs.PutBlock(ctx, data)
	if err != nil {
		t.Fatalf("PutBlock again: %v", err)
	}
	if id2 != id {
		t.Fatalf("second PutBlock cid = %s, want %s", id2, id)
	}
}

func TestSingleNodeGetBlockAbsent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	mn := mocknet.New()
	defer mn.Close()
	node := newMockNode(ctx, t, mn)
	gs := openStore(ctx, t, node, "absent-test", &Options{BlockFetchTimeout: 2 * time.Second})

	// A well-formed CID of bytes nobody has stored.
	h, err := mh.Sum(randomBytes(t, 32), mh.SHA2_256, -1)
	if err != nil {
		t.Fatalf("multihash: %v", err)
	}
	absent := cid.NewCidV1(cid.Raw, h).String()

	start := time.Now()
	_, err = gs.GetBlock(ctx, absent)
	elapsed := time.Since(start)
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("GetBlock(absent) err = %v, want storage.ErrNotFound", err)
	}
	if elapsed > 10*time.Second {
		t.Fatalf("GetBlock(absent) took %s, want bounded by the fetch timeout", elapsed)
	}

	if _, err := gs.GetBlock(ctx, "not-a-cid"); err == nil || errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("GetBlock(invalid) err = %v, want a decode error", err)
	}
}

func TestSingleNodePointers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	mn := mocknet.New()
	defer mn.Close()
	node := newMockNode(ctx, t, mn)
	gs := openStore(ctx, t, node, "pointers-test", nil)

	if gs.Address() == "" {
		t.Fatal("Address() is empty")
	}
	if gs.Identity() == "" {
		t.Fatal("Identity() is empty")
	}

	if _, err := gs.GetPointer(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("GetPointer(unknown) err = %v, want storage.ErrNotFound", err)
	}

	p := storage.Pointer{
		UUID:           "6d2a1e9c-5d8f-4a0b-9c3e-1f2a3b4c5d6e",
		CID:            "bafkreigh2akiscaildcqabsyg3dfr6chu3fgpregiymsck7e7aqa4s52zy",
		AttestationCID: "bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku",
		Status:         "current",
		UpdatedAt:      time.Now().UTC().Format(time.RFC3339),
	}
	if err := gs.PutPointer(ctx, p); err != nil {
		t.Fatalf("PutPointer: %v", err)
	}
	got, err := gs.GetPointer(ctx, p.UUID)
	if err != nil {
		t.Fatalf("GetPointer: %v", err)
	}
	if got.UUID != p.UUID || got.CID != p.CID || got.AttestationCID != p.AttestationCID ||
		got.Status != p.Status || got.UpdatedAt != p.UpdatedAt || len(got.SupersededBy) != 0 {
		t.Fatalf("GetPointer = %+v, want %+v", got, p)
	}

	// Replacing the pointer: the UUID keeps its identity, the record changes.
	p2 := p
	p2.Status = "obsolete"
	p2.SupersededBy = []string{"7e3b2f0d-6e9a-4b1c-8d4f-2a3b4c5d6e7f"}
	p2.UpdatedAt = time.Now().UTC().Add(time.Second).Format(time.RFC3339)
	if err := gs.PutPointer(ctx, p2); err != nil {
		t.Fatalf("PutPointer (replace): %v", err)
	}
	got, err = gs.GetPointer(ctx, p.UUID)
	if err != nil {
		t.Fatalf("GetPointer after replace: %v", err)
	}
	if got.Status != "obsolete" || len(got.SupersededBy) != 1 || got.SupersededBy[0] != p2.SupersededBy[0] {
		t.Fatalf("GetPointer after replace = %+v, want %+v", got, p2)
	}

	if err := gs.PutPointer(ctx, storage.Pointer{}); err == nil {
		t.Fatal("PutPointer with empty UUID succeeded, want error")
	}
}

// TestTwoNodeReplication proves that a pointer written on node A replicates
// to node B, which can then fetch the version block it names over bitswap.
// Convergence here is a transport fact, not a statement about the record.
func TestTwoNodeReplication(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping multi-node test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	mn := mocknet.New()
	defer mn.Close()

	nodeA := newMockNode(ctx, t, mn)
	nodeB := newMockNode(ctx, t, mn)

	if _, err := mn.LinkPeers(nodeA.PeerID(), nodeB.PeerID()); err != nil {
		t.Fatalf("link peers: %v", err)
	}
	if _, err := mn.ConnectPeers(nodeA.PeerID(), nodeB.PeerID()); err != nil {
		t.Fatalf("connect peers: %v", err)
	}

	// Node A owns the pointer store; node B is read-only here. Node A's write
	// access list contains only its own identity (the Open default).
	gsA := openStore(ctx, t, nodeA, "replication-test", nil)

	data := append([]byte("replicated version "), randomBytes(t, 128)...)
	id, err := gsA.PutBlock(ctx, data)
	if err != nil {
		t.Fatalf("A PutBlock: %v", err)
	}
	p := storage.Pointer{
		UUID:           "0b7d9c2e-4f61-4c8a-a3d5-9e8f7a6b5c4d",
		CID:            id,
		AttestationCID: id,
		Status:         "current",
		UpdatedAt:      time.Now().UTC().Format(time.RFC3339),
	}
	if err := gsA.PutPointer(ctx, p); err != nil {
		t.Fatalf("A PutPointer: %v", err)
	}

	// Node B opens the same store by address, with the same manifest.
	gsB := openStore(ctx, t, nodeB, gsA.Address(), &Options{
		WriteAccess:       []string{gsA.Identity()},
		BlockFetchTimeout: 20 * time.Second,
	})
	if gsB.Address() != gsA.Address() {
		t.Fatalf("B address = %s, want %s", gsB.Address(), gsA.Address())
	}

	sub, err := gsB.Docs().EventBus().Subscribe(new(stores.EventReplicated))
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Close()

	deadline := time.After(30 * time.Second)
	var got storage.Pointer
wait:
	for {
		// The event may already have fired before the subscription; poll on
		// each tick as well as on each event.
		got, err = gsB.GetPointer(ctx, p.UUID)
		if err == nil {
			break wait
		}
		if !errors.Is(err, storage.ErrNotFound) {
			t.Fatalf("B GetPointer: %v", err)
		}
		select {
		case <-sub.Out():
		case <-time.After(500 * time.Millisecond):
		case <-deadline:
			t.Fatal("timed out waiting for pointer replication")
		case <-ctx.Done():
			t.Fatal("context cancelled waiting for replication")
		}
	}

	if got.CID != id || got.UUID != p.UUID || got.Status != p.Status {
		t.Fatalf("B GetPointer = %+v, want %+v", got, p)
	}

	fetched, err := gsB.GetBlock(ctx, got.CID)
	if err != nil {
		t.Fatalf("B GetBlock over bitswap: %v", err)
	}
	if !bytes.Equal(fetched, data) {
		t.Fatalf("B GetBlock bytes mismatch")
	}
}
