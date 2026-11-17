package main

import (
	"bytes"
	"compress/bzip2"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

const sampleDump = `<mediawiki><siteinfo><sitename>Wikipedia</sitename></siteinfo>
<page><title>Ulm</title><ns>0</ns><id>12345</id><revision><id>1250000001</id><text xml:space="preserve">{{Infobox city|name=Ulm|pop=126,000}}
'''Ulm''' is a city in [[Baden-Württemberg]], [[Germany]].&lt;ref name="stat"&gt;{{cite web |url=https://example.org/ulm |title=Ulm statistics}}&lt;/ref&gt; [[Albert Einstein|Einstein]] was born in Ulm in 1879.&lt;ref&gt;{{cite journal |doi=10.1000/xyz |pmid=12345678 |title=Einstein}}&lt;/ref&gt;&lt;ref name="stat"/&gt;
[[File:Ulm Minster.jpg|thumb|The [[Ulm Minster]] spire]]
{| class="wikitable"
| a || b
|}
== History ==
The city was founded around 850 and grew along the [[Danube]] river, which is long enough for this test text to pass the minimum length filter easily.
== References ==
{{reflist}}
[[Category:Cities in Germany]]</text></revision></page>
<page><title>UlmRedirect</title><ns>0</ns><id>2</id><redirect title="Ulm"/><revision><id>3</id><text>#REDIRECT [[Ulm]]</text></revision></page>
<page><title>Talk:Ulm</title><ns>1</ns><id>4</id><revision><id>5</id><text>talk page text</text></revision></page>
</mediawiki>`

func TestExtract(t *testing.T) {
	var out bytes.Buffer
	n, err := extract(strings.NewReader(sampleDump), "enwiki", 0, 50, &out)
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v out=%s", n, err, out.String())
	}
	var a article
	if err := json.Unmarshal(out.Bytes(), &a); err != nil {
		t.Fatal(err)
	}
	if a.PageID != 12345 || a.RevID != 1250000001 || a.Title != "Ulm" {
		t.Fatalf("ids: %+v", a)
	}
	for _, want := range []string{"Ulm is a city in Baden-Württemberg, Germany.", "Einstein was born in Ulm in 1879.", "History.", "Danube river"} {
		if !strings.Contains(a.Text, want) {
			t.Errorf("text missing %q:\n%s", want, a.Text)
		}
	}
	if len(a.Refs) != 2 || a.Refs[0].URL != "https://example.org/ulm" || a.Refs[1].DOI != "10.1000/xyz" || a.Refs[1].PMID != "12345678" {
		t.Fatalf("refs: %+v", a.Refs)
	}
	if !strings.Contains(a.Text, "Germany. [ref:1]") || !strings.Contains(a.Text, "1879. [ref:2] [ref:1]") {
		t.Errorf("ref markers missing:\n%s", a.Text)
	}
	for _, bad := range []string{"Infobox", "cite web", "wikitable", "File:", "Category", "reflist", "[[", "'''", "Minster"} {
		if strings.Contains(a.Text, bad) {
			t.Errorf("text still contains %q:\n%s", bad, a.Text)
		}
	}
}

// TestMultistreamBzip2 checks that concatenated bz2 streams, as in the
// multistream dump, are read in sequence.
func TestMultistreamBzip2(t *testing.T) {
	if _, err := exec.LookPath("bzip2"); err != nil {
		t.Skip("bzip2 not installed")
	}
	half := len(sampleDump) / 2
	var cat bytes.Buffer
	for _, part := range []string{sampleDump[:half], sampleDump[half:]} {
		cmd := exec.Command("bzip2", "-c")
		cmd.Stdin = strings.NewReader(part)
		b, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		cat.Write(b)
	}
	var out bytes.Buffer
	n, err := extract(bzip2.NewReader(&cat), "enwiki", 0, 50, &out)
	if err != nil || n != 1 {
		t.Fatalf("multistream: n=%d err=%v", n, err)
	}
}

func TestTitlesFilter(t *testing.T) {
	titles = map[string]bool{"Somewhere Else": true}
	defer func() { titles = nil }()
	var out bytes.Buffer
	n, err := extract(strings.NewReader(sampleDump), "enwiki", 0, 50, &out)
	if err != nil || n != 0 {
		t.Fatalf("filtered out: n=%d err=%v", n, err)
	}
	titles = map[string]bool{"Ulm": true}
	if n, err := extract(strings.NewReader(sampleDump), "enwiki", 0, 50, &out); err != nil || n != 1 {
		t.Fatalf("kept: n=%d err=%v", n, err)
	}
}

func TestStripDropsUnmatchedTableRows(t *testing.T) {
	in := "First sentence.\n|-id=1 bgcolor=#fefefe\n| 1 || || x\n! Header\n| name = X\nSecond sentence."
	got := Strip(in)
	if strings.Contains(got, "|") || strings.Contains(got, "Header") {
		t.Fatalf("table rows left in text: %q", got)
	}
	if !strings.Contains(got, "First sentence.") || !strings.Contains(got, "Second sentence.") {
		t.Fatalf("prose lost: %q", got)
	}
}

func TestStripKeepsMathSource(t *testing.T) {
	got := Strip("His most famous equation is <math>E = mc^2</math> in which E is energy.")
	if got != "His most famous equation is E = mc^2 in which E is energy." {
		t.Fatalf("got %q", got)
	}
}

func TestStripRemovesBehaviourSwitches(t *testing.T) {
	if got := Strip("__NOTOC__\nWater is wet. __FORCETOC__"); got != "Water is wet." {
		t.Fatalf("got %q", got)
	}
}

func TestStripRendersInlineTemplates(t *testing.T) {
	cases := map[string]string{
		"It is {{convert|8849|m|ft}} high.":                  "It is 8849 m high.",
		"Between {{convert|10|-|20|C}} at night.":            "Between 10–20 °C at night.",
		"From {{convert|5|to|7|km|abbr=on}} away.":           "From 5 to 7 km away.",
		"The word {{lang|fr|bonjour}} means hello.":          "The word bonjour means hello.",
		"She met {{ill|Jean Dupont|fr}} in Paris.":           "She met Jean Dupont in Paris.",
		"A {{small|very}} small {{nowrap|thing}} {{other}}.": "A very small thing.",
	}
	for in, want := range cases {
		if got := Strip(in); got != want {
			t.Errorf("Strip(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestStripDecodesDoubleEncodedEntities(t *testing.T) {
	if got := Strip("From 1990&amp;ndash;2000 &amp; after."); got != "From 1990–2000 & after." {
		t.Fatalf("got %q", got)
	}
}
