// Package spike contains exploratory tests that validate architectural
// assumptions before committing to production code. These tests are the
// de-risk gate for the two-tier storage epic (validatorium-bqe).
package spike

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	orbitdb "berty.tech/go-orbit-db"
	"berty.tech/go-orbit-db/accesscontroller"
	"berty.tech/go-orbit-db/stores"
	"github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	cfg "github.com/ipfs/kubo/config"
	ipfsCore "github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/coreapi"
	coreiface "github.com/ipfs/kubo/core/coreiface"
	mock "github.com/ipfs/kubo/core/mock"
	"github.com/ipfs/kubo/repo"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
)

// testRepo creates an in-memory Kubo repo with pubsub enabled.
func testRepo(t *testing.T) repo.Repo {
	t.Helper()

	c := cfg.Config{}
	priv, pub, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
	if err != nil {
		t.Fatalf("generate key pair: %v", err)
	}

	pid, err := peer.IDFromPublicKey(pub)
	if err != nil {
		t.Fatalf("peer ID from public key: %v", err)
	}

	privkeyb, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
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
	}
}

// testNode creates an IPFS node on the given mock network with pubsub.
func testNode(ctx context.Context, t *testing.T, mn mocknet.Mocknet) (*ipfsCore.IpfsNode, func()) {
	t.Helper()

	core, err := ipfsCore.NewNode(ctx, &ipfsCore.BuildCfg{
		Online: true,
		Repo:   testRepo(t),
		Host:   mock.MockHostOption(mn),
		ExtraOpts: map[string]bool{
			"pubsub": true,
		},
	})
	if err != nil {
		t.Fatalf("create IPFS node: %v", err)
	}

	return core, func() { core.Close() }
}

// testCoreAPI wraps an IpfsNode in the CoreAPI interface.
func testCoreAPI(t *testing.T, node *ipfsCore.IpfsNode) coreiface.CoreAPI {
	t.Helper()

	api, err := coreapi.NewCoreAPI(node)
	if err != nil {
		t.Fatalf("create core API: %v", err)
	}
	return api
}

// TestDocumentStoreReplication is the core spike test for validatorium-bqe.2.
//
// It proves: two in-process OrbitDB nodes, each backed by an embedded IPFS
// mock node, can create a Documents store, write micropublication records on
// node A, and have them replicate automatically to node B via pubsub.
func TestDocumentStoreReplication(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping spike test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// --- set up mock network with two IPFS nodes ---
	mn := mocknet.New()
	defer mn.Close()

	node1, clean1 := testNode(ctx, t, mn)
	defer clean1()
	node2, clean2 := testNode(ctx, t, mn)
	defer clean2()

	api1 := testCoreAPI(t, node1)
	api2 := testCoreAPI(t, node2)

	// Link peers on the mock network (L2 link)
	_, err := mn.LinkPeers(node1.Identity, node2.Identity)
	if err != nil {
		t.Fatalf("link peers: %v", err)
	}

	// Connect peers (L3 connection on top of the link)
	_, err = mn.ConnectPeers(node1.Identity, node2.Identity)
	if err != nil {
		t.Fatalf("connect peers: %v", err)
	}

	// --- create OrbitDB instances ---
	dbDir1 := filepath.Join(t.TempDir(), "orbitdb1")
	dbDir2 := filepath.Join(t.TempDir(), "orbitdb2")

	odb1, err := orbitdb.NewOrbitDB(ctx, api1, &orbitdb.NewOrbitDBOptions{
		Directory: &dbDir1,
	})
	if err != nil {
		t.Fatalf("create OrbitDB 1: %v", err)
	}
	defer odb1.Close()

	odb2, err := orbitdb.NewOrbitDB(ctx, api2, &orbitdb.NewOrbitDBOptions{
		Directory: &dbDir2,
	})
	if err != nil {
		t.Fatalf("create OrbitDB 2: %v", err)
	}
	defer odb2.Close()

	// Access control: both nodes can write
	access := accesscontroller.CreateAccessControllerOptions{
		Access: map[string][]string{
			"write": {
				odb1.Identity().ID,
				odb2.Identity().ID,
			},
		},
	}

	// --- Node 1: create Documents store and write records ---
	db1, err := odb1.Docs(ctx, "micropublications-spike", &orbitdb.CreateDBOptions{
		Directory:        &dbDir1,
		AccessController: &access,
	})
	if err != nil {
		t.Fatalf("create docstore on node 1: %v", err)
	}
	defer db1.Close()

	// Write 3 micropublication-like records
	records := []map[string]interface{}{
		{
			"_id":    "premise-001",
			"claim":  "Water boils at 100°C at standard atmospheric pressure (1 atm).",
			"domain": "chemistry",
			"status": "established",
		},
		{
			"_id":    "premise-002",
			"claim":  "The speed of light in a vacuum is approximately 299,792,458 m/s.",
			"domain": "physics",
			"status": "established",
		},
		{
			"_id":    "premise-003",
			"claim":  "DNA replication is semiconservative.",
			"domain": "molecular-biology",
			"status": "established",
		},
	}

	for _, rec := range records {
		_, err := db1.Put(ctx, rec)
		if err != nil {
			t.Fatalf("put record %s on node 1: %v", rec["_id"], err)
		}
	}

	t.Logf("Node 1 wrote %d records to %s", len(records), db1.Address().String())

	// --- Node 2: open the SAME store by address, wait for replication ---
	db2, err := odb2.Docs(ctx, db1.Address().String(), &orbitdb.CreateDBOptions{
		Directory:        &dbDir2,
		AccessController: &access,
	})
	if err != nil {
		t.Fatalf("open docstore on node 2: %v", err)
	}
	defer db2.Close()

	// Subscribe to replication events
	sub, err := db2.EventBus().Subscribe(new(stores.EventReplicated))
	if err != nil {
		t.Fatalf("subscribe to replication events: %v", err)
	}
	defer sub.Close()

	// Wait for replication to converge (all 3 records)
	deadline := time.After(30 * time.Second)
	for {
		select {
		case <-sub.Out():
			// Check if all records have replicated
			allFound := true
			for _, rec := range records {
				id := rec["_id"].(string)
				docs, err := db2.Get(ctx, id, nil)
				if err != nil || len(docs) == 0 {
					allFound = false
					break
				}
			}
			if allFound {
				t.Log("All records replicated to node 2")
				goto verifyReplication
			}
		case <-deadline:
			t.Fatal("timed out waiting for replication")
		case <-ctx.Done():
			t.Fatal("context cancelled waiting for replication")
		}
	}

verifyReplication:
	// --- Verify: node 2 has all records with correct content ---
	for _, rec := range records {
		id := rec["_id"].(string)
		docs, err := db2.Get(ctx, id, nil)
		if err != nil {
			t.Errorf("get %s from node 2: %v", id, err)
			continue
		}
		if len(docs) != 1 {
			t.Errorf("expected 1 doc for %s, got %d", id, len(docs))
			continue
		}
		doc, ok := docs[0].(map[string]interface{})
		if !ok {
			t.Errorf("doc %s is not a map", id)
			continue
		}
		if doc["claim"] != rec["claim"] {
			t.Errorf("doc %s claim mismatch: got %q, want %q", id, doc["claim"], rec["claim"])
		}
		if doc["domain"] != rec["domain"] {
			t.Errorf("doc %s domain mismatch: got %q, want %q", id, doc["domain"], rec["domain"])
		}
		t.Logf("✓ %s: %s", id, doc["claim"])
	}
}

// TestDocumentStorePartitionReconvergence proves that after a network
// partition, nodes reconverge when reconnected. This directly tests the
// CRDT merge semantics of go-orbit-db.
func TestDocumentStorePartitionReconvergence(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping spike test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	mn := mocknet.New()
	defer mn.Close()

	node1, clean1 := testNode(ctx, t, mn)
	defer clean1()
	node2, clean2 := testNode(ctx, t, mn)
	defer clean2()

	api1 := testCoreAPI(t, node1)
	api2 := testCoreAPI(t, node2)

	// Link and connect
	_, err := mn.LinkPeers(node1.Identity, node2.Identity)
	if err != nil {
		t.Fatalf("link peers: %v", err)
	}
	conn, err := mn.ConnectPeers(node1.Identity, node2.Identity)
	if err != nil {
		t.Fatalf("connect peers: %v", err)
	}

	dbDir1 := filepath.Join(t.TempDir(), "orbitdb1")
	dbDir2 := filepath.Join(t.TempDir(), "orbitdb2")

	odb1, err := orbitdb.NewOrbitDB(ctx, api1, &orbitdb.NewOrbitDBOptions{Directory: &dbDir1})
	if err != nil {
		t.Fatalf("create OrbitDB 1: %v", err)
	}
	defer odb1.Close()

	odb2, err := orbitdb.NewOrbitDB(ctx, api2, &orbitdb.NewOrbitDBOptions{Directory: &dbDir2})
	if err != nil {
		t.Fatalf("create OrbitDB 2: %v", err)
	}
	defer odb2.Close()

	access := accesscontroller.CreateAccessControllerOptions{
		Access: map[string][]string{
			"write": {
				odb1.Identity().ID,
				odb2.Identity().ID,
			},
		},
	}

	// Phase 1: Create store on node 1, write initial record
	db1, err := odb1.Docs(ctx, "partition-test", &orbitdb.CreateDBOptions{
		Directory:        &dbDir1,
		AccessController: &access,
	})
	if err != nil {
		t.Fatalf("create docstore node 1: %v", err)
	}
	defer db1.Close()

	_, err = db1.Put(ctx, map[string]interface{}{
		"_id":   "shared-premise",
		"claim": "Initial claim before partition",
	})
	if err != nil {
		t.Fatalf("put initial record: %v", err)
	}

	// Node 2 opens the same store and waits for initial sync
	db2, err := odb2.Docs(ctx, db1.Address().String(), &orbitdb.CreateDBOptions{
		Directory:        &dbDir2,
		AccessController: &access,
	})
	if err != nil {
		t.Fatalf("open docstore node 2: %v", err)
	}
	defer db2.Close()

	// Wait for initial replication
	sub, err := db2.EventBus().Subscribe(new(stores.EventReplicated))
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	select {
	case <-sub.Out():
		t.Log("Phase 1: Initial record replicated")
	case <-time.After(15 * time.Second):
		t.Fatal("Phase 1: timed out waiting for initial replication")
	}
	sub.Close()

	// Phase 2: PARTITION — disconnect the peers
	err = conn.Close()
	if err != nil {
		t.Fatalf("disconnect peers: %v", err)
	}
	// Also unlink to prevent auto-reconnect
	err = mn.UnlinkPeers(node1.Identity, node2.Identity)
	if err != nil {
		t.Fatalf("unlink peers: %v", err)
	}

	t.Log("Phase 2: Network partitioned")

	// Node 1 writes during partition
	_, err = db1.Put(ctx, map[string]interface{}{
		"_id":    "partition-premise-a",
		"claim":  "Written by node 1 during partition",
		"author": "node1",
	})
	if err != nil {
		t.Fatalf("node 1 put during partition: %v", err)
	}

	// Node 2 writes during partition (different key — no conflict)
	_, err = db2.Put(ctx, map[string]interface{}{
		"_id":    "partition-premise-b",
		"claim":  "Written by node 2 during partition",
		"author": "node2",
	})
	if err != nil {
		t.Fatalf("node 2 put during partition: %v", err)
	}

	t.Log("Phase 2: Both nodes wrote independently during partition")

	// Phase 3: RECONVERGE — reconnect peers
	_, err = mn.LinkPeers(node1.Identity, node2.Identity)
	if err != nil {
		t.Fatalf("re-link peers: %v", err)
	}
	_, err = mn.ConnectPeers(node1.Identity, node2.Identity)
	if err != nil {
		t.Fatalf("reconnect peers: %v", err)
	}

	t.Log("Phase 3: Network reconnected, waiting for reconvergence...")

	// Allow pubsub/swarm mesh to stabilize
	time.Sleep(2 * time.Second)

	// Subscribe to replication events on both nodes
	sub1, err := db1.EventBus().Subscribe(new(stores.EventReplicated))
	if err != nil {
		t.Fatalf("subscribe db1: %v", err)
	}
	defer sub1.Close()

	sub2, err := db2.EventBus().Subscribe(new(stores.EventReplicated))
	if err != nil {
		t.Fatalf("subscribe db2: %v", err)
	}
	defer sub2.Close()

	// Write post-reconnection records to trigger synchronization of partition history
	_, err = db1.Put(ctx, map[string]interface{}{
		"_id":   "sync-trigger-a",
		"claim": "Sync trigger from node 1",
	})
	if err != nil {
		t.Fatalf("node 1 put sync trigger: %v", err)
	}

	_, err = db2.Put(ctx, map[string]interface{}{
		"_id":   "sync-trigger-b",
		"claim": "Sync trigger from node 2",
	})
	if err != nil {
		t.Fatalf("node 2 put sync trigger: %v", err)
	}

	// Wait for both nodes to converge
	deadline := time.After(30 * time.Second)
	converged := false
	for !converged {
		select {
		case <-sub1.Out():
		case <-sub2.Out():
		case <-deadline:
			t.Log("Warning: reconvergence deadline reached, checking state anyway")
			converged = true
		}

		// Check if both nodes have all records (including triggers)
		docsA1, _ := db1.Get(ctx, "partition-premise-a", nil)
		docsB1, _ := db1.Get(ctx, "partition-premise-b", nil)
		docsA2, _ := db2.Get(ctx, "partition-premise-a", nil)
		docsB2, _ := db2.Get(ctx, "partition-premise-b", nil)
		docsTA1, _ := db1.Get(ctx, "sync-trigger-a", nil)
		docsTB1, _ := db1.Get(ctx, "sync-trigger-b", nil)
		docsTA2, _ := db2.Get(ctx, "sync-trigger-a", nil)
		docsTB2, _ := db2.Get(ctx, "sync-trigger-b", nil)

		if len(docsA1) > 0 && len(docsB1) > 0 && len(docsA2) > 0 && len(docsB2) > 0 &&
			len(docsTA1) > 0 && len(docsTB1) > 0 && len(docsTA2) > 0 && len(docsTB2) > 0 {
			converged = true
		}
	}

	// Verify: both nodes have all records
	expectedIDs := []string{"shared-premise", "partition-premise-a", "partition-premise-b", "sync-trigger-a", "sync-trigger-b"}
	for _, id := range expectedIDs {
		docs1, err := db1.Get(ctx, id, nil)
		if err != nil || len(docs1) == 0 {
			t.Errorf("node 1 missing %s (err=%v, count=%d)", id, err, len(docs1))
		} else {
			t.Logf("✓ node 1 has %s", id)
		}

		docs2, err := db2.Get(ctx, id, nil)
		if err != nil || len(docs2) == 0 {
			t.Errorf("node 2 missing %s (err=%v, count=%d)", id, err, len(docs2))
		} else {
			t.Logf("✓ node 2 has %s", id)
		}
	}

	// Verify content matches across nodes
	for _, id := range expectedIDs {
		docs1, _ := db1.Get(ctx, id, nil)
		docs2, _ := db2.Get(ctx, id, nil)
		if len(docs1) > 0 && len(docs2) > 0 {
			d1, _ := docs1[0].(map[string]interface{})
			d2, _ := docs2[0].(map[string]interface{})
			if d1["claim"] != d2["claim"] {
				t.Errorf("%s: content diverged: node1=%q node2=%q", id, d1["claim"], d2["claim"])
			} else {
				t.Logf("✓ %s content matches across nodes", id)
			}
		}
	}
}

// TestDocumentStoreLocalCRUD verifies basic Document store CRUD operations
// on a single node to sanity-check the API before testing replication.
func TestDocumentStoreLocalCRUD(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mn := mocknet.New()
	defer mn.Close()

	node, clean := testNode(ctx, t, mn)
	defer clean()
	api := testCoreAPI(t, node)

	dbDir := filepath.Join(t.TempDir(), "orbitdb")
	odb, err := orbitdb.NewOrbitDB(ctx, api, &orbitdb.NewOrbitDBOptions{
		Directory: &dbDir,
	})
	if err != nil {
		t.Fatalf("create OrbitDB: %v", err)
	}
	defer odb.Close()

	db, err := odb.Docs(ctx, "crud-test", nil)
	if err != nil {
		t.Fatalf("create docstore: %v", err)
	}
	defer db.Close()

	// PUT
	doc := map[string]interface{}{
		"_id":    "test-uuid-001",
		"claim":  "Test premise for CRUD",
		"domain": "testing",
	}
	_, err = db.Put(ctx, doc)
	if err != nil {
		t.Fatalf("put: %v", err)
	}

	// GET
	results, err := db.Get(ctx, "test-uuid-001", nil)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	got, ok := results[0].(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}
	if got["claim"] != "Test premise for CRUD" {
		t.Errorf("claim mismatch: got %q", got["claim"])
	}

	// UPDATE (put with same _id)
	doc["claim"] = "Updated premise for CRUD"
	_, err = db.Put(ctx, doc)
	if err != nil {
		t.Fatalf("update put: %v", err)
	}
	results, err = db.Get(ctx, "test-uuid-001", nil)
	if err != nil {
		t.Fatalf("get after update: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result after update, got %d", len(results))
	}
	got = results[0].(map[string]interface{})
	if got["claim"] != "Updated premise for CRUD" {
		t.Errorf("claim after update: got %q", got["claim"])
	}

	// DELETE
	_, err = db.Delete(ctx, "test-uuid-001")
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	results, err = db.Get(ctx, "test-uuid-001", nil)
	if err != nil {
		t.Fatalf("get after delete: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results after delete, got %d", len(results))
	}

	// QUERY
	testDocs := []map[string]interface{}{
		{"_id": "q1", "domain": "physics", "claim": "F = ma"},
		{"_id": "q2", "domain": "biology", "claim": "DNA is double-helix"},
		{"_id": "q3", "domain": "physics", "claim": "E = mc²"},
	}
	for _, d := range testDocs {
		_, err := db.Put(ctx, d)
		if err != nil {
			t.Fatalf("put %s: %v", d["_id"], err)
		}
	}

	// Query: find all physics premises
	physDocs, err := db.Query(ctx, func(doc interface{}) (bool, error) {
		m, ok := doc.(map[string]interface{})
		if !ok {
			return false, nil
		}
		return m["domain"] == "physics", nil
	})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(physDocs) != 2 {
		t.Errorf("expected 2 physics docs, got %d", len(physDocs))
	}
	for _, d := range physDocs {
		m := d.(map[string]interface{})
		t.Logf("✓ query result: %s — %s", m["_id"], m["claim"])
	}

	fmt.Fprintf(os.Stderr, "CRUD test passed: put, get, update, delete, query all working\n")
}
