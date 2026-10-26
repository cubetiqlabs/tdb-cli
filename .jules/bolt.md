## 2024-03-20 - Dedupe String Concatenation Optimization
**Learning:** Found a loop in `dedupeAggregateSpecs` doing repeated allocations via `s.Operation + "|" + s.Field + "|" + strconv.FormatBool(s.Distinct)` to construct a string map key. In Go, struct values can be map keys without allocating a new heap string every iteration.
**Action:** Replace map string keys constructed dynamically in a loop with composite struct map keys (`make(map[specKey]struct{})`) to eliminate loop body string allocations, especially helpful when functions are executed heavily.
