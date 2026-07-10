## 2024-05-24 - Hoist Regex Compilation
**Learning:** Compiling regular expressions using `regexp.MustCompile` inside functions called repeatedly is a performance anti-pattern in Go.
**Action:** Hoist `regexp.MustCompile` to package-level variables so they are compiled only once at program startup.
