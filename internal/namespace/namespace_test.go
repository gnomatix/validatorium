package namespace

import (
	"errors"
	"testing"
)

func TestParseAcceptsWellFormedNames(t *testing.T) {
	for _, s := range []string{"validatorium", "validatorium:chemistry", "a:b:c-d:e2", "0:1"} {
		n, err := Parse(s)
		if err != nil {
			t.Errorf("Parse(%q): %v", s, err)
			continue
		}
		if n.String() != s {
			t.Errorf("Parse(%q) = %q", s, n)
		}
	}
}

func TestParseRejectsMalformedNames(t *testing.T) {
	for _, s := range []string{"", ":", "a:", ":a", "a::b", "A:b", "a b", "a:-b", "-a", "a:b/c", "a.b", "ä"} {
		if _, err := Parse(s); !errors.Is(err, ErrInvalidName) {
			t.Errorf("Parse(%q): expected ErrInvalidName, got %v", s, err)
		}
	}
}

func TestNameStructure(t *testing.T) {
	n := MustParse("validatorium:chemistry:thermodynamics")
	if segs := n.Segments(); len(segs) != 3 || segs[1] != "chemistry" {
		t.Fatalf("Segments = %v", segs)
	}
	p, ok := n.Parent()
	if !ok || p != "validatorium:chemistry" {
		t.Fatalf("Parent = %q, %v", p, ok)
	}
	if _, ok := Root.Parent(); ok {
		t.Fatal("root must have no parent")
	}
	c, err := Root.Child("physics")
	if err != nil || c != "validatorium:physics" {
		t.Fatalf("Child = %q, %v", c, err)
	}
	if _, err := Root.Child("Bad Segment"); err == nil {
		t.Fatal("invalid child segment accepted")
	}
	if !n.Within(Root) || !n.Within("validatorium:chemistry") || !n.Within(n) {
		t.Fatal("Within should hold for ancestors and self")
	}
	if n.Within("validatorium:chem") || Root.Within(n) {
		t.Fatal("Within must match whole segments only and never descendants")
	}
	if core := Core(); len(core) != 1 || core[0] != Root {
		t.Fatalf("Core = %v", core)
	}
}
