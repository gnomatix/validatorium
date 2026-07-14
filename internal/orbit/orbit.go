package orbit

import (
	"context"

	"berty.tech/go-orbit-db/iface"
	"gnomatix/validatorium/internal/ipfsnode"
)

type Manager struct {
	orbitDB iface.OrbitDB
}

func NewManager(ctx context.Context, node *ipfsnode.Node, dbPath string) (*Manager, error) {
	// Skeleton struct. Future tasks will instantiate go-orbit-db.
	return &Manager{}, nil
}

func (m *Manager) Close() error {
	return nil
}
