// Package memory provides an in-process storage.GlobalStore used by tests and
// by the promotion seam's own tests. It has the same contract as the
// go-orbit-db implementation (content-addressed immutable blocks plus
// UUID-keyed pointer records) but no replication and no persistence.
package memory

import (
	"context"
	"errors"
	"sort"
	"sync"

	"gnomatix/validatorium/internal/namespace"
	"gnomatix/validatorium/internal/storage"
)

// GlobalStore is an in-memory storage.GlobalStore.
type GlobalStore struct {
	mu         sync.RWMutex
	blocks     map[string][]byte
	pointers   map[string]storage.Pointer
	namespaces map[namespace.Name]map[string]storage.NamespacedPointer
	closed     bool
}

var _ storage.GlobalStore = (*GlobalStore)(nil)

// NewGlobalStore returns an empty in-memory global store.
func NewGlobalStore() *GlobalStore {
	return &GlobalStore{
		blocks:     map[string][]byte{},
		pointers:   map[string]storage.Pointer{},
		namespaces: map[namespace.Name]map[string]storage.NamespacedPointer{},
	}
}

func (g *GlobalStore) PutNamespacedPointer(ctx context.Context, p storage.NamespacedPointer) error {
	if _, err := namespace.Parse(string(p.Namespace)); err != nil {
		return err
	}
	if p.UUID == "" || p.CID == "" {
		return errors.New("memory: namespaced pointer needs UUID and CID")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	ns, ok := g.namespaces[p.Namespace]
	if !ok {
		ns = map[string]storage.NamespacedPointer{}
		g.namespaces[p.Namespace] = ns
	}
	ns[p.UUID] = p
	return nil
}

func (g *GlobalStore) GetNamespacedPointer(ctx context.Context, ns namespace.Name, uuid string) (storage.NamespacedPointer, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	p, ok := g.namespaces[ns][uuid]
	if !ok {
		return storage.NamespacedPointer{}, storage.ErrNotFound
	}
	return p, nil
}

func (g *GlobalStore) ListNamespace(ctx context.Context, ns namespace.Name) ([]storage.NamespacedPointer, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]storage.NamespacedPointer, 0, len(g.namespaces[ns]))
	for _, p := range g.namespaces[ns] {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UUID < out[j].UUID })
	return out, nil
}

func (g *GlobalStore) DeleteNamespacedPointer(ctx context.Context, ns namespace.Name, uuid string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.namespaces[ns], uuid)
	return nil
}

func (g *GlobalStore) PutBlock(ctx context.Context, data []byte) (string, error) {
	id, err := storage.ContentID(data)
	if err != nil {
		return "", err
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	g.mu.Lock()
	g.blocks[id] = cp
	g.mu.Unlock()
	return id, nil
}

func (g *GlobalStore) GetBlock(ctx context.Context, cid string) ([]byte, error) {
	g.mu.RLock()
	data, ok := g.blocks[cid]
	g.mu.RUnlock()
	if !ok {
		return nil, storage.ErrNotFound
	}
	if err := storage.VerifyContentID(cid, data); err != nil {
		return nil, err
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	return cp, nil
}

func (g *GlobalStore) PutPointer(ctx context.Context, p storage.Pointer) error {
	g.mu.Lock()
	g.pointers[p.UUID] = p
	g.mu.Unlock()
	return nil
}

func (g *GlobalStore) GetPointer(ctx context.Context, uuid string) (storage.Pointer, error) {
	g.mu.RLock()
	p, ok := g.pointers[uuid]
	g.mu.RUnlock()
	if !ok {
		return storage.Pointer{}, storage.ErrNotFound
	}
	return p, nil
}

// BlockCount reports how many distinct blocks are stored (for tests).
func (g *GlobalStore) BlockCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.blocks)
}

func (g *GlobalStore) Close() error {
	g.mu.Lock()
	g.closed = true
	g.mu.Unlock()
	return nil
}
