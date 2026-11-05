package bbolt_test

import (
	"context"
	"errors"
	"testing"

	"gnomatix/validatorium/internal/namespace"
	"gnomatix/validatorium/internal/storage"
)

func TestNamespacedPointersAreAPointerLayer(t *testing.T) {
	ws := openWorking(t)
	ctx := context.Background()
	chem := namespace.MustParse("validatorium:chemistry")
	phys := namespace.MustParse("validatorium:physics")

	if _, err := ws.GetNamespacedPointer(ctx, chem, "u1"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	// Invalid names and incomplete pointers are refused.
	if err := ws.PutNamespacedPointer(ctx, storage.NamespacedPointer{UUID: "u1", Namespace: "Bad Name", CID: "c1"}); err == nil {
		t.Fatal("invalid namespace accepted")
	}
	if err := ws.PutNamespacedPointer(ctx, storage.NamespacedPointer{UUID: "u1", Namespace: chem}); err == nil {
		t.Fatal("pointer without CID accepted")
	}

	// One record referenced from two namespaces, possibly at different versions.
	puts := []storage.NamespacedPointer{
		{UUID: "u2", Namespace: chem, CID: "c2", UpdatedAt: "t"},
		{UUID: "u1", Namespace: chem, CID: "c1", UpdatedAt: "t"},
		{UUID: "u1", Namespace: phys, CID: "c1-older", UpdatedAt: "t"},
		{UUID: "u3", Namespace: namespace.Root, CID: "c3", UpdatedAt: "t"},
	}
	for _, p := range puts {
		if err := ws.PutNamespacedPointer(ctx, p); err != nil {
			t.Fatalf("PutNamespacedPointer %+v: %v", p, err)
		}
	}
	got, err := ws.GetNamespacedPointer(ctx, phys, "u1")
	if err != nil || got.CID != "c1-older" || got.Namespace != phys {
		t.Fatalf("GetNamespacedPointer(phys,u1) = %+v, %v", got, err)
	}

	// Listing is per namespace, in UUID order, and a namespace that is a
	// textual prefix of another never sees the other's pointers.
	list, err := ws.ListNamespace(ctx, chem)
	if err != nil {
		t.Fatalf("ListNamespace: %v", err)
	}
	if len(list) != 2 || list[0].UUID != "u1" || list[1].UUID != "u2" {
		t.Fatalf("ListNamespace(chem) = %+v", list)
	}
	root, _ := ws.ListNamespace(ctx, namespace.Root)
	if len(root) != 1 || root[0].UUID != "u3" {
		t.Fatalf("ListNamespace(root) must not include children: %+v", root)
	}
	none, err := ws.ListNamespace(ctx, namespace.MustParse("validatorium:chem"))
	if err != nil || len(none) != 0 {
		t.Fatalf("prefix namespace matched foreign pointers: %+v %v", none, err)
	}

	// Replacing a pointer moves the namespace to a new version.
	if err := ws.PutNamespacedPointer(ctx, storage.NamespacedPointer{UUID: "u1", Namespace: chem, CID: "c1-v2", UpdatedAt: "t2"}); err != nil {
		t.Fatal(err)
	}
	got, _ = ws.GetNamespacedPointer(ctx, chem, "u1")
	if got.CID != "c1-v2" {
		t.Fatalf("pointer not replaced: %+v", got)
	}

	// Deleting from one namespace leaves the other namespace's pointer alone.
	if err := ws.DeleteNamespacedPointer(ctx, chem, "u1"); err != nil {
		t.Fatal(err)
	}
	if err := ws.DeleteNamespacedPointer(ctx, chem, "u1"); err != nil {
		t.Fatalf("deleting an absent pointer must not error: %v", err)
	}
	if _, err := ws.GetNamespacedPointer(ctx, chem, "u1"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("pointer still present after delete: %v", err)
	}
	if _, err := ws.GetNamespacedPointer(ctx, phys, "u1"); err != nil {
		t.Fatalf("delete in one namespace affected another: %v", err)
	}
}
