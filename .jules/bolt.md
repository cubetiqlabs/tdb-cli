## 2024-05-24 - Hoist regexp.MustCompile in Go
**Learning:** Repeatedly calling `regexp.MustCompile` inside a function re-compiles the regex every time the function is called, which is computationally expensive and degrades performance.
**Action:** Always hoist `regexp.MustCompile` to package-level global variables so the regular expression is compiled only once at program startup.
