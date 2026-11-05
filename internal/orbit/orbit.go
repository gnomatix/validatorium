// Package orbit is the tier-2 global store of Validatorium: content-addressed
// immutable blocks held by an embedded Kubo node, plus a go-orbit-db Documents
// store of replicated, UUID-keyed pointer records.
//
// Identity model. A UUID is the identity of a record; a CID identifies one
// immutable version of it. A pointer names the version currently associated
// with a UUID. Replacing a pointer never deletes the previous version's bytes,
// and an obsolete pointer remains addressable so its history can be followed.
//
// Storage never decides truth. Nothing in this package validates, votes on, or
// reconciles the meaning of what it stores. Replication is a transport
// property: when two peers converge on the same pointer record it means the
// bytes have propagated, not that the record is correct. Whether a version
// should have been published at all is settled before it reaches this layer,
// by the attestation and promotion packages.
package orbit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	orbitdb "berty.tech/go-orbit-db"
	"berty.tech/go-orbit-db/accesscontroller"
	"berty.tech/go-orbit-db/iface"
	"github.com/ipfs/boxo/path"
	"github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"
	coreiface "github.com/ipfs/kubo/core/coreiface"
	"github.com/ipfs/kubo/core/coreiface/options"
	mh "github.com/multiformats/go-multihash"

	"gnomatix/validatorium/internal/ipfsnode"
	"gnomatix/validatorium/internal/storage"
)

// DefaultBlockFetchTimeout bounds how long GetBlock waits for a block that is
// not held locally before reporting storage.ErrNotFound. Bitswap otherwise
// keeps asking the network until the context ends.
const DefaultBlockFetchTimeout = 10 * time.Second

// Options tunes how a GlobalStore is opened.
type Options struct {
	// WriteAccess lists the OrbitDB identity IDs allowed to write to the
	// pointer store. When empty, only the opening identity may write. The
	// list is part of the store manifest, so every peer that opens the same
	// address must pass the same list (or open by address and let the
	// manifest supply it).
	WriteAccess []string
	// BlockFetchTimeout overrides DefaultBlockFetchTimeout when positive.
	BlockFetchTimeout time.Duration
	// PinBlocks pins every block written through PutBlock so garbage
	// collection on the IPFS node keeps it.
	PinBlocks bool
}

// GlobalStore implements storage.GlobalStore over Kubo and go-orbit-db.
type GlobalStore struct {
	api     coreiface.CoreAPI
	odb     iface.OrbitDB
	docs    iface.DocumentStore
	fetchTO time.Duration
	pin     bool
	closeMu sync.Mutex
	closed  bool
}

var _ storage.GlobalStore = (*GlobalStore)(nil)

// Open creates an OrbitDB instance on top of node, with its cache and
// keystore under dir, and opens the pointer Documents store named by
// dbNameOrAddress. A plain name creates (or reopens) a store owned by this
// identity; an OrbitDB address (/orbitdb/...) opens an existing store so the
// pointer records replicate from whichever peers already hold it.
//
// The caller keeps ownership of node: Close releases the store and the
// OrbitDB instance but leaves the IPFS node running.
func Open(ctx context.Context, node *ipfsnode.Node, dir string, dbNameOrAddress string, opts *Options) (*GlobalStore, error) {
	if node == nil {
		return nil, errors.New("orbit: nil ipfs node")
	}
	if dbNameOrAddress == "" {
		return nil, errors.New("orbit: empty store name or address")
	}
	if opts == nil {
		opts = &Options{}
	}

	api := node.API()
	odb, err := orbitdb.NewOrbitDB(ctx, api, &orbitdb.NewOrbitDBOptions{Directory: &dir})
	if err != nil {
		return nil, fmt.Errorf("orbit: create orbitdb: %w", err)
	}

	write := opts.WriteAccess
	if len(write) == 0 {
		write = []string{odb.Identity().ID}
	}
	access := &accesscontroller.CreateAccessControllerOptions{
		Access: map[string][]string{"write": write},
	}

	docs, err := odb.Docs(ctx, dbNameOrAddress, &orbitdb.CreateDBOptions{
		Directory:        &dir,
		AccessController: access,
	})
	if err != nil {
		_ = odb.Close()
		return nil, fmt.Errorf("orbit: open pointer store %q: %w", dbNameOrAddress, err)
	}

	to := opts.BlockFetchTimeout
	if to <= 0 {
		to = DefaultBlockFetchTimeout
	}

	return &GlobalStore{
		api:     api,
		odb:     odb,
		docs:    docs,
		fetchTO: to,
		pin:     opts.PinBlocks,
	}, nil
}

// Address returns the OrbitDB address of the pointer store. Another node
// passes this string to Open to replicate the same pointer records.
func (g *GlobalStore) Address() string { return g.docs.Address().String() }

// Identity returns the OrbitDB identity ID this store writes with. Other
// peers list it in Options.WriteAccess to let this node write pointers.
func (g *GlobalStore) Identity() string { return g.odb.Identity().ID }

// Docs exposes the underlying Documents store, for callers that need to
// subscribe to its event bus (for example to wait for replication in tests).
func (g *GlobalStore) Docs() iface.DocumentStore { return g.docs }

// PutBlock stores data as a raw IPFS block hashed with SHA2-256 and returns
// its CIDv1 string, which equals storage.ContentID(data). Putting the same
// bytes twice yields the same CID and is harmless.
func (g *GlobalStore) PutBlock(ctx context.Context, data []byte) (string, error) {
	stat, err := g.api.Block().Put(ctx, bytes.NewReader(data),
		options.Block.CidCodec("raw"),
		options.Block.Hash(mh.SHA2_256, -1),
		options.Block.Pin(g.pin),
	)
	if err != nil {
		return "", fmt.Errorf("orbit: put block: %w", err)
	}
	id := stat.Path().RootCid().String()
	if err := storage.VerifyContentID(id, data); err != nil {
		return "", fmt.Errorf("orbit: put block returned unexpected cid: %w", err)
	}
	return id, nil
}

// GetBlock returns the bytes stored under id, fetched locally or from
// connected peers, and verifies that they hash to id. A block that cannot be
// found within the fetch timeout yields storage.ErrNotFound.
func (g *GlobalStore) GetBlock(ctx context.Context, id string) ([]byte, error) {
	c, err := cid.Decode(id)
	if err != nil {
		return nil, fmt.Errorf("orbit: decode cid %q: %w", id, err)
	}

	fetchCtx, cancel := context.WithTimeout(ctx, g.fetchTO)
	defer cancel()

	r, err := g.api.Block().Get(fetchCtx, path.FromCid(c))
	if err != nil {
		return nil, g.classifyGetErr(ctx, err)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, g.classifyGetErr(ctx, fmt.Errorf("orbit: read block: %w", err))
	}
	if err := storage.VerifyContentID(id, data); err != nil {
		return nil, fmt.Errorf("orbit: block %s failed verification: %w", id, err)
	}
	return data, nil
}

// classifyGetErr maps an IPFS not-found error, or a fetch timeout that fired
// while the caller's own context was still live, onto storage.ErrNotFound.
func (g *GlobalStore) classifyGetErr(parent context.Context, err error) error {
	if ipld.IsNotFound(err) {
		return storage.ErrNotFound
	}
	if errors.Is(err, context.DeadlineExceeded) && parent.Err() == nil {
		return storage.ErrNotFound
	}
	return fmt.Errorf("orbit: get block: %w", err)
}

// PutPointer stores or replaces the pointer record for p.UUID.
func (g *GlobalStore) PutPointer(ctx context.Context, p storage.Pointer) error {
	if p.UUID == "" {
		return errors.New("orbit: pointer has empty uuid")
	}
	doc, err := pointerToDoc(p)
	if err != nil {
		return err
	}
	if _, err := g.docs.Put(ctx, doc); err != nil {
		return fmt.Errorf("orbit: put pointer %s: %w", p.UUID, err)
	}
	return nil
}

// GetPointer returns the pointer record for uuid or storage.ErrNotFound.
func (g *GlobalStore) GetPointer(ctx context.Context, uuid string) (storage.Pointer, error) {
	res, err := g.docs.Get(ctx, uuid, nil)
	if err != nil {
		return storage.Pointer{}, fmt.Errorf("orbit: get pointer %s: %w", uuid, err)
	}
	if len(res) == 0 {
		return storage.Pointer{}, storage.ErrNotFound
	}
	p, err := docToPointer(res[0])
	if err != nil {
		return storage.Pointer{}, fmt.Errorf("orbit: decode pointer %s: %w", uuid, err)
	}
	return p, nil
}

// Close closes the pointer store and the OrbitDB instance. The IPFS node is
// left running for its owner to close. Close is idempotent.
func (g *GlobalStore) Close() error {
	g.closeMu.Lock()
	defer g.closeMu.Unlock()
	if g.closed {
		return nil
	}
	g.closed = true
	var errs []error
	if err := g.docs.Close(); err != nil {
		errs = append(errs, fmt.Errorf("orbit: close pointer store: %w", err))
	}
	if err := g.odb.Close(); err != nil {
		errs = append(errs, fmt.Errorf("orbit: close orbitdb: %w", err))
	}
	return errors.Join(errs...)
}

// pointerToDoc converts a Pointer into the map form the Documents store
// expects, via its JSON encoding so the "_id" key and field names match the
// struct tags exactly.
func pointerToDoc(p storage.Pointer) (map[string]interface{}, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("orbit: marshal pointer: %w", err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("orbit: pointer to document: %w", err)
	}
	return doc, nil
}

// docToPointer converts a Documents-store value back into a Pointer.
func docToPointer(doc interface{}) (storage.Pointer, error) {
	b, err := json.Marshal(doc)
	if err != nil {
		return storage.Pointer{}, fmt.Errorf("orbit: marshal document: %w", err)
	}
	var p storage.Pointer
	if err := json.Unmarshal(b, &p); err != nil {
		return storage.Pointer{}, fmt.Errorf("orbit: document to pointer: %w", err)
	}
	return p, nil
}
