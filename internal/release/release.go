// Package release builds Validatorium release manifests (docs/specs/data-model.md
// §9, docs/design/two-tier-storage.md §7.2).
//
// A release references exact versions. A premise receives the release tag only
// when, at the release time, it is current, a true attestation exists for its
// exact current version and has not expired, every term record that attestation
// relied on is still current, and every premise the attestation depends on is
// also in the release. Anything else is excluded with its reason; exclusion is
// not a judgment on the premise, only on its eligibility for this release.
//
// A manifest, once built, is immutable: its canonical bytes are content
// addressed, and a later release is a new manifest.
package release

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"gnomatix/validatorium/internal/model"
	"gnomatix/validatorium/internal/storage"
)

// Manifest is one immutable release.
type Manifest struct {
	// Version is the release's own label, chosen by the caller.
	Version string `json:"version"`
	// CreatedAt is the RFC 3339 release time; eligibility is evaluated at it.
	CreatedAt string `json:"createdAt"`
	// Premises lists the exact premise versions tagged in this release, in
	// UUID order.
	Premises []storage.VersionRef `json:"premises"`
	// Attestations lists, per tagged premise, the UUID of the attestation that
	// made it eligible, in the same order as Premises.
	Attestations []string `json:"attestations"`
	// Terms lists the exact term record versions the tagged attestations
	// relied on, in UUID order, without duplicates.
	Terms []storage.VersionRef `json:"terms,omitempty"`
}

// CanonicalBytes returns the manifest's encoding for content addressing.
func (m *Manifest) CanonicalBytes() ([]byte, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("canonical manifest: %w", err)
	}
	return b, nil
}

// Exclusion records why a candidate premise was not tagged.
type Exclusion struct {
	UUID   string `json:"uuid"`
	Reason string `json:"reason"`
}

// Result is what Build returns: the manifest, its content identifier, and the
// candidates left out.
type Result struct {
	Manifest   Manifest
	CID        string
	Exclusions []Exclusion
}

// Store is the subset of storage.WorkingStore the builder reads.
type Store interface {
	GetCurrent(ctx context.Context, uuid string) (storage.VersionRef, error)
	GetVersion(ctx context.Context, cid string) (*model.Micropublication, error)
	ListAttestations(ctx context.Context, premiseUUID string) ([]*model.ValidityAttestation, error)
	GetTermRecord(ctx context.Context, uuid string) (*model.TermRecord, error)
}

// candidate is a premise that passed the per-premise checks and still has to
// pass the dependency check.
type candidate struct {
	ref  storage.VersionRef
	att  *model.ValidityAttestation
	deps []string
}

// Build evaluates the candidate premise UUIDs at time at and returns the
// release manifest. version labels the release. Errors other than "not found"
// from the store abort the build; a premise with no current version is
// excluded, not an error.
func Build(ctx context.Context, st Store, version string, at time.Time, candidates []string) (Result, error) {
	if strings.TrimSpace(version) == "" {
		return Result{}, errors.New("release: version label required")
	}
	var res Result
	exclude := func(uuid, reason string) {
		res.Exclusions = append(res.Exclusions, Exclusion{UUID: uuid, Reason: reason})
	}

	eligible := map[string]*candidate{}
	terms := map[string]storage.VersionRef{}
	seen := map[string]bool{}
	for _, uuid := range candidates {
		if seen[uuid] {
			continue
		}
		seen[uuid] = true

		ref, err := st.GetCurrent(ctx, uuid)
		if errors.Is(err, storage.ErrNotFound) {
			exclude(uuid, "no current version")
			continue
		} else if err != nil {
			return Result{}, fmt.Errorf("release: current pointer for %s: %w", uuid, err)
		}
		prem, err := st.GetVersion(ctx, ref.CID)
		if err != nil {
			return Result{}, fmt.Errorf("release: version %s of %s: %w", ref.CID, uuid, err)
		}
		if prem.State.Status != model.Current {
			exclude(uuid, "premise is obsolete")
			continue
		}

		att, err := pickAttestation(ctx, st, uuid, ref.CID, at)
		if err != nil {
			return Result{}, err
		}
		if att == nil {
			exclude(uuid, "no true, unexpired attestation for the current version")
			continue
		}

		termRefs, reason, err := checkTerms(ctx, st, att.TermRecordUUIDs)
		if err != nil {
			return Result{}, err
		}
		if reason != "" {
			exclude(uuid, reason)
			continue
		}
		for _, tr := range termRefs {
			terms[tr.UUID] = tr
		}
		eligible[uuid] = &candidate{ref: ref, att: att, deps: premiseDependencies(att.Dependencies)}
	}

	// A premise stays only while every premise it depends on stays. Removing
	// one can strand another, so repeat until nothing changes.
	for changed := true; changed; {
		changed = false
		for _, uuid := range sortedKeys(eligible) {
			for _, dep := range eligible[uuid].deps {
				if _, ok := eligible[dep]; !ok {
					exclude(uuid, "depends on "+dep+", which is not in this release")
					delete(eligible, uuid)
					changed = true
					break
				}
			}
		}
	}

	m := Manifest{Version: version, CreatedAt: at.UTC().Format(time.RFC3339Nano)}
	usedTerms := map[string]bool{}
	for _, uuid := range sortedKeys(eligible) {
		c := eligible[uuid]
		m.Premises = append(m.Premises, c.ref)
		m.Attestations = append(m.Attestations, c.att.UUID)
		for _, t := range c.att.TermRecordUUIDs {
			usedTerms[t] = true
		}
	}
	for _, t := range sortedKeys(usedTerms) {
		m.Terms = append(m.Terms, terms[t])
	}
	sort.Slice(res.Exclusions, func(i, j int) bool { return res.Exclusions[i].UUID < res.Exclusions[j].UUID })

	data, err := m.CanonicalBytes()
	if err != nil {
		return Result{}, err
	}
	cid, err := storage.ContentID(data)
	if err != nil {
		return Result{}, err
	}
	res.Manifest, res.CID = m, cid
	return res, nil
}

// pickAttestation returns the latest-evaluated attestation that names this
// exact version and establishes validity at the release time, or nil.
func pickAttestation(ctx context.Context, st Store, uuid, cid string, at time.Time) (*model.ValidityAttestation, error) {
	atts, err := st.ListAttestations(ctx, uuid)
	if err != nil {
		return nil, fmt.Errorf("release: attestations for %s: %w", uuid, err)
	}
	var best *model.ValidityAttestation
	var bestAt time.Time
	for _, a := range atts {
		if a.PremiseVersionCID != cid || !a.EstablishesValidityAt(at) {
			continue
		}
		evAt, err := time.Parse(time.RFC3339Nano, a.EvaluatedAt)
		if err != nil || evAt.After(at) {
			continue // an evaluation after the release time cannot qualify it
		}
		if best == nil || evAt.After(bestAt) {
			best, bestAt = a, evAt
		}
	}
	return best, nil
}

// checkTerms requires every term record to exist and be current. It returns
// the exact versions in force, or a reason for exclusion.
func checkTerms(ctx context.Context, st Store, uuids []string) ([]storage.VersionRef, string, error) {
	var refs []storage.VersionRef
	for _, id := range uuids {
		rec, err := st.GetTermRecord(ctx, id)
		if errors.Is(err, storage.ErrNotFound) {
			return nil, "term record " + id + " not found", nil
		} else if err != nil {
			return nil, "", fmt.Errorf("release: term record %s: %w", id, err)
		}
		if rec.State.Status != model.Current {
			return nil, "term record " + id + " is obsolete", nil
		}
		data, err := rec.CanonicalBytes()
		if err != nil {
			return nil, "", fmt.Errorf("release: term record %s: %w", id, err)
		}
		cid, err := storage.ContentID(data)
		if err != nil {
			return nil, "", fmt.Errorf("release: term record %s version: %w", id, err)
		}
		refs = append(refs, storage.VersionRef{UUID: id, CID: cid})
	}
	return refs, "", nil
}

// premiseDependencies keeps the dependency entries that are premise UUIDs.
// Content identifiers name exact versions and are recorded in the attestation
// itself; they do not need to be released alongside.
func premiseDependencies(deps []string) []string {
	var out []string
	for _, d := range deps {
		if looksLikeUUID(d) {
			out = append(out, d)
		}
	}
	return out
}

func looksLikeUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, r := range s {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return false
			}
		default:
			if !strings.ContainsRune("0123456789abcdef", r) {
				return false
			}
		}
	}
	return true
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
