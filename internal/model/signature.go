package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Signature attributes one action on one immutable record version to an
// OpenPGP key (Beads decision validatorium-ilv). It is a separate record that
// names the signed version by content identifier, so signing never changes a
// record's canonical bytes or its CID.
//
// A signature is attribution only: it carries no authority weight and does
// not make the signed record valid (docs/specs/distributed-datastore.md §9).
// Its purpose is remediation: every version a key signed can be enumerated
// and flagged if that key's holder or node is found to have corrupted data.
type Signature struct {
	// SubjectCID is the content identifier of the signed version's canonical
	// bytes.
	SubjectCID string `json:"subjectCID"`
	// Signer is the upper-case hex fingerprint of the signing key.
	Signer string `json:"signer"`
	// Action names what the signer did with the subject, e.g. "promote".
	Action string `json:"action"`
	// ArmoredSignature is an ASCII-armored detached OpenPGP signature over
	// the subject's canonical bytes.
	ArmoredSignature string `json:"armoredSignature"`
	// SignedAt is the RFC 3339 UTC time the signature was made.
	SignedAt string `json:"signedAt"`
}

// ValidateRecord checks that the signature record is well formed. It does not
// verify the cryptographic signature; see internal/identity.
func (s *Signature) ValidateRecord() error {
	switch {
	case s == nil:
		return errors.New("signature: nil record")
	case strings.TrimSpace(s.SubjectCID) == "":
		return errors.New("signature: subjectCID is required")
	case len(s.Signer) != 40 || strings.ToUpper(s.Signer) != s.Signer:
		return fmt.Errorf("signature: signer %q is not an upper-case 40-hex fingerprint", s.Signer)
	case strings.TrimSpace(s.Action) == "":
		return errors.New("signature: action is required")
	case !strings.Contains(s.ArmoredSignature, "BEGIN PGP SIGNATURE"):
		return errors.New("signature: armoredSignature is not an armored OpenPGP signature")
	}
	if _, err := time.Parse(time.RFC3339, s.SignedAt); err != nil {
		return fmt.Errorf("signature: signedAt: %w", err)
	}
	return nil
}

// CanonicalBytes returns the deterministic encoding used for the record's CID.
func (s *Signature) CanonicalBytes() ([]byte, error) {
	if err := s.ValidateRecord(); err != nil {
		return nil, err
	}
	return json.Marshal(s)
}

// SignatureFromBytes decodes and validates a stored signature record.
func SignatureFromBytes(b []byte) (*Signature, error) {
	var s Signature
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("signature: decode: %w", err)
	}
	if err := s.ValidateRecord(); err != nil {
		return nil, err
	}
	return &s, nil
}
