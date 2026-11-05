package orbit

import (
	"context"
	"errors"
	"testing"
	"time"

	"berty.tech/go-orbit-db/stores"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"

	"gnomatix/validatorium/internal/namespace"
	"gnomatix/validatorium/internal/storage"
)

const (
	nsUUID1 = "1a2b3c4d-0000-4000-8000-000000000001"
	nsUUID2 = "1a2b3c4d-0000-4000-8000-000000000002"
	nsUUID3 = "1a2b3c4d-0000-4000-8000-000000000003"
	nsCID1  = "bafkreigh2akiscaildcqabsyg3dfr6chu3fgpregiymsck7e7aqa4s52zy"
	nsCID2  = "bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku"
)

func nsPointer(ns namespace.Name, uuid, cid string) storage.NamespacedPointer {
	return storage.NamespacedPointer{
		UUID:      uuid,
		Namespace: ns,
		CID:       cid,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

func TestSingleNodeNamespacedPointers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	mn := mocknet.New()
	defer mn.Close()
	node := newMockNode(ctx, t, mn)
	gs := openStore(ctx, t, node, "ns-single-test", nil)

	nsA := namespace.MustParse("validatorium:alpha")
	nsB := namespace.MustParse("validatorium:beta")

	// Unknown namespace, unknown UUID.
	if _, err := gs.GetNamespacedPointer(ctx, nsA, nsUUID1); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("Get(unknown) err = %v, want storage.ErrNotFound", err)
	}
	if got, err := gs.ListNamespace(ctx, nsA); err != nil || len(got) != 0 {
		t.Fatalf("ListNamespace(empty) = %v, %v; want empty, nil", got, err)
	}

	// Put/Get round trip.
	p1 := nsPointer(nsA, nsUUID1, nsCID1)
	if err := gs.PutNamespacedPointer(ctx, p1); err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, err := gs.GetNamespacedPointer(ctx, nsA, nsUUID1)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != p1 {
		t.Fatalf("Get = %+v, want %+v", got, p1)
	}

	// Replacing keeps the key, changes the version.
	p1b := p1
	p1b.CID = nsCID2
	if err := gs.PutNamespacedPointer(ctx, p1b); err != nil {
		t.Fatalf("Put (replace): %v", err)
	}
	got, err = gs.GetNamespacedPointer(ctx, nsA, nsUUID1)
	if err != nil {
		t.Fatalf("Get after replace: %v", err)
	}
	if got.CID != nsCID2 {
		t.Fatalf("Get after replace CID = %s, want %s", got.CID, nsCID2)
	}

	// Two namespaces with overlapping UUIDs stay separate, and List is in
	// UUID order regardless of insertion order.
	for _, p := range []storage.NamespacedPointer{
		nsPointer(nsA, nsUUID3, nsCID1),
		nsPointer(nsA, nsUUID2, nsCID1),
		nsPointer(nsB, nsUUID2, nsCID2),
		nsPointer(nsB, nsUUID1, nsCID2),
	} {
		if err := gs.PutNamespacedPointer(ctx, p); err != nil {
			t.Fatalf("Put %s/%s: %v", p.Namespace, p.UUID, err)
		}
	}
	listA, err := gs.ListNamespace(ctx, nsA)
	if err != nil {
		t.Fatalf("ListNamespace(A): %v", err)
	}
	wantA := []string{nsUUID1, nsUUID2, nsUUID3}
	if len(listA) != len(wantA) {
		t.Fatalf("ListNamespace(A) len = %d, want %d: %+v", len(listA), len(wantA), listA)
	}
	for i, p := range listA {
		if p.UUID != wantA[i] || p.Namespace != nsA {
			t.Fatalf("ListNamespace(A)[%d] = %+v, want uuid %s in %s", i, p, wantA[i], nsA)
		}
	}
	listB, err := gs.ListNamespace(ctx, nsB)
	if err != nil {
		t.Fatalf("ListNamespace(B): %v", err)
	}
	wantB := []string{nsUUID1, nsUUID2}
	if len(listB) != len(wantB) {
		t.Fatalf("ListNamespace(B) len = %d, want %d: %+v", len(listB), len(wantB), listB)
	}
	for i, p := range listB {
		if p.UUID != wantB[i] || p.Namespace != nsB || p.CID != nsCID2 {
			t.Fatalf("ListNamespace(B)[%d] = %+v, want uuid %s in %s with cid %s", i, p, wantB[i], nsB, nsCID2)
		}
	}
	// The main pointer store is untouched by namespaced writes.
	if _, err := gs.GetPointer(ctx, nsUUID1); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("GetPointer(main store) err = %v, want storage.ErrNotFound", err)
	}

	// Delete then Get => ErrNotFound; the other namespace keeps its pointer.
	if err := gs.DeleteNamespacedPointer(ctx, nsA, nsUUID2); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := gs.GetNamespacedPointer(ctx, nsA, nsUUID2); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("Get after delete err = %v, want storage.ErrNotFound", err)
	}
	if _, err := gs.GetNamespacedPointer(ctx, nsB, nsUUID2); err != nil {
		t.Fatalf("Get in other namespace after delete: %v", err)
	}
	listA, err = gs.ListNamespace(ctx, nsA)
	if err != nil {
		t.Fatalf("ListNamespace(A) after delete: %v", err)
	}
	if len(listA) != 2 || listA[0].UUID != nsUUID1 || listA[1].UUID != nsUUID3 {
		t.Fatalf("ListNamespace(A) after delete = %+v", listA)
	}

	// Delete absent => nil, both for a deleted key and a never-written one.
	if err := gs.DeleteNamespacedPointer(ctx, nsA, nsUUID2); err != nil {
		t.Fatalf("Delete(absent, deleted) err = %v, want nil", err)
	}
	if err := gs.DeleteNamespacedPointer(ctx, namespace.MustParse("validatorium:never"), nsUUID1); err != nil {
		t.Fatalf("Delete(absent, unknown namespace) err = %v, want nil", err)
	}

	// Invalid namespace names are rejected everywhere, before any store opens.
	bad := namespace.Name("Not:Valid")
	if err := gs.PutNamespacedPointer(ctx, nsPointer(bad, nsUUID1, nsCID1)); !errors.Is(err, namespace.ErrInvalidName) {
		t.Fatalf("Put(invalid ns) err = %v, want namespace.ErrInvalidName", err)
	}
	if _, err := gs.GetNamespacedPointer(ctx, bad, nsUUID1); !errors.Is(err, namespace.ErrInvalidName) {
		t.Fatalf("Get(invalid ns) err = %v, want namespace.ErrInvalidName", err)
	}
	if _, err := gs.ListNamespace(ctx, bad); !errors.Is(err, namespace.ErrInvalidName) {
		t.Fatalf("List(invalid ns) err = %v, want namespace.ErrInvalidName", err)
	}
	if err := gs.DeleteNamespacedPointer(ctx, bad, nsUUID1); !errors.Is(err, namespace.ErrInvalidName) {
		t.Fatalf("Delete(invalid ns) err = %v, want namespace.ErrInvalidName", err)
	}
	if err := gs.PutNamespacedPointer(ctx, nsPointer("", nsUUID1, nsCID1)); !errors.Is(err, namespace.ErrInvalidName) {
		t.Fatalf("Put(empty ns) err = %v, want namespace.ErrInvalidName", err)
	}
	if err := gs.PutNamespacedPointer(ctx, nsPointer(nsA, "", nsCID1)); err == nil {
		t.Fatal("Put with empty UUID succeeded, want error")
	}

	// The namespace stores are distinct from each other and from the main
	// store, and named deterministically.
	addrA, err := gs.NamespaceAddress(ctx, nsA)
	if err != nil {
		t.Fatalf("NamespaceAddress(A): %v", err)
	}
	addrB, err := gs.NamespaceAddress(ctx, nsB)
	if err != nil {
		t.Fatalf("NamespaceAddress(B): %v", err)
	}
	if addrA == addrB || addrA == gs.Address() {
		t.Fatalf("namespace addresses not distinct: A=%s B=%s main=%s", addrA, addrB, gs.Address())
	}
	if NamespaceStoreName(nsA) != "validatorium-ns-validatorium:alpha" {
		t.Fatalf("NamespaceStoreName = %q", NamespaceStoreName(nsA))
	}

	// Close closes namespace stores too and is idempotent; use after close
	// is refused rather than reopening stores.
	if err := gs.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := gs.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	if _, err := gs.ListNamespace(ctx, nsA); err == nil {
		t.Fatal("ListNamespace after Close succeeded, want error")
	}
}

// TestTwoNodeNamespaceReplication proves that namespaced pointers written on
// node A reach node B through a namespace store that B opens by name alone:
// the same name and write list yield the same OrbitDB address on both sides.
// As with TestTwoNodeReplication, convergence is a transport fact and says
// nothing about whether the pointers are correct.
func TestTwoNodeNamespaceReplication(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping multi-node test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	mn := mocknet.New()
	defer mn.Close()

	nodeA := newMockNode(ctx, t, mn)
	nodeB := newMockNode(ctx, t, mn)

	if _, err := mn.LinkPeers(nodeA.PeerID(), nodeB.PeerID()); err != nil {
		t.Fatalf("link peers: %v", err)
	}
	if _, err := mn.ConnectPeers(nodeA.PeerID(), nodeB.PeerID()); err != nil {
		t.Fatalf("connect peers: %v", err)
	}

	ns := namespace.MustParse("validatorium:test")

	gsA := openStore(ctx, t, nodeA, "ns-replication-test", nil)
	want := []storage.NamespacedPointer{
		nsPointer(ns, nsUUID2, nsCID2),
		nsPointer(ns, nsUUID1, nsCID1),
	}
	for _, p := range want {
		if err := gsA.PutNamespacedPointer(ctx, p); err != nil {
			t.Fatalf("A Put %s: %v", p.UUID, err)
		}
	}
	addrA, err := gsA.NamespaceAddress(ctx, ns)
	if err != nil {
		t.Fatalf("A NamespaceAddress: %v", err)
	}

	// Node B opens the main store by address with A's write list; its
	// namespace stores inherit that list and so land on A's addresses.
	gsB := openStore(ctx, t, nodeB, gsA.Address(), &Options{
		WriteAccess:       []string{gsA.Identity()},
		BlockFetchTimeout: 20 * time.Second,
	})
	addrB, err := gsB.NamespaceAddress(ctx, ns)
	if err != nil {
		t.Fatalf("B NamespaceAddress: %v", err)
	}
	if addrB != addrA {
		t.Fatalf("B namespace address = %s, want A's %s (same name, same write list)", addrB, addrA)
	}

	docsB, err := gsB.NamespaceDocs(ctx, ns)
	if err != nil {
		t.Fatalf("B NamespaceDocs: %v", err)
	}
	sub, err := docsB.EventBus().Subscribe(new(stores.EventReplicated))
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Close()

	deadline := time.After(30 * time.Second)
	var got []storage.NamespacedPointer
wait:
	for {
		// The event may already have fired before the subscription; poll on
		// each tick as well as on each event.
		got, err = gsB.ListNamespace(ctx, ns)
		if err != nil {
			t.Fatalf("B ListNamespace: %v", err)
		}
		if len(got) == len(want) {
			break wait
		}
		select {
		case <-sub.Out():
		case <-time.After(500 * time.Millisecond):
		case <-deadline:
			t.Fatalf("timed out waiting for namespace replication; B has %d of %d pointers", len(got), len(want))
		case <-ctx.Done():
			t.Fatal("context cancelled waiting for replication")
		}
	}

	// List is in UUID order; want was inserted out of order.
	if got[0].UUID != nsUUID1 || got[0].CID != nsCID1 || got[1].UUID != nsUUID2 || got[1].CID != nsCID2 {
		t.Fatalf("B ListNamespace = %+v", got)
	}
	for _, p := range got {
		if p.Namespace != ns {
			t.Fatalf("B replicated pointer in wrong namespace: %+v", p)
		}
	}
	one, err := gsB.GetNamespacedPointer(ctx, ns, nsUUID1)
	if err != nil {
		t.Fatalf("B GetNamespacedPointer: %v", err)
	}
	if one.CID != nsCID1 {
		t.Fatalf("B GetNamespacedPointer CID = %s, want %s", one.CID, nsCID1)
	}

	// Another namespace nobody wrote to is empty on both sides.
	other := namespace.MustParse("validatorium:other")
	if got, err := gsB.ListNamespace(ctx, other); err != nil || len(got) != 0 {
		t.Fatalf("B ListNamespace(other) = %v, %v; want empty, nil", got, err)
	}
}
