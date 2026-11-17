package main

import (
	"bufio"
	"compress/bzip2"
	"encoding/xml"
	"io"
	"os"
	"regexp"
	"strings"
)

// Template expansion (-templates): the dump's own Template-namespace pages
// supply the template bodies. Expansion is innermost-first with parameter
// substitution and the parser functions #if, #ifeq and #switch. Templates
// that call Lua modules (#invoke) and magic words expand to nothing: they
// cannot run here. This covers templates written in wikitext only.

// templates maps a normalized template name to its transcluded body; nil
// disables expansion.
var templates map[string]string

var (
	reInnermost = regexp.MustCompile(`\{\{([^{}]*)\}\}`)
	reParam     = regexp.MustCompile(`\{\{\{([^{}|]*)(?:\|([^{}]*))?\}\}\}`)
	reNoinclude = regexp.MustCompile(`(?s)<noinclude>.*?</noinclude>`)
	reOnlyIncl  = regexp.MustCompile(`(?s)<onlyinclude>(.*?)</onlyinclude>`)
	reInclOnly  = regexp.MustCompile(`(?s)</?includeonly>`)
)

func normTemplateName(n string) string {
	n = strings.TrimSpace(strings.ReplaceAll(n, "_", " "))
	n = strings.TrimPrefix(n, "Template:")
	n = strings.TrimPrefix(n, "template:")
	if n == "" {
		return n
	}
	return strings.ToUpper(n[:1]) + n[1:]
}

// transcludedBody returns what a template page contributes when used.
func transcludedBody(src string) string {
	src = reComment.ReplaceAllString(src, "") // comments are documentation, never output
	if m := reOnlyIncl.FindAllStringSubmatch(src, -1); m != nil {
		var b strings.Builder
		for _, x := range m {
			b.WriteString(x[1])
		}
		return b.String()
	}
	return reInclOnly.ReplaceAllString(reNoinclude.ReplaceAllString(src, ""), "")
}

// loadTemplates reads every Template-namespace page (ns 10) from the dump.
func loadTemplates(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var r io.Reader = bufio.NewReaderSize(f, 1<<20)
	if strings.HasSuffix(path, ".bz2") {
		r = bzip2.NewReader(r)
	}
	dec := xml.NewDecoder(r)
	out := map[string]string{}
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return out, err
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "page" {
			continue
		}
		var p page
		if err := dec.DecodeElement(&p, &se); err != nil {
			return out, err
		}
		if p.NS == 10 && p.Redirect == nil {
			out[normTemplateName(p.Title)] = transcludedBody(p.Revision.Text)
		}
	}
}

// dropBlockTemplates removes template calls that begin a line (infoboxes,
// navigation boxes, hatnotes): their output is layout that Strip removes,
// so expanding them only costs time. Calls inside a sentence remain.
func dropBlockTemplates(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		lineStart := i == 0 || s[i-1] == '\n'
		if lineStart && strings.HasPrefix(s[i:], "{{") {
			depth, j := 0, i
			for j < len(s) {
				if strings.HasPrefix(s[j:], "{{") {
					depth++
					j += 2
					continue
				}
				if strings.HasPrefix(s[j:], "}}") {
					depth--
					j += 2
					if depth == 0 {
						break
					}
					continue
				}
				j++
			}
			// Only a call that fills its lines (ends at a line break or the
			// end of the text) is a block; "{{x}} text" stays.
			if depth == 0 && (j == len(s) || s[j] == '\n') {
				i = j
				continue
			}
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// expandTemplates expands template calls in s using the loaded templates.
func expandTemplates(s string) string {
	if templates == nil {
		return s
	}
	s = dropBlockTemplates(s)
	for pass := 0; pass < 40 && strings.Contains(s, "{{"); pass++ {
		next := reInnermost.ReplaceAllStringFunc(s, func(call string) string {
			if strings.HasPrefix(call, "{{{") || strings.HasSuffix(call, "}}}") {
				return call // a parameter, handled inside bodies
			}
			return expandCall(call[2 : len(call)-2])
		})
		if next == s {
			break
		}
		s = next
		if len(s) > 4<<20 {
			break // runaway expansion
		}
	}
	return s
}

// splitPipes splits on | outside [[...]] links.
func splitPipes(s string) []string {
	var out []string
	depth, start := 0, 0
	for i := 0; i < len(s); i++ {
		switch {
		case strings.HasPrefix(s[i:], "[["):
			depth++
			i++
		case strings.HasPrefix(s[i:], "]]") && depth > 0:
			depth--
			i++
		case s[i] == '|' && depth == 0:
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}

func splitArgs(inner string) (string, []string, map[string]string) {
	parts := splitPipes(inner)
	named := map[string]string{}
	var pos []string
	for _, a := range parts[1:] {
		if i := strings.IndexByte(a, '='); i > 0 && !strings.ContainsAny(a[:i], "[<") {
			named[strings.TrimSpace(a[:i])] = strings.TrimSpace(a[i+1:])
			continue
		}
		pos = append(pos, a)
	}
	return parts[0], pos, named
}

func expandCall(inner string) string {
	name, pos, named := splitArgs(inner)
	name = strings.TrimSpace(name)
	if strings.HasPrefix(name, "#") {
		return parserFunction(name, inner)
	}
	if i := strings.IndexByte(name, ':'); i >= 0 && !strings.HasPrefix(strings.ToLower(name), "template:") {
		return "" // magic word or namespace-qualified call
	}
	body, ok := templates[normTemplateName(name)]
	if !ok {
		return ""
	}
	return reParam.ReplaceAllStringFunc(body, func(p string) string {
		m := reParam.FindStringSubmatch(p)
		key := strings.TrimSpace(m[1])
		if v, ok := named[key]; ok {
			return v
		}
		if n := atoiSmall(key); n > 0 && n <= len(pos) {
			return strings.TrimSpace(pos[n-1])
		}
		return m[2]
	})
}

func atoiSmall(s string) int {
	if s == "" || len(s) > 3 {
		return 0
	}
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// parserFunction evaluates #if, #ifeq and #switch; anything else is empty.
func parserFunction(name, inner string) string {
	colon := strings.IndexByte(inner, ':')
	if colon < 0 {
		return ""
	}
	fn := strings.ToLower(strings.TrimSpace(inner[1:colon]))
	args := splitPipes(inner[colon+1:])
	arg := func(i int) string {
		if i < len(args) {
			return strings.TrimSpace(args[i])
		}
		return ""
	}
	switch fn {
	case "if":
		if arg(0) != "" {
			return arg(1)
		}
		return arg(2)
	case "ifeq":
		if arg(0) == arg(1) {
			return arg(2)
		}
		return arg(3)
	case "switch":
		v := arg(0)
		def := ""
		fall := false
		for _, c := range args[1:] {
			c = strings.TrimSpace(c)
			if i := strings.IndexByte(c, '='); i >= 0 {
				k, r := strings.TrimSpace(c[:i]), strings.TrimSpace(c[i+1:])
				if fall || k == v {
					return r
				}
				if k == "#default" {
					def = r
				}
				continue
			}
			if c == v {
				fall = true
			} else if !strings.Contains(c, "=") {
				def = c
			}
		}
		return def
	}
	return ""
}
