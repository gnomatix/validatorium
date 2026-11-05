// Package memory provides an in-process storage.GlobalStore used by tests and
// by the promotion seam's own tests. It has the same contract as the
// go-orbit-db implementation (content-addressed immutable blocks plus
// UUID-keyed pointer records) but no replication and no persistence.
package memory

import (
	"context"
	"sync"

	"gnomatix/validatorium/internal/storage"
)

// GlobalStore is an in-memory storage.GlobalStore.
type GlobalStore struct {
	mu       sync.RWMutex
	blocks   map[string][]byte
	pointers map[string]storage.Pointer
	closed   bool
}

var _ storage.GlobalStore = (*GlobalStore)(nil)

// NewGlobalStore returns an empty in-memory global store.
func NewGlobalStore() *GlobalStore {
	return &GlobalStore{
		blocks:   map[string][]byte{},
		pointers: map[string]storage.Pointer{},
	}
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
