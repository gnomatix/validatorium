package main

import "testing"

func TestExpandTemplates(t *testing.T) {
	templates = map[string]string{
		"Ushr":   transcludedBody("{{#switch:{{{2}}}|AL=at-large|#default={{{2}}}}} district of [[{{{1}}}]]<noinclude>docs</noinclude>"),
		"Greet":  "Hello {{{name|world}}}",
		"Lua":    "{{#invoke:Mod|main}}",
		"Nested": "{{Greet|name={{{1}}}}}!",
	}
	defer func() { templates = nil }()
	cases := map[string]string{
		"{{ushr|Alabama Territory|AL|Territory}}: 1818": "at-large district of [[Alabama Territory]]: 1818",
		"In {{ushr|Ohio|3}}.":                           "In 3 district of [[Ohio]].",
		"{{Greet}} and {{greet|name=Ann}}":              "Hello world and Hello Ann",
		"Say {{Nested|Bob}}":                            "Say Hello Bob!",
		"{{Lua}}x{{Unknown|a}}y":                        "xy",
		"{{#if: yes | A | B}}{{#if: | A | B}}":          "AB",
		"It is {{#ifeq: a | a | same | diff}}":          "It is same",
		"{{Greet|name=[[Paris|the city]]}} now":         "Hello [[Paris|the city]] now",
		"{{Navbox|x}}\nText.\n{{Infobox\n|a=1\n}}":      "\nText.\n",
	}
	for in, want := range cases {
		if got := expandTemplates(in); got != want {
			t.Errorf("expand(%q) = %q, want %q", in, got, want)
		}
	}
	if got := dropBlockTemplates("{{Infobox|a={{x}}\n|b=1}}\nText {{Greet}} here.\n{{Navbox}}"); got != "\nText {{Greet}} here.\n" {
		t.Errorf("dropBlockTemplates = %q", got)
	}
	if got := transcludedBody("a<noinclude>b</noinclude><includeonly>c</includeonly>"); got != "ac" {
		t.Errorf("transcludedBody = %q", got)
	}
}
