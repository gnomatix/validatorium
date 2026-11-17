package main

import (
	"regexp"
	"strings"
)

// reInline matches one innermost call of a template that renderTemplate
// knows, with no nested braces in its arguments.
var reInline = regexp.MustCompile(`(?i)\{\{\s*(convert|cvt|lang|small|nowrap|ill|isbn)\s*\|([^{}]*)\}\}`)

// renderInline replaces the templates that commonly sit inside sentences
// with the text they display, so removing the remaining templates does not
// cut words out of the sentence. Other templates are left for Strip to
// remove. Rules follow the templates' documented displays; arguments are
// taken positionally and named arguments (name=value) are ignored.
func renderInline(s string) string {
	for i := 0; i < 3; i++ { // a few passes for templates nested in arguments
		next := reInline.ReplaceAllStringFunc(s, func(call string) string {
			m := reInline.FindStringSubmatch(call)
			return renderTemplate(strings.ToLower(m[1]), positional(m[2]))
		})
		if next == s {
			break
		}
		s = next
	}
	return s
}

func positional(args string) []string {
	var out []string
	for _, a := range strings.Split(args, "|") {
		if strings.Contains(a, "=") {
			continue
		}
		out = append(out, strings.TrimSpace(a))
	}
	return out
}

var rangeWords = map[string]string{"-": "–", "–": "–", "to": "to", "and": "and", "or": "or", "by": "by", "x": "×", "+/-": "±"}

func renderTemplate(name string, a []string) string {
	arg := func(i int) string {
		if i < len(a) {
			return a[i]
		}
		return ""
	}
	switch name {
	case "convert", "cvt":
		// {{convert|value|unit|...}} or {{convert|v1|range|v2|unit|...}}
		if w, ok := rangeWords[arg(1)]; ok && arg(2) != "" {
			sep := " " + w + " "
			if w == "–" {
				sep = "–"
			}
			return strings.TrimSpace(arg(0) + sep + arg(2) + " " + unit(arg(3)))
		}
		return strings.TrimSpace(arg(0) + " " + unit(arg(1)))
	case "lang", "ill":
		if name == "lang" {
			return arg(1) // {{lang|code|text}}
		}
		return arg(0) // {{ill|Title|lang|...}}
	case "small", "nowrap":
		return arg(0)
	case "isbn":
		return "ISBN " + arg(0)
	}
	return ""
}

// unit spells the temperature codes; other unit codes are kept as written.
func unit(u string) string {
	switch u {
	case "C":
		return "°C"
	case "F":
		return "°F"
	case "K":
		return "K"
	}
	return u
}
