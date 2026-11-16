package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// SealOfTruth is the ITVB self-certification standard
// (https://truth.dataglut.org, "Seal of Truth").
const SealOfTruth = "seal-of-truth"

// SelfCertification records that an author certified a piece of content
// under a self-certification standard such as the ITVB Seal of Truth.
//
// It is a separate record class from a premise. It records what its author
// certified and how; it does not make the content valid in Validatorium's
// sense, which only an evidence-based evaluation can do. Like every record it
// is versioned and never deleted: anyone may obsolete it, the obsoleting
// version names who did so and why, and earlier versions stay addressable.
type SelfCertification struct {
	UUID string `json:"_id"`
	// Subject is the URL of the certified content.
	Subject string `json:"subject"`
	// Standard is the certification standard, e.g. SealOfTruth.
	Standard string `json:"standard"`
	// Method is the verification method the author states, e.g.
	// "frontier-model-analysis".
	Method string `json:"method"`
	// Date is the verification date the author states (YYYY-MM-DD).
	Date string `json:"date"`
	// Signer identifies the certifying author, e.g. an OpenPGP fingerprint.
	// Attribution only; it confers no evidential weight.
	Signer string `json:"signer"`
	// SignerVerified is true only when the signer's signature over this
	// record has been checked. Until GPG verification is implemented it is
	// always false.
	SignerVerified bool        `json:"signerVerified"`
	Attribution    Attribution `json:"attribution"`
	State          Lifecycle   `json:"state"`
	// ObsoletedBy and ObsoleteReason are set on the version that obsoleted
	// the record.
	ObsoletedBy    string `json:"obsoletedBy,omitempty"`
	ObsoleteReason string `json:"obsoleteReason,omitempty"`
}

// NewSelfCertification builds a current self-certification with a fresh UUID.
func NewSelfCertification(subject, standard, method, date, signer string, attribution Attribution) (*SelfCertification, error) {
	id, err := newUUIDv4()
	if err != nil {
		return nil, fmt.Errorf("generate uuid: %w", err)
	}
	c := &SelfCertification{
		UUID: id, Subject: subject, Standard: standard, Method: method, Date: date,
		Signer: signer, Attribution: attribution, State: Lifecycle{Status: Current},
	}
	return c, c.ValidateRecord()
}

// ValidateRecord checks that the record is well-formed.
func (c *SelfCertification) ValidateRecord() error {
	if strings.TrimSpace(c.UUID) == "" {
		return errors.New("self-certification: missing UUID")
	}
	u, err := url.Parse(c.Subject)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("self-certification %s: subject must be an http(s) URL", c.UUID)
	}
	if strings.TrimSpace(c.Standard) == "" || strings.TrimSpace(c.Method) == "" {
		return fmt.Errorf("self-certification %s: standard and method are required", c.UUID)
	}
	if _, err := time.Parse("2006-01-02", c.Date); err != nil {
		return fmt.Errorf("self-certification %s: date must be YYYY-MM-DD", c.UUID)
	}
	if strings.TrimSpace(c.Signer) == "" {
		return fmt.Errorf("self-certification %s: signer is required", c.UUID)
	}
	switch c.State.Status {
	case Current:
		if c.ObsoletedBy != "" || len(c.State.SupersededBy) != 0 {
			return fmt.Errorf("self-certification %s: a current record carries no obsoleting fields", c.UUID)
		}
	case Obsolete:
		if strings.TrimSpace(c.ObsoletedBy) == "" {
			return fmt.Errorf("self-certification %s: an obsolete record must name who obsoleted it", c.UUID)
		}
	default:
		return fmt.Errorf("self-certification %s: invalid lifecycle status %q", c.UUID, c.State.Status)
	}
	return nil
}

// Obsoleted returns the obsoleting version of c: same UUID, obsolete, with
// the agent, reason and any replacement UUIDs. c itself is not changed.
func (c *SelfCertification) Obsoleted(by, reason string, supersededBy []string) (*SelfCertification, error) {
	if c.State.Status == Obsolete {
		return nil, fmt.Errorf("self-certification %s is already obsolete", c.UUID)
	}
	n := *c
	n.State = Lifecycle{Status: Obsolete, SupersededBy: append([]string(nil), supersededBy...)}
	n.ObsoletedBy = by
	n.ObsoleteReason = reason
	return &n, n.ValidateRecord()
}

// CanonicalBytes returns the encoding of one immutable version.
func (c *SelfCertification) CanonicalBytes() ([]byte, error) { return json.Marshal(c) }

// SelfCertificationFromBytes decodes CanonicalBytes output.
func SelfCertificationFromBytes(b []byte) (*SelfCertification, error) {
	var c SelfCertification
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("decode self-certification: %w", err)
	}
	return &c, nil
}
