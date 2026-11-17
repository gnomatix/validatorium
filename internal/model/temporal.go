package model

import (
	"fmt"
	"strings"
	"time"
)

// Interval is a half-open time interval [Start, End). An empty Start means
// unbounded in the past; an empty End means still open (e.g. "as of" a date
// with no end yet).
type Interval struct {
	Start string `json:"start,omitempty"`
	End   string `json:"end,omitempty"`
}

// TemporalPolicy is the premise's time scope (docs/specs/data-model.md §8).
// Temporal inactivity or expiry does not make a premise obsolete.
type TemporalPolicy struct {
	// ValidTime is the interval over which the premise holds, e.g. "as of
	// the date Jeff said it" (Start only) or "the formulation they used to
	// make" (Start and End).
	ValidTime *Interval `json:"validTime,omitempty"`
	// ExpiresAt is when the premise stops being current without re-check.
	ExpiresAt string `json:"expiresAt,omitempty"`
	// TTL is how long an evaluation stays good, as a Go duration ("720h").
	TTL string `json:"ttl,omitempty"`
	// Recurrence describes repeating validity windows; recorded, not yet
	// evaluated by ActiveAt.
	Recurrence map[string]interface{} `json:"recurrence,omitempty"`
	Timezone   string                 `json:"timezone,omitempty"`
	Calendar   string                 `json:"calendar,omitempty"`
}

func parseTime(field, v string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return time.Time{}, fmt.Errorf("temporal policy: %s %q is not RFC 3339: %w", field, v, err)
	}
	return t, nil
}

// Validate checks that every time and duration parses and that the
// interval is not inverted.
func (p *TemporalPolicy) Validate() error {
	if p == nil {
		return nil
	}
	if p.ValidTime != nil {
		var s, e time.Time
		var err error
		if p.ValidTime.Start != "" {
			if s, err = parseTime("validTime.start", p.ValidTime.Start); err != nil {
				return err
			}
		}
		if p.ValidTime.End != "" {
			if e, err = parseTime("validTime.end", p.ValidTime.End); err != nil {
				return err
			}
		}
		if !s.IsZero() && !e.IsZero() && !e.After(s) {
			return fmt.Errorf("temporal policy: validTime end %s is not after start %s", p.ValidTime.End, p.ValidTime.Start)
		}
	}
	if p.ExpiresAt != "" {
		if _, err := parseTime("expiresAt", p.ExpiresAt); err != nil {
			return err
		}
	}
	if strings.TrimSpace(p.TTL) != "" {
		if d, err := time.ParseDuration(p.TTL); err != nil || d <= 0 {
			return fmt.Errorf("temporal policy: ttl %q is not a positive duration", p.TTL)
		}
	}
	if p.Timezone != "" {
		if _, err := time.LoadLocation(p.Timezone); err != nil {
			return fmt.Errorf("temporal policy: timezone %q: %w", p.Timezone, err)
		}
	}
	return nil
}

// ActiveAt reports whether the premise's time scope includes t: inside
// ValidTime (when set) and before ExpiresAt (when set). A nil policy is
// always active. Recurrence is not evaluated here.
func (p *TemporalPolicy) ActiveAt(t time.Time) bool {
	if p == nil {
		return true
	}
	if p.ValidTime != nil {
		if p.ValidTime.Start != "" {
			if s, err := time.Parse(time.RFC3339, p.ValidTime.Start); err != nil || t.Before(s) {
				return false
			}
		}
		if p.ValidTime.End != "" {
			if e, err := time.Parse(time.RFC3339, p.ValidTime.End); err != nil || !t.Before(e) {
				return false
			}
		}
	}
	if p.ExpiresAt != "" {
		if x, err := time.Parse(time.RFC3339, p.ExpiresAt); err != nil || !t.Before(x) {
			return false
		}
	}
	return true
}
