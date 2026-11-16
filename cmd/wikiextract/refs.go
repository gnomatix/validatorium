package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Ref is one citation from an article, numbered in order of first use. The
// text keeps a marker "[ref:N]" where the citation was attached, so a worker
// can tie each premise to the sources the article cites for it.
type Ref struct {
	N     int    `json:"n"`
	URL   string `json:"url,omitempty"`
	DOI   string `json:"doi,omitempty"`
	PMID  string `json:"pmid,omitempty"`
	ISBN  string `json:"isbn,omitempty"`
	Title string `json:"title,omitempty"`
	Raw   string `json:"raw"`
}

var (
	reRefOpen  = regexp.MustCompile(`(?is)<ref([^>]*?)(/>|>(.*?)</ref>)`)
	reRefName  = regexp.MustCompile(`(?i)name\s*=\s*(?:"([^"]+)"|'([^']+)'|([^\s/>]+))`)
	reBareURL  = regexp.MustCompile(`https?://[^\s\]|}<]+`)
	reParamVal = func(name string) *regexp.Regexp {
		return regexp.MustCompile(`(?i)\|\s*` + name + `\s*=\s*([^|}]*)`)
	}
	reURLParam   = reParamVal("url")
	reDOIParam   = reParamVal("doi")
	rePMIDParam  = reParamVal("pmid")
	reISBNParam  = reParamVal("isbn")
	reTitleParam = reParamVal("title")
)

// extractRefs replaces each <ref> with a "[ref:N]" marker and returns the
// citations. Named references reused later (<ref name="x"/>) get the number
// of their first use.
func extractRefs(s string) (string, []Ref) {
	var refs []Ref
	byName := map[string]int{}
	out := reRefOpen.ReplaceAllStringFunc(s, func(m string) string {
		sub := reRefOpen.FindStringSubmatch(m)
		attrs, body := sub[1], strings.TrimSpace(sub[3])
		name := ""
		if nm := reRefName.FindStringSubmatch(attrs); nm != nil {
			name = nm[1] + nm[2] + nm[3]
		}
		if body == "" { // self-closing reuse
			if n, ok := byName[name]; ok {
				return fmt.Sprintf(" [ref:%d]", n)
			}
			return ""
		}
		if n, ok := byName[name]; ok && name != "" {
			return fmt.Sprintf(" [ref:%d]", n)
		}
		r := parseRef(len(refs)+1, body)
		refs = append(refs, r)
		if name != "" {
			byName[name] = r.N
		}
		return " [ref:" + strconv.Itoa(r.N) + "]"
	})
	return out, refs
}

func parseRef(n int, body string) Ref {
	r := Ref{N: n, Raw: body}
	get := func(re *regexp.Regexp) string {
		if m := re.FindStringSubmatch(body); m != nil {
			return strings.TrimSpace(m[1])
		}
		return ""
	}
	r.URL, r.DOI, r.PMID, r.ISBN, r.Title = get(reURLParam), get(reDOIParam), get(rePMIDParam), get(reISBNParam), get(reTitleParam)
	if r.URL == "" {
		r.URL = reBareURL.FindString(body)
	}
	return r
}
