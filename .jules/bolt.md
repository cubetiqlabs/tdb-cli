## 2024-07-02 - Hoist regexp.MustCompile to package scope
**Learning:** Calling `regexp.MustCompile` inside a function recompiles the regular expression on every execution, causing unnecessary CPU overhead and allocation in frequently called functions like `extractSQLParams`.
**Action:** Always hoist `regexp.MustCompile` calls out of functions and into package-level global variables so they are compiled exactly once at program startup.
