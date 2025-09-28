package cli

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func TestDecodeCollectionSyncPayload_Array(t *testing.T) {
	payload := []byte(`[{"name":"users","schema":{"type":"object"}}]`)
	entries, err := decodeCollectionSyncPayload(payload)
	if err != nil {
		t.Fatalf("decodeCollectionSyncPayload returned error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Name != "users" {
		t.Fatalf("expected name users, got %s", entries[0].Name)
	}
	schema, err := entries[0].schemaString()
	if err != nil {
		t.Fatalf("schemaString returned error: %v", err)
	}
	if schema != `{"type":"object"}` {
		t.Fatalf("unexpected schema: %s", schema)
	}
}

func TestDecodeCollectionSyncPayload_Map(t *testing.T) {
	payload := []byte(`{"users":{"schema":{"type":"object"}}}`)
	entries, err := decodeCollectionSyncPayload(payload)
	if err != nil {
		t.Fatalf("decodeCollectionSyncPayload returned error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Name != "users" {
		t.Fatalf("expected name users, got %s", entries[0].Name)
	}
}

func TestCollectionSyncPayloadSchemaStringVariants(t *testing.T) {
	cases := []struct {
		name    string
		payload collectionSyncPayload
		expect  string
		wantErr error
	}{
		{
			name: "embedded object",
			payload: collectionSyncPayload{
				Schema: []byte(`{"type":"object"}`),
			},
			expect: `{"type":"object"}`,
		},
		{
			name: "string wrapped",
			payload: collectionSyncPayload{
				Schema: []byte(`" {\"type\":\"object\"} "`),
			},
			expect: `{"type":"object"}`,
		},
		{
			name:    "empty",
			payload: collectionSyncPayload{},
			expect:  "",
		},
		{
			name: "null",
			payload: collectionSyncPayload{
				Schema: []byte("null"),
			},
			expect: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			schema, err := tc.payload.schemaString()
			if !errors.Is(err, tc.wantErr) {
				if tc.wantErr != nil || err != nil {
					t.Fatalf("unexpected error: %v, want %v", err, tc.wantErr)
				}
			}
			if schema != tc.expect {
				t.Fatalf("expected %q, got %q", tc.expect, schema)
			}
		})
	}
}

func TestDecodeDocumentSyncPayload(t *testing.T) {
	single := []byte(`{"key":"user-1"}`)
	docs, err := decodeDocumentSyncPayload(single)
	if err != nil {
		t.Fatalf("single decode error: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}

	arrayPayload := []byte(`[{"id":1},{"id":2}]`)
	docs, err = decodeDocumentSyncPayload(arrayPayload)
	if err != nil {
		t.Fatalf("array decode error: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("expected 2 docs, got %d", len(docs))
	}

	invalid := []byte(`123`)
	if _, err := decodeDocumentSyncPayload(invalid); err == nil {
		t.Fatalf("expected error for invalid payload")
	}
}

func TestExtractDocumentKey(t *testing.T) {
	doc := map[string]any{"id": "123"}
	key, err := extractDocumentKey(doc, "id", "string")
	if err != nil {
		t.Fatalf("extractDocumentKey returned error: %v", err)
	}
	if key != "123" {
		t.Fatalf("expected 123, got %s", key)
	}

	doc = map[string]any{"userid": json.Number("42")}
	key, err = extractDocumentKey(doc, "UserID", "number")
	if err != nil {
		t.Fatalf("case-insensitive key failed: %v", err)
	}
	if key != "42" {
		t.Fatalf("expected 42, got %s", key)
	}

	doc = map[string]any{}
	if _, err := extractDocumentKey(doc, "id", "string"); err == nil {
		t.Fatalf("expected error for missing key")
	}
}

func TestStringifyKey(t *testing.T) {
	cases := []struct {
		name    string
		value   any
		pkType  string
		expect  string
		wantErr bool
	}{
		{
			name:   "string",
			value:  " user ",
			pkType: "string",
			expect: "user",
		},
		{
			name:   "float number",
			value:  float64(5),
			pkType: "number",
			expect: "5",
		},
		{
			name:   "json number",
			value:  json.Number("12"),
			pkType: "number",
			expect: "12",
		},
		{
			name:    "empty string",
			value:   " ",
			pkType:  "string",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			key, err := stringifyKey(tc.value, tc.pkType)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if key != tc.expect {
				t.Fatalf("expected %q, got %q", tc.expect, key)
			}
		})
	}
}

func TestPrepareDocumentSyncPayload(t *testing.T) {
	source := map[string]any{
		"id":   "1",
		"name": "Alice",
		"key":  "should-ignore",
	}
	cleaned := prepareDocumentSyncPayload(source, "id", false)
	if _, ok := cleaned["id"]; ok {
		t.Fatalf("id should be removed when keepPrimary=false")
	}
	if _, ok := cleaned["key"]; ok {
		t.Fatalf("key should be removed from payload")
	}
	if !reflect.DeepEqual(cleaned["name"], "Alice") {
		t.Fatalf("expected name preserved")
	}

	cleanedKeep := prepareDocumentSyncPayload(source, "id", true)
	if cleanedKeep["id"] != "1" {
		t.Fatalf("expected id kept when keepPrimary=true")
	}
}

func TestPrepareDocumentCreatePayload(t *testing.T) {
	source := map[string]any{
		"ID":   "42",
		"name": "Bob",
		"key":  "ignored",
	}
	cleaned := prepareDocumentCreatePayload(source, "id")
	if cleaned["ID"] != "42" && cleaned["id"] != "42" {
		t.Fatalf("expected primary key retained in create payload")
	}
	if _, ok := cleaned["key"]; ok {
		t.Fatalf("key metadata should be stripped from create payload")
	}
	if cleaned["name"] != "Bob" {
		t.Fatalf("expected non-reserved field preserved in create payload")
	}

	lowerSource := map[string]any{"userid": "abc"}
	cleaned = prepareDocumentCreatePayload(lowerSource, "UserID")
	if cleaned["UserID"] != "abc" {
		t.Fatalf("expected case-insensitive primary key added back")
	}
}
