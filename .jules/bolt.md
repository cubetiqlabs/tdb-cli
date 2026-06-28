## 2024-05-01 - Avoid Regex Compilation in Loops/Hot Paths
**Learning:** `regexp.MustCompile` is an expensive operation. If it's called repeatedly within a function (such as `extractSQLParams` that might be called for every query parsed), it adds significant overhead.
**Action:** Always hoist `regexp.MustCompile` to global package scope so it is compiled only once at program startup.
