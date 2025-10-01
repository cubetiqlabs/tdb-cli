package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTenantClientReportQuery(t *testing.T) {
	t.Helper()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/api/query" {
			t.Errorf("expected path /api/query, got %s", r.URL.Path)
		}
		if got := r.Header.Get("X-API-Key"); got != "secret" {
			t.Errorf("missing api key header, got %q", got)
		}
		if got := r.Header.Get("X-App-ID"); got != "app123" {
			t.Errorf("missing app id header, got %q", got)
		}
		query := r.URL.Query()
		if got := query.Get("limit"); got != "5" {
			t.Errorf("expected limit query 5, got %q", got)
		}
		if got := query.Get("offset"); got != "3" {
			t.Errorf("expected offset query 3, got %q", got)
		}
		if got := query.Get("cursor"); got != "cursor123" {
			t.Errorf("expected cursor query cursor123, got %q", got)
		}
		if got := query.Get("select"); got != "category,total" {
			t.Errorf("expected select query category,total, got %q", got)
		}
		if got := query.Get("app_id"); got != "app123" {
			t.Errorf("expected app_id query app123, got %q", got)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("reading body: %v", err)
		}
		defer r.Body.Close()
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		if payload["collection"] != "orders" {
			t.Errorf("expected collection orders, got %v", payload["collection"])
		}
		if payload["cursor"] != "cursor123" {
			t.Errorf("expected cursor in body cursor123, got %v", payload["cursor"])
		}
		if payload["limit"] != float64(5) {
			t.Errorf("expected limit in body 5, got %v", payload["limit"])
		}
		if payload["offset"] != float64(3) {
			t.Errorf("expected offset in body 3, got %v", payload["offset"])
		}
		if _, ok := payload["aggregate"]; !ok {
			t.Errorf("expected aggregate clause in payload")
		}
		selectRaw := payload["select"]
		selectSlice, okCast := selectRaw.([]any)
		if !okCast || len(selectSlice) != 2 {
			t.Errorf("expected select array in payload, got %T with value %v", selectRaw, selectRaw)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"category":"books","total":42}],"pagination":{"limit":5,"offset":3,"total":1,"next_cursor":"next"}}`))
	}))
	defer ts.Close()

	client, err := NewTenantClient(ts.URL, "secret")
	if err != nil {
		t.Fatalf("NewTenantClient: %v", err)
	}

	resp, err := client.ReportQuery(context.Background(), ReportQueryParams{
		AppID:        "app123",
		Collection:   "orders",
		Limit:        5,
		Offset:       3,
		Cursor:       "cursor123",
		SelectFields: []string{"category", "total"},
		Body: map[string]any{
			"aggregate": []any{
				map[string]any{"field": "price", "operation": "sum", "alias": "total"},
			},
			"groupBy": []any{"category"},
		},
	})
	if err != nil {
		t.Fatalf("ReportQuery: %v", err)
	}

	if resp.Pagination.Limit != 5 {
		t.Errorf("expected pagination limit 5, got %d", resp.Pagination.Limit)
	}
	if resp.Pagination.Offset != 3 {
		t.Errorf("expected pagination offset 3, got %d", resp.Pagination.Offset)
	}
	if resp.Pagination.Total != 1 {
		t.Errorf("expected pagination total 1, got %d", resp.Pagination.Total)
	}
	if resp.Pagination.NextCursor != "next" {
		t.Errorf("expected next cursor 'next', got %q", resp.Pagination.NextCursor)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 data row, got %d", len(resp.Data))
	}
	row := resp.Data[0]
	if row["category"] != "books" {
		t.Errorf("expected category books, got %v", row["category"])
	}
	if row["total"] != float64(42) {
		t.Errorf("expected total 42, got %v", row["total"])
	}
}
