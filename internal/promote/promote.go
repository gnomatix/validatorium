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

	"gnomatix/validatorium/internal/identity"
	"gnomatix/validatorium/internal/model"
	"gnomatix/validatorium/internal/namespace"
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
	// Namespaced lists the namespaced pointers written, one per requested
	// namespace, each naming the promoted version.
	Namespaced []storage.NamespacedPointer
	// SignatureID is the content identifier of the signature record made
	// over the promoted version, when the Promoter has a Signer.
	SignatureID string
}

// SignatureStore records signature records; the bbolt working store
// implements it.
type SignatureStore interface {
	PutSignature(ctx context.Context, s *model.Signature) (string, error)
}

// ActionPromote is the action recorded on a promotion signature.
const ActionPromote = "promote"

// Promoter wires a working store to a global store.
type Promoter struct {
	Working storage.WorkingStore
	Global  storage.GlobalStore
	// Now supplies the promotion time; nil means time.Now.
	Now func() time.Time
	// Signer, when set, signs every promoted version with its OpenPGP key,
	// and the signature record is kept in the working store (which must
	// then implement SignatureStore). The signature attributes the action;
	// it adds no evidential weight.
	Signer *identity.Signer
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
	return p.PromoteInto(ctx, premiseUUID, att, nil)
}

// PromoteInto is Promote followed by placing the promoted version into each
// named namespace, in both tiers. Namespaces organise records; they add no
// validity, so the admission checks are exactly those of Promote. Every name
// is validated before anything is written.
func (p *Promoter) PromoteInto(ctx context.Context, premiseUUID string, att *model.ValidityAttestation, namespaces []namespace.Name) (Result, error) {
	if p.Working == nil || p.Global == nil {
		return Result{}, errors.New("promote: working and global stores are required")
	}
	if att == nil {
		return Result{}, errors.New("promote: attestation is required")
	}
	for _, ns := range namespaces {
		if _, err := namespace.Parse(string(ns)); err != nil {
			return Result{}, fmt.Errorf("promote %s: %w", premiseUUID, err)
		}
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

	// Sign before writing anything, so a signing failure leaves both tiers
	// unchanged.
	var sig *model.Signature
	var sigStore SignatureStore
	if p.Signer != nil {
		var ok bool
		if sigStore, ok = p.Working.(SignatureStore); !ok {
			return Result{}, fmt.Errorf("promote %s: a signer needs a working store that records signatures", premiseUUID)
		}
		armored, err := p.Signer.SignDetached(data)
		if err != nil {
			return Result{}, fmt.Errorf("promote %s: %w", premiseUUID, err)
		}
		sig = &model.Signature{SubjectCID: versionCID, Signer: p.Signer.Fingerprint(), Action: ActionPromote, ArmoredSignature: armored, SignedAt: now().UTC().Format(time.RFC3339)}
		if err := sig.ValidateRecord(); err != nil {
			return Result{}, fmt.Errorf("promote %s: %w", premiseUUID, err)
		}
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

	res := Result{Version: ref, AttestationCID: attCID, Pointer: ptr}
	if sig != nil {
		id, err := sigStore.PutSignature(ctx, sig)
		if err != nil {
			return res, fmt.Errorf("promote %s: record signature: %w", premiseUUID, err)
		}
		res.SignatureID = id
	}
	for _, ns := range namespaces {
		np := storage.NamespacedPointer{UUID: cand.UUID, Namespace: ns, CID: ref.CID, UpdatedAt: ptr.UpdatedAt}
		if err := p.Global.PutNamespacedPointer(ctx, np); err != nil {
			return res, fmt.Errorf("promote %s: publish into namespace %s: %w", premiseUUID, ns, err)
		}
		if err := p.Working.PutNamespacedPointer(ctx, np); err != nil {
			return res, fmt.Errorf("promote %s: record namespace %s: %w", premiseUUID, ns, err)
		}
		res.Namespaced = append(res.Namespaced, np)
	}
	return res, nil
}
