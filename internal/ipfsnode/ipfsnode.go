// Package ipfsnode wraps an embedded Kubo node so that the tier-2 global store
// (package orbit) can address immutable blocks and replicate pointer records
// without owning the lifecycle of the IPFS node itself.
//
// This package moves bytes around. It does not decide whether any record is
// true, current, or admissible; that is the job of the attestation and
// promotion layers above the storage boundary.
package ipfsnode

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	cfg "github.com/ipfs/kubo/config"
	ipfsCore "github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/coreapi"
	coreiface "github.com/ipfs/kubo/core/coreiface"
	mock "github.com/ipfs/kubo/core/mock"
	"github.com/ipfs/kubo/plugin/loader"
	"github.com/ipfs/kubo/repo"
	"github.com/ipfs/kubo/repo/fsrepo"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
)

// Node is an embedded Kubo node together with its CoreAPI handle.
type Node struct {
	core *ipfsCore.IpfsNode
	api  coreiface.CoreAPI
}

// API returns the CoreAPI view of the node.
func (n *Node) API() coreiface.CoreAPI { return n.api }

// Core returns the underlying Kubo node. Callers that only need the public
// IPFS surface should prefer API.
func (n *Node) Core() *ipfsCore.IpfsNode { return n.core }

// PeerID returns the libp2p peer identity of the node.
func (n *Node) PeerID() peer.ID { return n.core.Identity }

// Close shuts the node down. It is safe to call on a nil receiver.
func (n *Node) Close() error {
	if n == nil || n.core == nil {
		return nil
	}
	return n.core.Close()
}

// NewMockNode builds an online, pubsub-enabled node whose libp2p host lives on
// the supplied in-process mock network. The repo is in-memory, no bootstrap
// peers are configured, and the resource manager is disabled, so a test can
// link several nodes on one Mocknet and exercise replication without touching
// the real network.
func NewMockNode(ctx context.Context, mn mocknet.Mocknet) (*Node, error) {
	if mn == nil {
		return nil, errors.New("ipfsnode: nil mocknet")
	}
	r, err := mockRepo()
	if err != nil {
		return nil, err
	}
	return build(ctx, &ipfsCore.BuildCfg{
		Online:    true,
		Repo:      r,
		Host:      mock.MockHostOption(mn),
		ExtraOpts: map[string]bool{"pubsub": true},
	})
}

// NewNode opens (initialising it first when needed) the on-disk Kubo
// repository at repoPath and starts an online, pubsub-enabled node on the
// real network. Datastore plugins are loaded once per process.
func NewNode(ctx context.Context, repoPath string) (*Node, error) {
	if repoPath == "" {
		return nil, errors.New("ipfsnode: empty repo path")
	}
	if err := loadPlugins(repoPath); err != nil {
		return nil, err
	}
	if !fsrepo.IsInitialized(repoPath) {
		c, err := cfg.Init(io.Discard, 2048)
		if err != nil {
			return nil, fmt.Errorf("ipfsnode: init config: %w", err)
		}
		c.Pubsub.Enabled = cfg.True
		if err := fsrepo.Init(repoPath, c); err != nil {
			return nil, fmt.Errorf("ipfsnode: init repo %q: %w", repoPath, err)
		}
	}
	r, err := fsrepo.Open(repoPath)
	if err != nil {
		return nil, fmt.Errorf("ipfsnode: open repo %q: %w", repoPath, err)
	}
	return build(ctx, &ipfsCore.BuildCfg{
		Online:    true,
		Repo:      r,
		ExtraOpts: map[string]bool{"pubsub": true},
	})
}

func build(ctx context.Context, bc *ipfsCore.BuildCfg) (*Node, error) {
	core, err := ipfsCore.NewNode(ctx, bc)
	if err != nil {
		return nil, fmt.Errorf("ipfsnode: create node: %w", err)
	}
	api, err := coreapi.NewCoreAPI(core)
	if err != nil {
		_ = core.Close()
		return nil, fmt.Errorf("ipfsnode: create core api: %w", err)
	}
	return &Node{core: core, api: api}, nil
}

// mockRepo returns an in-memory repo with a fresh identity, pubsub enabled,
// no bootstrap peers and the resource manager disabled.
func mockRepo() (repo.Repo, error) {
	c := cfg.Config{}
	priv, pub, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("ipfsnode: generate key pair: %w", err)
	}
	pid, err := peer.IDFromPublicKey(pub)
	if err != nil {
		return nil, fmt.Errorf("ipfsnode: peer id: %w", err)
	}
	privkeyb, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("ipfsnode: marshal private key: %w", err)
	}

	c.Pubsub.Enabled = cfg.True
	c.Bootstrap = []string{}
	c.Addresses.Swarm = []string{"/ip4/127.0.0.1/tcp/4001", "/ip4/127.0.0.1/udp/4001/quic"}
	c.Identity.PeerID = pid.String()
	c.Identity.PrivKey = base64.StdEncoding.EncodeToString(privkeyb)
	c.Swarm.ResourceMgr.Enabled = cfg.False

	return &repo.Mock{
		D: dsync.MutexWrap(datastore.NewMapDatastore()),
		C: c,
	}, nil
}

var (
	pluginsOnce sync.Once
	pluginsErr  error
)

// loadPlugins registers Kubo's preloaded plugins (datastore backends among
// them) exactly once per process; fsrepo cannot open a repo without them.
func loadPlugins(repoPath string) error {
	pluginsOnce.Do(func() {
		pl, err := loader.NewPluginLoader(repoPath)
		if err != nil {
			pluginsErr = fmt.Errorf("ipfsnode: plugin loader: %w", err)
			return
		}
		if err := pl.Initialize(); err != nil {
			pluginsErr = fmt.Errorf("ipfsnode: initialize plugins: %w", err)
			return
		}
		if err := pl.Inject(); err != nil {
			pluginsErr = fmt.Errorf("ipfsnode: inject plugins: %w", err)
			return
		}
	})
	return pluginsErr
}
