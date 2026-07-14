package ipfsnode

import (
	"context"
)

type Node struct {
	// Skeleton struct for embedded IPFS node.
	// Future tasks will wrap Kubo's core.IpfsNode or coreapi.CoreAPI.
}

func NewNode(ctx context.Context, repoPath string) (*Node, error) {
	return &Node{}, nil
}

func (n *Node) Close() error {
	return nil
}
