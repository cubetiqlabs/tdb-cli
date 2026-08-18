## 2024-03-24 - Hoist regexp.MustCompile
**Learning:** Found a performance bottleneck where `regexp.MustCompile` was inside `extractSQLParams`, meaning it compiled on every call. In Go, compiling regular expressions is expensive.
**Action:** Move `regexp.MustCompile` to global scope so it's compiled once on init.
