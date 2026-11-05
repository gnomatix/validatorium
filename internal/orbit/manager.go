package orbit

import (
	"context"

	"gnomatix/validatorium/internal/ipfsnode"
)

// DefaultPointerStoreName is the pointer store a Manager opens when the
// caller gives no address.
const DefaultPointerStoreName = "validatorium-pointers"

// Manager is a thin compatibility wrapper kept so existing callers that
// construct the global store by directory alone keep compiling. New code
// should call Open directly.
//
// Deprecated: use Open.
type Manager struct {
	*GlobalStore
}

// NewManager opens a GlobalStore under dbPath with the default pointer store
// name and the opening identity as sole writer.
//
// Deprecated: use Open.
func NewManager(ctx context.Context, node *ipfsnode.Node, dbPath string) (*Manager, error) {
	gs, err := Open(ctx, node, dbPath, DefaultPointerStoreName, nil)
	if err != nil {
		return nil, err
	}
	return &Manager{GlobalStore: gs}, nil
}
