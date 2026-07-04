## 2026-07-04 - Avoid repeated regex compilation
**Learning:** Repeatedly calling `regexp.MustCompile` inside a function in Go causes expensive compilation overhead on every invocation.
**Action:** Always hoist `regexp.MustCompile` to a global, package-level variable so it is evaluated only once at initialization.
