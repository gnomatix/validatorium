package promote

import (
	"context"
	"errors"
	"testing"

	"gnomatix/validatorium/internal/model"
	"gnomatix/validatorium/internal/namespace"
	"gnomatix/validatorium/internal/storage"
)

func TestPromoteIntoPlacesVersionInNamespacesOnBothTiers(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	chem := namespace.MustParse("validatorium:chemistry")
	phys := namespace.MustParse("validatorium:physics")

	res, err := f.p.PromoteInto(ctx, f.cand.UUID, f.attestation(t, model.ResultTrue), []namespace.Name{chem, phys})
	if err != nil {
		t.Fatalf("PromoteInto: %v", err)
	}
	if len(res.Namespaced) != 2 {
		t.Fatalf("expected 2 namespaced pointers, got %+v", res.Namespaced)
	}
	for _, ns := range []namespace.Name{chem, phys} {
		for name, store := range map[string]storage.NamespaceStore{"global": f.gs, "working": f.ws} {
			np, err := store.GetNamespacedPointer(ctx, ns, f.cand.UUID)
			if err != nil {
				t.Fatalf("%s: GetNamespacedPointer(%s): %v", name, ns, err)
			}
			if np.CID != f.cid || np.Namespace != ns {
				t.Fatalf("%s: namespaced pointer = %+v", name, np)
			}
		}
	}
	list, _ := f.gs.ListNamespace(ctx, namespace.Root)
	if len(list) != 0 {
		t.Fatalf("root namespace must not be populated implicitly: %+v", list)
	}
}

func TestPromoteIntoRejectsInvalidNamespaceBeforeWriting(t *testing.T) {
	f := setup(t)
	_, err := f.p.PromoteInto(context.Background(), f.cand.UUID, f.attestation(t, model.ResultTrue), []namespace.Name{"validatorium:chemistry", "Not Valid"})
	if !errors.Is(err, namespace.ErrInvalidName) {
		t.Fatalf("expected ErrInvalidName, got %v", err)
	}
	f.assertUnchanged(t)
}

func TestPromoteIntoRefusalWritesNoNamespacePointers(t *testing.T) {
	f := setup(t)
	chem := namespace.MustParse("validatorium:chemistry")
	_, err := f.p.PromoteInto(context.Background(), f.cand.UUID, f.attestation(t, model.ResultFalse), []namespace.Name{chem})
	if !errors.Is(err, ErrNotValid) {
		t.Fatalf("expected ErrNotValid, got %v", err)
	}
	f.assertUnchanged(t)
	if list, _ := f.gs.ListNamespace(context.Background(), chem); len(list) != 0 {
		t.Fatalf("namespace populated despite refusal: %+v", list)
	}
}
