## 2026-07-17 - Hoist regular expression compilation
**Learning:** Calling `regexp.MustCompile` inside a function recompiles the regular expression every time the function is called, which is a performance bottleneck in Go.
**Action:** Hoist `regexp.MustCompile` to global package scope variables so they are compiled only once at program startup.
