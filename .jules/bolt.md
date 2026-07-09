## 2026-07-09 - Hoist regex compilation in extractSQLParams
**Learning:** Using `regexp.MustCompile` inside frequently called functions causes repeated expensive regex compilations in Go, leading to unnecessary CPU overhead.
**Action:** Always hoist `regexp.MustCompile` to global package scope variables so they are compiled only once at program startup.
