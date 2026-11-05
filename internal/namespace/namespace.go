// Package namespace defines the names of the pointer namespaces that organise
// Validatorium records (Beads decision validatorium-7oj).
//
// Records are identified by UUID and stored once. A namespace is a pointer
// layer over those records: a namespaced pointer says "within this namespace,
// this UUID refers to this exact version". The same record may be referenced
// from any number of namespaces, and removing a namespaced pointer never
// touches the record or its versions.
//
// Namespaces organise structure. They carry no evidential weight and do not
// make a premise valid; a record reaches a namespace only after promotion.
package namespace

import (
	"errors"
	"fmt"
	"strings"
)

// Separator separates the segments of a namespace name.
const Separator = ":"

// Name is a validated namespace name of the form `namespace:subdomain:...`.
// Each segment is one or more of a-z, 0-9 or '-', must start with a letter or
// digit, and segments are joined by Separator. Names are case-sensitive and
// only lower case is valid, so two names are equal exactly when their strings
// are equal.
type Name string

// ErrInvalidName is returned for a string that is not a well-formed name.
var ErrInvalidName = errors.New("invalid namespace name")

// Parse validates s and returns it as a Name.
func Parse(s string) (Name, error) {
	if s == "" {
		return "", fmt.Errorf("%w: empty", ErrInvalidName)
	}
	for i, seg := range strings.Split(s, Separator) {
		if err := validateSegment(seg); err != nil {
			return "", fmt.Errorf("%w %q: segment %d: %v", ErrInvalidName, s, i, err)
		}
	}
	return Name(s), nil
}

// MustParse is Parse for compile-time constants; it panics on an invalid name.
func MustParse(s string) Name {
	n, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return n
}

func validateSegment(seg string) error {
	if seg == "" {
		return errors.New("empty segment")
	}
	for i, r := range seg {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
		case r == '-' && i > 0:
		default:
			return fmt.Errorf("character %q not allowed", r)
		}
	}
	return nil
}

// String returns the name.
func (n Name) String() string { return string(n) }

// Segments returns the name's segments in order.
func (n Name) Segments() []string { return strings.Split(string(n), Separator) }

// Parent returns the enclosing namespace, or "" and false for a root name.
func (n Name) Parent() (Name, bool) {
	i := strings.LastIndex(string(n), Separator)
	if i < 0 {
		return "", false
	}
	return Name(n[:i]), true
}

// Child returns the name with one more segment appended.
func (n Name) Child(segment string) (Name, error) {
	return Parse(string(n) + Separator + segment)
}

// Within reports whether n is other or lies beneath it.
func (n Name) Within(other Name) bool {
	if n == other {
		return true
	}
	return strings.HasPrefix(string(n), string(other)+Separator)
}

// Root is the namespace shipped with the software under which the core
// structure is organised. The owner of the software controls what structure
// exists beneath it; that control is over organisation, never over truth.
const Root Name = "validatorium"

// Core lists the namespaces shipped with the binary. Only the root ships for
// now; the owner adds structure beneath it as domains are defined.
func Core() []Name { return []Name{Root} }
