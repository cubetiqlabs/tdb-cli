package cli

import (
	"bytes"
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	"github.com/spf13/cobra"
)

func TestCollectDocumentFieldTypes(t *testing.T) {
	summary := make(map[string]map[string]struct{})
	payload := map[string]any{
		"name":  "Alice",
		"age":   json.Number("29"),
		"score": 82.5,
		"tags":  []any{"pro", "beta"},
		"settings": map[string]any{
			"enabled": true,
			"meta": map[string]any{
				"count": json.Number("3"),
			},
		},
		"items": []any{
			map[string]any{"id": "a1", "qty": json.Number("2")},
			map[string]any{"id": "b2", "qty": json.Number("4")},
		},
	}

	collectDocumentFieldTypes(summary, payload, "")

	got := make(map[string][]string)
	for field, types := range summary {
		list := make([]string, 0, len(types))
		for typ := range types {
			list = append(list, typ)
		}
		sort.Strings(list)
		got[field] = list
	}

	expected := map[string][]string{
		"name":                {"string"},
		"age":                 {"integer"},
		"score":               {"number"},
		"tags":                {"array"},
		"tags[]":              {"string"},
		"settings":            {"object"},
		"settings.enabled":    {"boolean"},
		"settings.meta":       {"object"},
		"settings.meta.count": {"integer"},
		"items":               {"array"},
		"items[]":             {"object"},
		"items[].id":          {"string"},
		"items[].qty":         {"integer"},
	}

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected field type summary: got %+v want %+v", got, expected)
	}
}

func TestExtractSchemaProperties(t *testing.T) {
	schema := `{
        "type": "object",
        "properties": {
            "name": {"type": "string"},
            "settings": {
                "type": "object",
                "properties": {
                    "enabled": {"type": "boolean"},
                    "labels": {
                        "type": "array",
                        "items": {"type": "string"}
                    }
                }
            },
            "items": {
                "type": "array",
                "items": {
                    "type": "object",
                    "properties": {
                        "id": {"type": "string"},
                        "qty": {"type": "integer"}
                    }
                }
            }
        }
    }`

	fields := extractSchemaProperties(schema)
	expected := map[string]struct{}{
		"name":              {},
		"settings":          {},
		"settings.enabled":  {},
		"settings.labels":   {},
		"settings.labels[]": {},
		"items":             {},
		"items[]":           {},
		"items[].id":        {},
		"items[].qty":       {},
	}

	if len(fields) != len(expected) {
		t.Fatalf("unexpected field count: got %d want %d", len(fields), len(expected))
	}
	for field := range expected {
		if _, ok := fields[field]; !ok {
			t.Fatalf("expected field %q missing", field)
		}
	}
	for field := range fields {
		if _, ok := expected[field]; !ok {
			t.Fatalf("unexpected field %q found", field)
		}
	}
}

func TestPrintInferredDocumentSummary(t *testing.T) {
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)

	summary := map[string][]string{
		"settings.enabled": {"boolean"},
		"extra":            {"string"},
		"name":             {"string"},
	}
	schemaFields := map[string]struct{}{
		"settings":         {},
		"settings.enabled": {},
		"name":             {},
	}

	flagged := printInferredDocumentSummary(cmd, summary, 5, true, schemaFields)
	if !flagged {
		t.Fatalf("expected missing schema fields to be flagged")
	}

	expected := "  Inferred fields (sample of up to 5 documents):\n    - extra: string *\n    - name: string\n    - settings.enabled: boolean\n"
	if got := out.String(); got != expected {
		t.Fatalf("unexpected output:\n%s\nwant:\n%s", got, expected)
	}
}
