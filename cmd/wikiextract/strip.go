package main

import (
	"html"
	"regexp"
	"strings"
)

var (
	reComment  = regexp.MustCompile(`(?s)<!--.*?-->`)
	reDropTags = regexp.MustCompile(`(?is)<(gallery|syntaxhighlight|source|timeline|score|imagemap|graph|mapframe)[^>]*>.*?</(gallery|syntaxhighlight|source|timeline|score|imagemap|graph|mapframe)>`)
	reSwitch   = regexp.MustCompile(`__[A-Z]+__`)
	reMath     = regexp.MustCompile(`(?is)<math[^>]*>(.*?)</math>`)
	reTag      = regexp.MustCompile(`</?[a-zA-Z][^>]*>`)
	reExtLink  = regexp.MustCompile(`\[(?:https?:)?//[^\s\]]+\s+([^\]]+)\]`)
	reBareExt  = regexp.MustCompile(`\[(?:https?:)?//[^\]]+\]`)
	reHeading  = regexp.MustCompile(`(?m)^=+\s*(.*?)\s*=+\s*$`)
	reBoldItal = regexp.MustCompile(`'{2,5}`)
	reListMark = regexp.MustCompile(`(?m)^[ \t]*[*#:;]+[ \t]*`)
	reTableRow = regexp.MustCompile(`(?m)^[ \t]*[|!].*$`)
	reSpacePun = regexp.MustCompile(`[ \t]+([.,;:!?])`)
	reEntity   = regexp.MustCompile(`&(?:[a-zA-Z]{2,8}|#[0-9]{2,6}|#x[0-9a-fA-F]{2,5});`)
	reSpaces   = regexp.MustCompile(`[ \t]+`)
	reBlank    = regexp.MustCompile(`\n{3,}`)
	reStopHead = regexp.MustCompile(`(?mi)^==\s*(references|notes|see also|external links|further reading|bibliography|sources|citations)\s*==\s*$`)
)

// Strip removes wikitext markup and returns readable plain text. It is a
// heuristic, sufficient as worker input: templates, tables, files and
// categories are dropped; citations must already be replaced by [ref:N]
// markers (extractRefs); links keep their visible text; sections
// from "References"/"See also"/"External links" onward are cut.
func Strip(s string) string {
	if loc := reStopHead.FindStringIndex(s); loc != nil {
		s = s[:loc[0]]
	}
	s = reComment.ReplaceAllString(s, "")
	s = reDropTags.ReplaceAllString(s, "")
	s = reSwitch.ReplaceAllString(s, "") // behaviour switches such as __NOTOC__
	s = reMath.ReplaceAllString(s, "$1") // keep the formula source so the sentence stays whole
	s = renderInline(s)
	s = expandTemplates(s)
	s = removeNested(s, "{{", "}}")
	s = removeNested(s, "{|", "|}")
	s = links(s)
	s = reExtLink.ReplaceAllString(s, "$1")
	s = reBareExt.ReplaceAllString(s, "")
	s = reTag.ReplaceAllString(s, "")
	s = reHeading.ReplaceAllString(s, "$1.")
	s = reBoldItal.ReplaceAllString(s, "")
	s = reListMark.ReplaceAllString(s, "")
	s = reTableRow.ReplaceAllString(s, "") // rows of tables and infoboxes whose opening was not matched
	s = html.UnescapeString(s)
	// Wikitext that writes an entity as text ("&amp;ndash;") decodes to the
	// entity once more; decode those to the character.
	s = reEntity.ReplaceAllStringFunc(s, html.UnescapeString)
	s = reSpaces.ReplaceAllString(s, " ")
	s = reSpacePun.ReplaceAllString(s, "$1") // space left where a template was removed
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	s = reBlank.ReplaceAllString(strings.Join(lines, "\n"), "\n\n")
	return strings.TrimSpace(s)
}

// removeNested deletes every balanced open…close span, including nested ones.
func removeNested(s, open, close string) string {
	var b strings.Builder
	depth := 0
	for i := 0; i < len(s); {
		switch {
		case strings.HasPrefix(s[i:], open):
			depth++
			i += len(open)
		case depth > 0 && strings.HasPrefix(s[i:], close):
			depth--
			i += len(close)
		default:
			if depth == 0 {
				b.WriteByte(s[i])
			}
			i++
		}
	}
	return b.String()
}

// links replaces [[target|label]] with label and [[target]] with target, and
// drops file, image and category links (which may nest links in captions).
func links(s string) string {
	var b strings.Builder
	for {
		i := strings.Index(s, "[[")
		if i < 0 {
			b.WriteString(s)
			return b.String()
		}
		b.WriteString(s[:i])
		// find the matching ]] allowing nesting
		depth, j := 0, i
		for j < len(s) {
			if strings.HasPrefix(s[j:], "[[") {
				depth++
				j += 2
				continue
			}
			if strings.HasPrefix(s[j:], "]]") {
				depth--
				j += 2
				if depth == 0 {
					break
				}
				continue
			}
			j++
		}
		if depth != 0 { // unbalanced: drop the rest of the markup marker
			b.WriteString(s[i+2:])
			return b.String()
		}
		inner := s[i+2 : j-2]
		s = s[j:]
		lower := strings.ToLower(inner)
		if strings.HasPrefix(lower, "file:") || strings.HasPrefix(lower, "image:") || strings.HasPrefix(lower, "category:") {
			continue
		}
		if k := strings.LastIndex(inner, "|"); k >= 0 {
			inner = inner[k+1:]
		}
		if strings.Contains(inner, "[[") {
			inner = links(inner)
		}
		b.WriteString(inner)
	}
}
