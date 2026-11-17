package main

import (
	"reflect"
	"testing"
)

func TestSentences(t *testing.T) {
	text := "Hydrogen is the lightest element. [ref:1] [ref:2] It has the symbol H.\n" +
		"Dr. Smith measured it, e.g. in 1898, with J. Dewar. The value is 1.008 [ref:3]. Is it stable? Yes!\n" +
		"No final period here [ref:4]"
	got := Sentences(text)
	want := []struct {
		text string
		refs []int
	}{
		{"Hydrogen is the lightest element.", []int{1, 2}},
		{"It has the symbol H.", nil},
		{"Dr. Smith measured it, e.g. in 1898, with J. Dewar.", nil},
		{"The value is 1.008.", []int{3}},
		{"Is it stable?", nil},
		{"Yes!", nil},
		{"No final period here", []int{4}},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d sentences: %+v", len(got), got)
	}
	for i, w := range want {
		if got[i].Text != w.text || !reflect.DeepEqual(got[i].Refs, w.refs) {
			t.Errorf("sentence %d = %q %v, want %q %v", i, got[i].Text, got[i].Refs, w.text, w.refs)
		}
		span := text[got[i].Start:got[i].End]
		if len(span) == 0 || span[0] == ' ' || span[len(span)-1] == ' ' {
			t.Errorf("sentence %d span %q not trimmed", i, span)
		}
	}
}
