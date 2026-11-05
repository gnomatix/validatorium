package model

import (
	"encoding/json"
	"fmt"
)

// CanonicalBytes returns the byte encoding of one immutable version of the
// record. The storage layer content-addresses these bytes (CID); the UUID
// remains the premise's identity across versions.
//
// Canonical form is the JSON encoding of the struct: encoding/json emits
// struct fields in declaration order and sorts map keys, so equal records
// yield equal bytes. Any change to the record, including a lifecycle change,
// produces a new version with a new CID.
func (m *Micropublication) CanonicalBytes() ([]byte, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("canonical micropublication: %w", err)
	}
	return b, nil
}

// CanonicalBytes returns the byte encoding of the attestation record for
// content addressing. Attestations are immutable, so one attestation always
// has one CID.
func (a *ValidityAttestation) CanonicalBytes() ([]byte, error) {
	b, err := json.Marshal(a)
	if err != nil {
		return nil, fmt.Errorf("canonical attestation: %w", err)
	}
	return b, nil
}

// MicropublicationFromBytes decodes bytes produced by CanonicalBytes.
func MicropublicationFromBytes(b []byte) (*Micropublication, error) {
	var m Micropublication
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("decode micropublication: %w", err)
	}
	return &m, nil
}

// AttestationFromBytes decodes bytes produced by ValidityAttestation.CanonicalBytes.
func AttestationFromBytes(b []byte) (*ValidityAttestation, error) {
	var a ValidityAttestation
	if err := json.Unmarshal(b, &a); err != nil {
		return nil, fmt.Errorf("decode attestation: %w", err)
	}
	return &a, nil
}
