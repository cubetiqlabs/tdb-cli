## 2025-01-20 - Hoist regexp.MustCompile in extractSQLParams
**Learning:** Repeatedly calling `regexp.MustCompile` inside a function re-compiles the regex on every execution, which is an expensive operation and causes unnecessary CPU and memory overhead, especially for functions called frequently.
**Action:** Always hoist `regexp.MustCompile` to global package scope variables in Go so they are compiled only once at program startup.
