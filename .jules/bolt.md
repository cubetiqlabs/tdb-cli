## 2024-05-29 - Regex Compilation Hoisting
**Learning:** Calling `regexp.MustCompile` inside a function that may be executed repeatedly (like `extractSQLParams` for SQL queries) causes expensive regex compilation on every invocation, creating unnecessary CPU overhead and garbage collection pressure in Go.
**Action:** Always hoist `regexp.MustCompile` calls out of functions and into global package scope variables so they are compiled only once at program startup.
