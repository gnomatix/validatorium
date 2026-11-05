// Package promote implements the seam between the tier-1 working store and
// the tier-2 global store (docs/specs/storage-architecture.md §4).
//
// A local candidate is admitted to the global store only when a true validity
// attestation exists for the exact version being promoted. This package does
// not evaluate premises: the attestation is produced elsewhere by executing
// the premise's method against its evidence. Promote enforces the admission
// rule and records the exact versions involved; it cannot make a premise true.
package promote

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gnomatix/validatorium/internal/model"
	"gnomatix/validatorium/internal/storage"
)

var (
	// ErrNotValid is returned when the attestation does not establish validity
	// for the candidate: its result is not true, or it has expired.
	ErrNotValid = errors.New("promotion refused: attestation does not establish validity")
	// ErrVersionMismatch is returned when the attestation names a different
	// premise or premise version than the candidate being promoted.
	ErrVersionMismatch = errors.New("promotion refused: attestation does not describe this premise version")
	// ErrNotCurrent is returned when the candidate is not in the current
	// lifecycle state. Obsolescence is recorded through supersession, not
	// through promotion.
	ErrNotCurrent = errors.New("promotion refused: candidate is not current")
)

// Result records what one promotion wrote.
type Result struct {
	// Version is the promoted premise version.
	Version storage.VersionRef
	// AttestationCID is the content identifier of the admitting attestation.
	AttestationCID string
	// Pointer is the global pointer record written for the premise UUID.
	Pointer storage.Pointer
}

// Promoter wires a working store to a global store.
type Promoter struct {
	Working storage.WorkingStore
	Global  storage.GlobalStore
	// Now supplies the promotion time; nil means time.Now.
	Now func() time.Time
}

// Promote admits the candidate with the given UUID to the global store under
// the given attestation. It:
//
//  1. checks the candidate and attestation are well-formed records;
//  2. requires the candidate to be current;
//  3. requires the attestation to name this premise and the CID of the
//     candidate's exact canonical bytes;
//  4. requires the attestation's result to be true and unexpired at
//     promotion time;
//  5. records the version and attestation in the working store, publishes
//     both as immutable blocks in the global store, writes the global pointer,
//     sets the local current pointer, and removes the draft.
//
// Any refusal leaves both stores unchanged.
func (p *Promoter) Promote(ctx context.Context, premiseUUID string, att *model.ValidityAttestation) (Result, error) {
	if p.Working == nil || p.Global == nil {
		return Result{}, errors.New("promote: working and global stores are required")
	}
	if att == nil {
		return Result{}, errors.New("promote: attestation is required")
	}
	now := time.Now
	if p.Now != nil {
		now = p.Now
	}

	cand, err := p.Working.GetCandidate(ctx, premiseUUID)
	if err != nil {
		return Result{}, fmt.Errorf("promote %s: load candidate: %w", premiseUUID, err)
	}
	if err := cand.ValidateRecord(); err != nil {
		return Result{}, fmt.Errorf("promote %s: %w", premiseUUID, err)
	}
	if err := att.ValidateRecord(); err != nil {
		return Result{}, fmt.Errorf("promote %s: %w", premiseUUID, err)
	}
	if cand.State.Status != model.Current {
		return Result{}, fmt.Errorf("promote %s: %w (status %q)", premiseUUID, ErrNotCurrent, cand.State.Status)
	}

	data, err := cand.CanonicalBytes()
	if err != nil {
		return Result{}, err
	}
	versionCID, err := storage.ContentID(data)
	if err != nil {
		return Result{}, err
	}
	if att.PremiseUUID != cand.UUID || att.PremiseVersionCID != versionCID {
		return Result{}, fmt.Errorf("promote %s: %w (attestation names %s@%s, candidate is %s@%s)",
			premiseUUID, ErrVersionMismatch, att.PremiseUUID, att.PremiseVersionCID, cand.UUID, versionCID)
	}
	if !att.EstablishesValidityAt(now()) {
		return Result{}, fmt.Errorf("promote %s: %w (result %q, expiresAt %q)", premiseUUID, ErrNotValid, att.Result, att.ExpiresAt)
	}

	// All checks passed. Record locally first so the working store never
	// points at a version it does not hold.
	ref, err := p.Working.PutVersion(ctx, cand)
	if err != nil {
		return Result{}, fmt.Errorf("promote %s: store version: %w", premiseUUID, err)
	}
	if err := p.Working.PutAttestation(ctx, att); err != nil {
		return Result{}, fmt.Errorf("promote %s: store attestation: %w", premiseUUID, err)
	}

	attData, err := att.CanonicalBytes()
	if err != nil {
		return Result{}, err
	}
	gotVersionCID, err := p.Global.PutBlock(ctx, data)
	if err != nil {
		return Result{}, fmt.Errorf("promote %s: publish version: %w", premiseUUID, err)
	}
	if gotVersionCID != ref.CID {
		return Result{}, fmt.Errorf("promote %s: global store returned CID %s for version %s", premiseUUID, gotVersionCID, ref.CID)
	}
	attCID, err := p.Global.PutBlock(ctx, attData)
	if err != nil {
		return Result{}, fmt.Errorf("promote %s: publish attestation: %w", premiseUUID, err)
	}

	ptr := storage.Pointer{
		UUID:           cand.UUID,
		CID:            ref.CID,
		AttestationCID: attCID,
		Status:         cand.State.Status,
		SupersededBy:   cand.State.SupersededBy,
		UpdatedAt:      now().UTC().Format(time.RFC3339Nano),
	}
	if err := p.Global.PutPointer(ctx, ptr); err != nil {
		return Result{}, fmt.Errorf("promote %s: publish pointer: %w", premiseUUID, err)
	}
	if err := p.Working.SetCurrent(ctx, ref); err != nil {
		return Result{}, fmt.Errorf("promote %s: set current: %w", premiseUUID, err)
	}
	if err := p.Working.DeleteCandidate(ctx, cand.UUID); err != nil {
		return Result{}, fmt.Errorf("promote %s: remove draft: %w", premiseUUID, err)
	}
	return Result{Version: ref, AttestationCID: attCID, Pointer: ptr}, nil
}
