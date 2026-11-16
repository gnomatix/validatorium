// Package identity checks OpenPGP keys and signatures for Validatorium
// identities. A registered key identifies an actor for attribution only.
package identity

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
)

// ProofPrefix begins the text a key owner signs to register the key.
const ProofPrefix = "validatorium identity registration v1\n"

// ProofText is the exact text whose signature proves possession of the key
// with this fingerprint.
func ProofText(fingerprint string) string { return ProofPrefix + fingerprint + "\n" }

// Fingerprint parses an ASCII-armored public key holding exactly one primary
// key and returns its upper-case hex fingerprint.
func Fingerprint(armoredKey string) (string, error) {
	ents, err := openpgp.ReadArmoredKeyRing(strings.NewReader(armoredKey))
	if err != nil {
		return "", fmt.Errorf("identity: read public key: %w", err)
	}
	if len(ents) != 1 {
		return "", fmt.Errorf("identity: expected one key, got %d", len(ents))
	}
	return strings.ToUpper(fmt.Sprintf("%x", ents[0].PrimaryKey.Fingerprint)), nil
}

// VerifyDetached checks an ASCII-armored detached signature over message
// made by armoredKey, and returns the key's fingerprint.
func VerifyDetached(armoredKey, message, armoredSig string) (string, error) {
	ring, err := openpgp.ReadArmoredKeyRing(strings.NewReader(armoredKey))
	if err != nil {
		return "", fmt.Errorf("identity: read public key: %w", err)
	}
	if len(ring) != 1 {
		return "", fmt.Errorf("identity: expected one key, got %d", len(ring))
	}
	block, err := armor.Decode(strings.NewReader(armoredSig))
	if err != nil {
		return "", fmt.Errorf("identity: read signature: %w", err)
	}
	if block.Type != openpgp.SignatureType {
		return "", errors.New("identity: armored block is not a signature")
	}
	signer, err := openpgp.CheckDetachedSignature(ring, bytes.NewReader([]byte(message)), block.Body, nil)
	if err != nil {
		return "", fmt.Errorf("identity: signature does not verify: %w", err)
	}
	return strings.ToUpper(fmt.Sprintf("%x", signer.PrimaryKey.Fingerprint)), nil
}

// Signer makes detached signatures with one OpenPGP private key.
type Signer struct {
	entity      *openpgp.Entity
	fingerprint string
}

// NewSigner reads an ASCII-armored private key. The key must not be
// passphrase-protected; decrypt it before handing it to the node.
func NewSigner(armoredPrivateKey string) (*Signer, error) {
	ring, err := openpgp.ReadArmoredKeyRing(strings.NewReader(armoredPrivateKey))
	if err != nil {
		return nil, fmt.Errorf("identity: read private key: %w", err)
	}
	if len(ring) != 1 || ring[0].PrivateKey == nil {
		return nil, errors.New("identity: expected exactly one private key")
	}
	if ring[0].PrivateKey.Encrypted {
		return nil, errors.New("identity: private key is passphrase-protected")
	}
	return &Signer{entity: ring[0], fingerprint: strings.ToUpper(hex.EncodeToString(ring[0].PrimaryKey.Fingerprint))}, nil
}

// Fingerprint returns the signing key's upper-case hex fingerprint.
func (s *Signer) Fingerprint() string { return s.fingerprint }

// SignDetached returns an ASCII-armored detached signature over data.
func (s *Signer) SignDetached(data []byte) (string, error) {
	var buf bytes.Buffer
	if err := openpgp.ArmoredDetachSign(&buf, s.entity, bytes.NewReader(data), nil); err != nil {
		return "", fmt.Errorf("identity: sign: %w", err)
	}
	return buf.String(), nil
}
