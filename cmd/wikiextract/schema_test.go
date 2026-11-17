package main

import (
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// jsonFields lists the JSON names of a struct type's fields and which of
// them are always written (no omitempty).
func jsonFields(t reflect.Type) (all, always []string) {
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("json")
		name := strings.Split(tag, ",")[0]
		all = append(all, name)
		if !strings.Contains(tag, "omitempty") {
			always = append(always, name)
		}
	}
	sort.Strings(all)
	sort.Strings(always)
	return
}

type schemaObj struct {
	Required   []string             `json:"required"`
	Properties map[string]schemaObj `json:"properties"`
	Items      *schemaObj           `json:"items"`
}

func (s schemaObj) names() (props, req []string) {
	for k := range s.Properties {
		props = append(props, k)
	}
	req = append(req, s.Required...)
	sort.Strings(props)
	sort.Strings(req)
	return
}

// TestSchemaMatchesStructs keeps ingest/wikipedia/schemas/article.schema.json
// in step with the Go types that write the records.
func TestSchemaMatchesStructs(t *testing.T) {
	b, err := os.ReadFile("../../ingest/wikipedia/schemas/article.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var s schemaObj
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatal(err)
	}
	check := func(name string, obj schemaObj, typ reflect.Type) {
		props, req := obj.names()
		all, always := jsonFields(typ)
		if !reflect.DeepEqual(props, all) || !reflect.DeepEqual(req, always) {
			t.Errorf("%s: schema properties %v required %v; struct fields %v always written %v", name, props, req, all, always)
		}
	}
	check("article", s, reflect.TypeOf(article{}))
	check("refs", *s.Properties["refs"].Items, reflect.TypeOf(Ref{}))
	check("sentences", *s.Properties["sentences"].Items, reflect.TypeOf(Sentence{}))
}
