// Package storage defines the tier boundaries of the Validatorium storage
// architecture (docs/specs/storage-architecture.md):
//
//   - Tier 1, the local working store (WorkingStore), holds mutable candidate
//     records, immutable local versions, UUID-to-version pointers, and the
//     attestation log. It works with zero connectivity.
//   - Tier 2, the global store (GlobalStore), holds content-addressed immutable
//     versions and replicated UUID pointer records. go-orbit-db over an
//     embedded Kubo node is the intended implementation; the interface lets a
//     different replication backend be substituted without redesign.
//
// Storage never determines truth. A record enters the global store only
// through promotion, which requires a true validity attestation for the exact
// version being promoted (see package promote).
package storage

import (
	"context"
	"errors"

	"gnomatix/validatorium/internal/model"
)

var (
	// ErrNotFound is returned when a key, record, version, or pointer does not
	// exist in the store that was asked.
	ErrNotFound = errors.New("not found")
	// ErrImmutable is returned when a write would change bytes already stored
	// under a content identifier or an attestation UUID.
	ErrImmutable = errors.New("record is immutable")
)

// Store is the minimal key/value abstraction used by early scaffolding. New
// code should use WorkingStore and GlobalStore.
type Store interface {
	Put(ctx context.Context, key string, val []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
	Close() error
}

// VersionRef names one immutable version of a UUID-identified record.
type VersionRef struct {
	UUID string `json:"uuid"`
	CID  string `json:"cid"`
}

// Pointer is the replicated, UUID-keyed record that names the current version
// of a premise in the global store. The version bytes themselves are immutable
// blocks addressed by CID; the pointer is the only mutable record per UUID.
//
// JSON tags match the go-orbit-db Documents-store convention: "_id" is the key.
type Pointer struct {
	// UUID is the premise identity and the Documents-store key.
	UUID string `json:"_id"`
	// CID is the content identifier of the current premise version.
	CID string `json:"cid"`
	// AttestationCID is the content identifier of the true validity
	// attestation under which this version was admitted.
	AttestationCID string `json:"attestationCID"`
	// Status is current or obsolete. An obsolete pointer stays addressable and
	// names its replacements.
	Status model.LifecycleStatus `json:"status"`
	// SupersededBy lists replacement premise UUIDs when Status is obsolete.
	SupersededBy []string `json:"supersededBy,omitempty"`
	// UpdatedAt is the RFC 3339 time this pointer record was written.
	UpdatedAt string `json:"updatedAt"`
}

// WorkingStore is the tier-1 local working store.
type WorkingStore interface {
	// PutCandidate stores or replaces a mutable local draft keyed by its UUID.
	PutCandidate(ctx context.Context, m *model.Micropublication) error
	// GetCandidate returns the draft for a UUID or ErrNotFound.
	GetCandidate(ctx context.Context, uuid string) (*model.Micropublication, error)
	// ListCandidates returns the UUIDs of all drafts in key order.
	ListCandidates(ctx context.Context) ([]string, error)
	// DeleteCandidate removes a draft. Deleting an absent draft is not an error.
	DeleteCandidate(ctx context.Context, uuid string) error

	// PutVersion stores an immutable version of the record and returns its
	// reference. Storing identical bytes again is a no-op.
	PutVersion(ctx context.Context, m *model.Micropublication) (VersionRef, error)
	// GetVersion returns the version stored under a CID or ErrNotFound.
	GetVersion(ctx context.Context, cid string) (*model.Micropublication, error)

	// SetCurrent points a UUID at the CID of its current version.
	SetCurrent(ctx context.Context, ref VersionRef) error
	// GetCurrent returns the current version reference for a UUID or ErrNotFound.
	GetCurrent(ctx context.Context, uuid string) (VersionRef, error)

	// PutAttestation appends an immutable attestation. Writing a different
	// record under an existing attestation UUID returns ErrImmutable.
	PutAttestation(ctx context.Context, a *model.ValidityAttestation) error
	// GetAttestation returns one attestation by UUID or ErrNotFound.
	GetAttestation(ctx context.Context, uuid string) (*model.ValidityAttestation, error)
	// ListAttestations returns every attestation recorded for a premise UUID
	// in attestation-UUID order. An empty result is not an error.
	ListAttestations(ctx context.Context, premiseUUID string) ([]*model.ValidityAttestation, error)

	// PutTermRecord stores an immutable version of a term record and points
	// its UUID at that version. Earlier versions remain addressable.
	PutTermRecord(ctx context.Context, t *model.TermRecord) (VersionRef, error)
	// GetTermRecord returns the current version of a term record or ErrNotFound.
	GetTermRecord(ctx context.Context, uuid string) (*model.TermRecord, error)
	// GetTermRecordVersion returns one stored term record version by CID.
	GetTermRecordVersion(ctx context.Context, cid string) (*model.TermRecord, error)
	// ResolveTerm follows replacement pointers from a term record UUID to the
	// current records that replace it (see model.ResolveTermReplacements).
	ResolveTerm(ctx context.Context, uuid string) ([]*model.TermRecord, error)

	Close() error
}

// GlobalStore is the tier-2 boundary: content-addressed immutable blocks plus
// replicated UUID pointer records.
type GlobalStore interface {
	// PutBlock stores immutable bytes and returns their CID. Implementations
	// must return a CID equal to ContentID(data) so that tier-1 version
	// references and tier-2 block addresses agree.
	PutBlock(ctx context.Context, data []byte) (string, error)
	// GetBlock returns the bytes for a CID or ErrNotFound. Implementations must
	// verify that the bytes hash to the requested CID.
	GetBlock(ctx context.Context, cid string) ([]byte, error)

	// PutPointer stores or replaces the pointer record for a UUID.
	PutPointer(ctx context.Context, p Pointer) error
	// GetPointer returns the pointer record for a UUID or ErrNotFound.
	GetPointer(ctx context.Context, uuid string) (Pointer, error)

	Close() error
}
