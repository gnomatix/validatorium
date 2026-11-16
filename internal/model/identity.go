package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var fingerprintRe = regexp.MustCompile(`^[0-9A-F]{40}([0-9A-F]{24})?$`)

// Identity is a registered OpenPGP key. It identifies an actor for
// attribution, like a wiki editor account: it traces actions to one key, not
// to a person, and carries no evidential weight. An identity can be
// obsoleted (by its owner or by anyone, with the agent and reason recorded);
// signatures made with an obsoleted identity's key no longer count as
// verified. Every version stays on record.
type Identity struct {
	// UUID is the key fingerprint, upper-case hex.
	UUID           string      `json:"_id"`
	PublicKey      string      `json:"publicKey"`
	Attribution    Attribution `json:"attribution"`
	State          Lifecycle   `json:"state"`
	ObsoletedBy    string      `json:"obsoletedBy,omitempty"`
	ObsoleteReason string      `json:"obsoleteReason,omitempty"`
}

// ValidateRecord checks that the identity is well-formed.
func (i *Identity) ValidateRecord() error {
	if !fingerprintRe.MatchString(i.UUID) {
		return fmt.Errorf("identity %q: id must be an upper-case hex OpenPGP fingerprint", i.UUID)
	}
	if !strings.Contains(i.PublicKey, "BEGIN PGP PUBLIC KEY BLOCK") {
		return fmt.Errorf("identity %s: public key must be ASCII-armored", i.UUID)
	}
	switch i.State.Status {
	case Current:
		if i.ObsoletedBy != "" {
			return fmt.Errorf("identity %s: a current identity carries no obsoleting fields", i.UUID)
		}
	case Obsolete:
		if strings.TrimSpace(i.ObsoletedBy) == "" {
			return fmt.Errorf("identity %s: an obsolete identity must name who obsoleted it", i.UUID)
		}
	default:
		return fmt.Errorf("identity %s: invalid lifecycle status %q", i.UUID, i.State.Status)
	}
	return nil
}

// Obsoleted returns the obsoleting version of i; i is not changed.
func (i *Identity) Obsoleted(by, reason string) (*Identity, error) {
	if i.State.Status == Obsolete {
		return nil, errors.New("identity " + i.UUID + " is already obsolete")
	}
	n := *i
	n.State = Lifecycle{Status: Obsolete}
	n.ObsoletedBy, n.ObsoleteReason = by, reason
	return &n, n.ValidateRecord()
}

// CanonicalBytes returns the encoding of one immutable version.
func (i *Identity) CanonicalBytes() ([]byte, error) { return json.Marshal(i) }

// IdentityFromBytes decodes CanonicalBytes output.
func IdentityFromBytes(b []byte) (*Identity, error) {
	var i Identity
	if err := json.Unmarshal(b, &i); err != nil {
		return nil, fmt.Errorf("decode identity: %w", err)
	}
	return &i, nil
}

// SignedPayload is the exact text a signer signs to certify c. It covers
// every field that states what was certified.
func (c *SelfCertification) SignedPayload() string {
	return "validatorium self-certification v1\n" + c.Standard + "\n" + c.Subject + "\n" + c.Method + "\n" + c.Date + "\n"
}
