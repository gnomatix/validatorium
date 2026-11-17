package main

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Sentence is one sentence of an article's text, found by rules, not by a
// model. Start and End are byte offsets into the article text, so
// text[Start:End] is the verbatim span, citation markers included. Text is
// that span with the [ref:N] markers removed and spaces closed up; Refs
// lists the N of every marker in the span, including markers that follow
// the sentence's final punctuation.
type Sentence struct {
	Start int    `json:"start"`
	End   int    `json:"end"`
	Text  string `json:"text"`
	Refs  []int  `json:"refs,omitempty"`
}

var (
	reRefMarker = regexp.MustCompile(`\[ref:(\d+)\]`)
	reRefStrip  = regexp.MustCompile(`\s*\[ref:\d+\]`)
	reSpaces2   = regexp.MustCompile(`\s{2,}`)
	// abbreviations whose final period does not end a sentence (lower case,
	// without the period).
	abbrevs = map[string]bool{
		"e.g": true, "i.e": true, "etc": true, "vs": true, "cf": true, "ca": true, "c": true,
		"approx": true, "fig": true, "no": true, "nos": true, "vol": true, "pp": true, "p": true,
		"mr": true, "mrs": true, "ms": true, "dr": true, "prof": true, "st": true, "mt": true,
		"jr": true, "sr": true, "inc": true, "ltd": true, "co": true, "corp": true, "jan": true,
		"feb": true, "mar": true, "apr": true, "jun": true, "jul": true, "aug": true, "sep": true,
		"sept": true, "oct": true, "nov": true, "dec": true, "u.s": true, "u.k": true, "a.d": true,
		"b.c": true, "op": true, "ed": true, "eds": true, "al": true, "gen": true, "gov": true,
		"sen": true, "rep": true, "rev": true, "sgt": true, "col": true, "lt": true, "capt": true,
	}
)

// Sentences splits article text into sentences. A line break always ends
// a sentence. Within a line, ".", "!" or "?" (with any closing quotes or
// brackets) ends a sentence when what follows, after spaces and citation
// markers, is the end of the line or starts with an upper-case letter, a
// digit, a quote or an opening bracket, and the word before a period is
// neither a known abbreviation nor a single letter (an initial).
func Sentences(text string) []Sentence {
	var out []Sentence
	lineStart := 0
	for lineStart < len(text) {
		lineEnd := strings.IndexByte(text[lineStart:], '\n')
		if lineEnd < 0 {
			lineEnd = len(text)
		} else {
			lineEnd += lineStart
		}
		out = appendLine(out, text, lineStart, lineEnd)
		lineStart = lineEnd + 1
	}
	return out
}

func appendLine(out []Sentence, text string, start, end int) []Sentence {
	s := start
	for i := start; i < end; i++ {
		c := text[i]
		if c != '.' && c != '!' && c != '?' {
			continue
		}
		j := i + 1
		for j < end && strings.IndexByte(`"')]’”`, text[j]) >= 0 {
			j++
		}
		punctEnd := j
		// Include citation markers (and the spaces around them) that follow.
		k := j
		for {
			m := k
			for m < end && text[m] == ' ' {
				m++
			}
			if loc := reRefMarker.FindStringIndex(text[m:end]); loc != nil && loc[0] == 0 {
				k = m + loc[1]
				j = k
				continue
			}
			k = m
			break
		}
		// Mid-line, a boundary needs a space or a marker after the
		// punctuation ("1.008" and "e.g" are not boundaries).
		if k < end && (k == punctEnd || !startsSentence(text[k:end])) {
			continue
		}
		if c == '.' && isAbbrevBefore(text[s:i]) {
			continue
		}
		out = appendSentence(out, text, s, j)
		s = k
		i = k - 1
	}
	if strings.TrimSpace(text[s:end]) != "" {
		out = appendSentence(out, text, s, end)
	}
	return out
}

func startsSentence(rest string) bool {
	r, _ := utf8.DecodeRuneInString(rest)
	return unicode.IsUpper(r) || unicode.IsDigit(r) || strings.ContainsRune(`"'(“‘[`, r)
}

// isAbbrevBefore reports whether the word ending at the period is an
// abbreviation or a single-letter initial.
func isAbbrevBefore(before string) bool {
	w := before
	if i := strings.LastIndexAny(w, " (\t"); i >= 0 {
		w = w[i+1:]
	}
	if utf8.RuneCountInString(w) == 1 {
		r, _ := utf8.DecodeRuneInString(w)
		return unicode.IsUpper(r)
	}
	return abbrevs[strings.ToLower(w)]
}

func appendSentence(out []Sentence, text string, start, end int) []Sentence {
	for start < end && text[start] == ' ' {
		start++
	}
	for end > start && text[end-1] == ' ' {
		end--
	}
	if start >= end {
		return out
	}
	span := text[start:end]
	var refs []int
	for _, m := range reRefMarker.FindAllStringSubmatch(span, -1) {
		if n, err := strconv.Atoi(m[1]); err == nil {
			refs = append(refs, n)
		}
	}
	clean := strings.TrimSpace(reSpaces2.ReplaceAllString(reRefStrip.ReplaceAllString(span, ""), " "))
	return append(out, Sentence{Start: start, End: end, Text: clean, Refs: refs})
}
