package cli

import "testing"

func TestParseAggregateSpecs(t *testing.T) {
	input := []string{"sum:price:total","count","avg:score","min:ts:first_seen","badop:value","avg::missing_field"}
	specs, warnings := parseAggregateSpecs(input)
	if len(specs) != 4 { t.Fatalf("expected 4 valid specs got %d", len(specs)) }
	if len(warnings) != 2 { t.Fatalf("expected 2 warnings got %d (%v)", len(warnings), warnings) }
}

func TestExpandAggregateSugar(t *testing.T) {
	specs := expandAggregateSugar(true, "user_id", []string{"price","amount"}, []string{"ts"}, []string{"ts"}, []string{"score"})
	if len(specs) != 1+1+2+1+1+1 { t.Fatalf("unexpected spec count %d", len(specs)) }
	foundDistinct := false
	for _, s := range specs { if s.Distinct { foundDistinct = true } }
	if !foundDistinct { t.Fatalf("expected a distinct aggregate") }
}

// Added tests for dedupeAggregateSpecs helper
func TestDedupeAggregateSpecs(t *testing.T) {
    specs := []aggregateSpecCLI{
        {Operation: "count"},
        {Operation: "count"}, // dup
        {Operation: "sum", Field: "price"},
        {Operation: "sum", Field: "price", Alias: "total_price"}, // dup with alias
        {Operation: "sum", Field: "amount"},
        {Operation: "count", Field: "user_id", Distinct: true},
        {Operation: "count", Field: "user_id", Distinct: true, Alias: "dupe"}, // dup distinct
    }
    deduped, warnings := dedupeAggregateSpecs(specs)
    if len(deduped) != 4 { // expected unique: count(*), sum(price), sum(amount), count(distinct user_id)
        t.Fatalf("expected 4 unique specs, got %d: %#v", len(deduped), deduped)
    }
    if len(warnings) != 3 { // 3 duplicates above
        t.Fatalf("expected 3 warnings, got %d: %#v", len(warnings), warnings)
    }
    // Ensure order preserved for first occurrences
    wantOrder := []aggregateSpecCLI{{Operation: "count"}, {Operation: "sum", Field: "price"}, {Operation: "sum", Field: "amount"}, {Operation: "count", Field: "user_id", Distinct: true}}
    for i, w := range wantOrder {
        if deduped[i].Operation != w.Operation || deduped[i].Field != w.Field || deduped[i].Distinct != w.Distinct {
            t.Fatalf("order mismatch at %d: got %#v want %#v", i, deduped[i], w)
        }
    }
}
