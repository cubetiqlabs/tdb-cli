## 2026-07-16 - Hoisted regexp compilation to package scope
**Learning:** Discovered a repeated expensive compilation of a regular expression (`regexp.MustCompile`) inside `extractSQLParams`, which is called during query parameter extraction.
**Action:** Hoisted the compilation to a global package-scoped variable so it is compiled only once at program startup, avoiding repeated overhead.
