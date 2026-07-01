## 2024-05-24 - Hoist `regexp.MustCompile` to prevent repeated allocations
**Learning:** `regexp.MustCompile` inside frequently called functions (like `extractSQLParams`) causes expensive repeated compilations and allocations during runtime, creating a performance bottleneck in Go applications.
**Action:** Always hoist `regexp.MustCompile` out of functions and into global package scope variables so they are compiled only once at program startup.
