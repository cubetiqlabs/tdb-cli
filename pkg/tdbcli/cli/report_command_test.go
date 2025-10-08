package cli

import (
	"encoding/json"
	"testing"
)

func TestBuildReportBodyAggregatesAndGroupBy(t *testing.T) {
	base := map[string]any{"collection":"orders"}
	body := buildReportBody(base, []string{"category","region"}, []string{"sum:price:total_sales","count::row_count","count!distinct:customer_id:unique_customers"})
	gb, ok := body["groupBy"].([]string)
	if !ok || len(gb) != 2 || gb[0] != "category" || gb[1] != "region" {
		b,_ := json.Marshal(body)
		t.Fatalf("unexpected groupBy: %s", string(b))
	}
	agg, ok := body["aggregate"].([]map[string]any)
	if !ok || len(agg) != 3 {
		b,_ := json.Marshal(body)
		t.Fatalf("unexpected aggregate slice: %s", string(b))
	}
	// ensure operations and aliases present
	wantOps := map[string]struct{}{ "sum":{}, "count":{} }
	for _, spec := range agg {
		op, _ := spec["operation"].(string)
		if _, ok := wantOps[op]; !ok { t.Fatalf("unexpected op %v", op) }
		if op == "sum" && spec["alias"] != "total_sales" { t.Fatalf("missing alias for sum: %#v", spec) }
	}
}

func TestDecideStreamingExport(t *testing.T) {
	if ok, _ := decideStreamingExport(true, nil, false, "jsonl"); !ok { t.Fatalf("expected streaming true") }
	if ok, reason := decideStreamingExport(true, []string{"f=1"}, false, "jsonl"); ok || reason == "" { t.Fatalf("expected filter rejection") }
	if ok, _ := decideStreamingExport(true, nil, true, "jsonl"); ok { t.Fatalf("expected includeDeleted rejection") }
	if ok, _ := decideStreamingExport(true, nil, false, "json"); ok { t.Fatalf("expected json format rejection") }
}
