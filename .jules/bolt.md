## 2024-10-18 - String Builder Optimization
**Learning:** The `strings.Join` combined with intermediate string allocations for table rendering can be slow. Calculating length and using `Builder.Grow` avoids unnecessary intermediate memory allocations and slice operations.
**Action:** When building large strings (like table borders or lines), use a `strings.Builder` with `Grow` pre-calculated to reduce runtime execution time and heap allocations.
