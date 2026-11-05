package orbit

import (
	"context"
	"errors"
	"fmt"
	"sort"

	orbitdb "berty.tech/go-orbit-db"
	"berty.tech/go-orbit-db/accesscontroller"
	"berty.tech/go-orbit-db/iface"

	"gnomatix/validatorium/internal/namespace"
	"gnomatix/validatorium/internal/storage"
)

// NamespaceStorePrefix prefixes the OrbitDB store name of every namespace
// store. The full name is NamespaceStorePrefix followed by the namespace
// name verbatim, colons included, so each namespace maps to exactly one
// store name and the mapping is readable in the OrbitDB address path.
//
// OrbitDB derives a store address from its manifest: name, store type and
// the access-controller address, which in turn is derived only from the
// write list. Two nodes that open the same namespace name with the same
// write list therefore compute the same address and replicate the same
// log without exchanging addresses out of band. The two-node test in
// namespace_test.go asserts this.
//
// The store name also becomes a directory under the OrbitDB cache
// directory. Colons are valid there on POSIX filesystems; a port to a
// filesystem that rejects them would need a different NamespaceStoreName.
const NamespaceStorePrefix = "validatorium-ns-"

// NamespaceStoreName returns the OrbitDB store name used for ns.
func NamespaceStoreName(ns namespace.Name) string {
	return NamespaceStorePrefix + string(ns)
}

// namespaceStore returns the Documents store for ns, opening it on first
// use with the write list the pointer store was opened with and caching it
// for the life of the GlobalStore. ns must already be validated.
func (g *GlobalStore) namespaceStore(ctx context.Context, ns namespace.Name) (iface.DocumentStore, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return nil, errors.New("orbit: store is closed")
	}
	if ds, ok := g.nsStores[ns]; ok {
		return ds, nil
	}

	access := &accesscontroller.CreateAccessControllerOptions{
		Access: map[string][]string{"write": append([]string(nil), g.write...)},
	}
	ds, err := g.odb.Docs(ctx, NamespaceStoreName(ns), &orbitdb.CreateDBOptions{
		Directory:        &g.dir,
		AccessController: access,
	})
	if err != nil {
		return nil, fmt.Errorf("orbit: open namespace store %s: %w", ns, err)
	}
	g.nsStores[ns] = ds
	return ds, nil
}

// validNamespace parses ns so a malformed name is rejected before it can
// name a store.
func validNamespace(ns namespace.Name) (namespace.Name, error) {
	parsed, err := namespace.Parse(string(ns))
	if err != nil {
		return "", fmt.Errorf("orbit: %w", err)
	}
	return parsed, nil
}

// NamespaceDocs returns the Documents store backing ns, opening it if
// needed, for callers that need its event bus (for example to wait for
// replication in tests).
func (g *GlobalStore) NamespaceDocs(ctx context.Context, ns namespace.Name) (iface.DocumentStore, error) {
	ns, err := validNamespace(ns)
	if err != nil {
		return nil, err
	}
	return g.namespaceStore(ctx, ns)
}

// NamespaceAddress returns the OrbitDB address of the store backing ns,
// opening it if needed. Peers opened with the same write list arrive at the
// same address by name alone; the address is exposed so that can be checked
// and so an operator can inspect which log a namespace replicates through.
func (g *GlobalStore) NamespaceAddress(ctx context.Context, ns namespace.Name) (string, error) {
	ds, err := g.NamespaceDocs(ctx, ns)
	if err != nil {
		return "", err
	}
	return ds.Address().String(), nil
}

// PutNamespacedPointer stores or replaces the pointer for (p.Namespace,
// p.UUID) in that namespace's own store. The main pointer store is never
// written here. Placing a record in a namespace is organisation, not
// validation: the record's validity was settled before promotion.
func (g *GlobalStore) PutNamespacedPointer(ctx context.Context, p storage.NamespacedPointer) error {
	ns, err := validNamespace(p.Namespace)
	if err != nil {
		return err
	}
	if p.UUID == "" {
		return errors.New("orbit: namespaced pointer has empty uuid")
	}
	ds, err := g.namespaceStore(ctx, ns)
	if err != nil {
		return err
	}
	doc, err := namespacedPointerToDoc(p)
	if err != nil {
		return err
	}
	if _, err := ds.Put(ctx, doc); err != nil {
		return fmt.Errorf("orbit: put namespaced pointer %s/%s: %w", ns, p.UUID, err)
	}
	return nil
}

// GetNamespacedPointer returns the pointer for (ns, uuid) or
// storage.ErrNotFound.
func (g *GlobalStore) GetNamespacedPointer(ctx context.Context, ns namespace.Name, uuid string) (storage.NamespacedPointer, error) {
	ns, err := validNamespace(ns)
	if err != nil {
		return storage.NamespacedPointer{}, err
	}
	ds, err := g.namespaceStore(ctx, ns)
	if err != nil {
		return storage.NamespacedPointer{}, err
	}
	res, err := ds.Get(ctx, uuid, nil)
	if err != nil {
		return storage.NamespacedPointer{}, fmt.Errorf("orbit: get namespaced pointer %s/%s: %w", ns, uuid, err)
	}
	if len(res) == 0 {
		return storage.NamespacedPointer{}, storage.ErrNotFound
	}
	p, err := docToNamespacedPointer(res[0])
	if err != nil {
		return storage.NamespacedPointer{}, fmt.Errorf("orbit: decode namespaced pointer %s/%s: %w", ns, uuid, err)
	}
	return p, nil
}

// ListNamespace returns every pointer in ns in UUID order. A namespace
// nobody has written to yields an empty list: opening it merely creates an
// empty store.
func (g *GlobalStore) ListNamespace(ctx context.Context, ns namespace.Name) ([]storage.NamespacedPointer, error) {
	ns, err := validNamespace(ns)
	if err != nil {
		return nil, err
	}
	ds, err := g.namespaceStore(ctx, ns)
	if err != nil {
		return nil, err
	}
	docs, err := ds.Query(ctx, func(interface{}) (bool, error) { return true, nil })
	if err != nil {
		return nil, fmt.Errorf("orbit: list namespace %s: %w", ns, err)
	}
	out := make([]storage.NamespacedPointer, 0, len(docs))
	for _, doc := range docs {
		p, err := docToNamespacedPointer(doc)
		if err != nil {
			return nil, fmt.Errorf("orbit: decode namespaced pointer in %s: %w", ns, err)
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UUID < out[j].UUID })
	return out, nil
}

// DeleteNamespacedPointer removes the pointer for (ns, uuid). Deleting an
// absent pointer is not an error. The record and its versions are untouched;
// only this namespace stops referencing it.
//
// go-orbit-db's Documents store returns a plain error for a DEL of a key
// that is not in its index, so absence is checked through the index first
// rather than by matching that error's text. If the key vanishes between
// the check and the delete (a concurrent replicated DEL), the delete is
// retried as a lookup and reported as success when the key is gone.
func (g *GlobalStore) DeleteNamespacedPointer(ctx context.Context, ns namespace.Name, uuid string) error {
	ns, err := validNamespace(ns)
	if err != nil {
		return err
	}
	ds, err := g.namespaceStore(ctx, ns)
	if err != nil {
		return err
	}
	present, err := namespaceHasKey(ctx, ds, uuid)
	if err != nil {
		return fmt.Errorf("orbit: delete namespaced pointer %s/%s: %w", ns, uuid, err)
	}
	if !present {
		return nil
	}
	if _, err := ds.Delete(ctx, uuid); err != nil {
		stillPresent, lookupErr := namespaceHasKey(ctx, ds, uuid)
		if lookupErr == nil && !stillPresent {
			return nil
		}
		return fmt.Errorf("orbit: delete namespaced pointer %s/%s: %w", ns, uuid, err)
	}
	return nil
}

// namespaceHasKey reports whether the store's index currently holds key.
func namespaceHasKey(ctx context.Context, ds iface.DocumentStore, key string) (bool, error) {
	res, err := ds.Get(ctx, key, nil)
	if err != nil {
		return false, err
	}
	return len(res) > 0, nil
}

// namespacedPointerToDoc converts a NamespacedPointer into the map form the
// Documents store expects, via its JSON encoding so the "_id" key and field
// names match the struct tags exactly.
func namespacedPointerToDoc(p storage.NamespacedPointer) (map[string]interface{}, error) {
	return structToDoc(p, "namespaced pointer")
}

// docToNamespacedPointer converts a Documents-store value back into a
// NamespacedPointer.
func docToNamespacedPointer(doc interface{}) (storage.NamespacedPointer, error) {
	var p storage.NamespacedPointer
	if err := docToStruct(doc, &p, "namespaced pointer"); err != nil {
		return storage.NamespacedPointer{}, err
	}
	return p, nil
}
