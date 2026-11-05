package ipfsnode

import (
	"context"
	"testing"
	"time"

	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
)

func TestNewMockNode(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mn := mocknet.New()
	defer mn.Close()

	n, err := NewMockNode(ctx, mn)
	if err != nil {
		t.Fatalf("NewMockNode: %v", err)
	}
	if n.API() == nil {
		t.Fatal("API() is nil")
	}
	if n.Core() == nil {
		t.Fatal("Core() is nil")
	}
	if n.PeerID() == "" {
		t.Fatal("PeerID() is empty")
	}
	if len(mn.Peers()) != 1 {
		t.Fatalf("mocknet has %d peers, want 1", len(mn.Peers()))
	}
	if err := n.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	var nilNode *Node
	if err := nilNode.Close(); err != nil {
		t.Fatalf("Close on nil node: %v", err)
	}
}

func TestNewMockNodeNilMocknet(t *testing.T) {
	if _, err := NewMockNode(context.Background(), nil); err == nil {
		t.Fatal("NewMockNode(nil) succeeded, want error")
	}
}

func TestNewNodeEmptyPath(t *testing.T) {
	if _, err := NewNode(context.Background(), ""); err == nil {
		t.Fatal("NewNode(\"\") succeeded, want error")
	}
}
