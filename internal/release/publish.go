package release

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gnomatix/validatorium/internal/model"
	"gnomatix/validatorium/internal/namespace"
	"gnomatix/validatorium/internal/storage"
)

// Namespace is the namespace in which published releases are listed, keyed by
// release label.
var Namespace = namespace.MustParse("validatorium:releases")

// ErrLabelTaken is returned when a release label is already published for a
// different manifest. A historical release never changes.
var ErrLabelTaken = errors.New("release label already published for a different manifest")

// ErrMissingVersion is returned when a premise version the manifest tags is
// not in the global store. Premise versions reach the global store only
// through promotion, so an absent block means the release is not publishable.
var ErrMissingVersion = errors.New("tagged premise version is not in the global store")

// TermSource is the read access Publish needs to copy exact term record
// versions into the global store with the manifest.
type TermSource interface {
	GetTermRecordVersion(ctx context.Context, cid string) (*model.TermRecord, error)
}

// Publish stores the manifest as an immutable block and lists it under its
// label in the releases namespace. It first checks that every tagged premise
// version is already in the global store, and copies the exact term record
// versions the release relies on, so the release is retrievable as a unit.
// Publishing the same manifest under the same label again is a no-op.
func Publish(ctx context.Context, gs storage.GlobalStore, terms TermSource, res Result, now time.Time) (storage.NamespacedPointer, error) {
	label := res.Manifest.Version
	if label == "" {
		return storage.NamespacedPointer{}, errors.New("release: manifest has no label")
	}
	data, err := res.Manifest.CanonicalBytes()
	if err != nil {
		return storage.NamespacedPointer{}, err
	}
	if err := storage.VerifyContentID(res.CID, data); err != nil {
		return storage.NamespacedPointer{}, fmt.Errorf("release %s: manifest does not match its CID: %w", label, err)
	}

	existing, err := gs.GetNamespacedPointer(ctx, Namespace, label)
	switch {
	case err == nil && existing.CID == res.CID:
		return existing, nil
	case err == nil:
		return storage.NamespacedPointer{}, fmt.Errorf("release %s: %w (published %s, this one %s)", label, ErrLabelTaken, existing.CID, res.CID)
	case !errors.Is(err, storage.ErrNotFound):
		return storage.NamespacedPointer{}, fmt.Errorf("release %s: read releases namespace: %w", label, err)
	}

	for _, p := range res.Manifest.Premises {
		if _, err := gs.GetBlock(ctx, p.CID); err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return storage.NamespacedPointer{}, fmt.Errorf("release %s: %w: %s@%s", label, ErrMissingVersion, p.UUID, p.CID)
			}
			return storage.NamespacedPointer{}, fmt.Errorf("release %s: check %s: %w", label, p.CID, err)
		}
	}

	for _, t := range res.Manifest.Terms {
		rec, err := terms.GetTermRecordVersion(ctx, t.CID)
		if err != nil {
			return storage.NamespacedPointer{}, fmt.Errorf("release %s: term record %s@%s: %w", label, t.UUID, t.CID, err)
		}
		tb, err := rec.CanonicalBytes()
		if err != nil {
			return storage.NamespacedPointer{}, err
		}
		got, err := gs.PutBlock(ctx, tb)
		if err != nil {
			return storage.NamespacedPointer{}, fmt.Errorf("release %s: publish term %s: %w", label, t.UUID, err)
		}
		if got != t.CID {
			return storage.NamespacedPointer{}, fmt.Errorf("release %s: term %s stored as %s, manifest says %s", label, t.UUID, got, t.CID)
		}
	}

	got, err := gs.PutBlock(ctx, data)
	if err != nil {
		return storage.NamespacedPointer{}, fmt.Errorf("release %s: publish manifest: %w", label, err)
	}
	if got != res.CID {
		return storage.NamespacedPointer{}, fmt.Errorf("release %s: manifest stored as %s, expected %s", label, got, res.CID)
	}
	ptr := storage.NamespacedPointer{UUID: label, Namespace: Namespace, CID: res.CID, UpdatedAt: now.UTC().Format(time.RFC3339Nano)}
	if err := gs.PutNamespacedPointer(ctx, ptr); err != nil {
		return storage.NamespacedPointer{}, fmt.Errorf("release %s: list in %s: %w", label, Namespace, err)
	}
	return ptr, nil
}
