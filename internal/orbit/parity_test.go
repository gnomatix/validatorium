package orbit

// Capability checks for the vendored go-orbit-db (v1.22.2) behind
// validatorium-bqe.3: which OrbitDB features the Go port has, tested.

import (
	"bytes"
	"context"
	"encoding/base64"
	"path/filepath"
	"testing"
	"time"

	orbitdb "berty.tech/go-orbit-db"
	"berty.tech/go-orbit-db/accesscontroller"
	"berty.tech/go-orbit-db/iface"
	"github.com/ipfs/boxo/path"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
)

func newODB(ctx context.Context, t *testing.T) (iface.OrbitDB, string) {
	t.Helper()
	mn := mocknet.New()
	t.Cleanup(func() { _ = mn.Close() })
	node := newMockNode(ctx, t, mn)
	dir := filepath.Join(t.TempDir(), "orbitdb")
	odb, err := orbitdb.NewOrbitDB(ctx, node.API(), &orbitdb.NewOrbitDBOptions{Directory: &dir})
	if err != nil {
		t.Fatalf("new orbitdb: %v", err)
	}
	t.Cleanup(func() { _ = odb.Close() })
	return odb, dir
}

func TestParityKeyValueStore(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	odb, dir := newODB(ctx, t)
	kv, err := odb.KeyValue(ctx, "parity-kv", &orbitdb.CreateDBOptions{Directory: &dir})
	if err != nil {
		t.Fatalf("open keyvalue: %v", err)
	}
	if _, err := kv.Put(ctx, "k", []byte("v1")); err != nil {
		t.Fatalf("put: %v", err)
	}
	got, err := kv.Get(ctx, "k")
	if err != nil || string(got) != "v1" {
		t.Fatalf("get = %q, %v; want v1", got, err)
	}
}

func TestParityEventLogStoreAndPlaintextPayload(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	odb, dir := newODB(ctx, t)
	lg, err := odb.Log(ctx, "parity-log", &orbitdb.CreateDBOptions{Directory: &dir})
	if err != nil {
		t.Fatalf("open eventlog: %v", err)
	}
	marker := []byte("validatorium-parity-marker-7f3a")
	op, err := lg.Add(ctx, marker)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	ops, err := lg.List(ctx, &iface.StreamOptions{})
	if err != nil || len(ops) != 1 || !bytes.Equal(ops[0].GetValue(), marker) {
		t.Fatalf("list = %d ops, %v", len(ops), err)
	}
	// The log entry block, as any replicating peer receives it, carries the
	// payload readable: go-orbit-db has no payload encryption option.
	blk, err := odb.IPFS().Block().Get(ctx, path.FromCid(op.GetEntry().GetHash()))
	if err != nil {
		t.Fatalf("block get: %v", err)
	}
	raw := new(bytes.Buffer)
	if _, err := raw.ReadFrom(blk); err != nil {
		t.Fatalf("read block: %v", err)
	}
	// The operation is JSON with the value as base64; it is encoded, not
	// encrypted.
	if !bytes.Contains(raw.Bytes(), []byte(base64.StdEncoding.EncodeToString(marker))) {
		t.Fatalf("payload not found (base64) in the entry block (%d bytes)", raw.Len())
	}
}

func TestParityIdentitySignVerify(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	odb, _ := newODB(ctx, t)
	id := odb.Identity()
	if id == nil || id.ID == "" || len(id.PublicKey) == 0 {
		t.Fatalf("identity missing fields: %+v", id)
	}
	data := []byte("signed by the orbitdb identity")
	sig, err := id.Provider.Sign(ctx, id, data)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	// Identity.GetPublicKey expects a protobuf-wrapped key and fails on the
	// default provider's raw secp256k1 key; the provider's own
	// UnmarshalPublicKey reads it.
	if _, err := id.GetPublicKey(); err == nil {
		t.Log("Identity.GetPublicKey now accepts the key; revisit the note above")
	}
	pub, err := id.Provider.UnmarshalPublicKey(id.PublicKey)
	if err != nil {
		t.Fatalf("public key: %v", err)
	}
	ok, err := pub.Verify(data, sig)
	if err != nil || !ok {
		t.Fatalf("verify = %v, %v", ok, err)
	}
	if ok, _ := pub.Verify([]byte("other data"), sig); ok {
		t.Fatal("signature verified for different data")
	}
}

func TestParityWriteAccessRefusesOtherIdentity(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	odb, dir := newODB(ctx, t)
	other := "021234567890abcdef021234567890abcdef021234567890abcdef0212345678ab"
	docs, err := odb.Docs(ctx, "parity-acl", &orbitdb.CreateDBOptions{
		Directory:        &dir,
		AccessController: &accesscontroller.CreateAccessControllerOptions{Access: map[string][]string{"write": {other}}},
	})
	if err != nil {
		t.Fatalf("open docs: %v", err)
	}
	if _, err := docs.Put(ctx, map[string]interface{}{"_id": "x"}); err == nil {
		t.Fatal("put by an identity outside the write list succeeded")
	}
}
