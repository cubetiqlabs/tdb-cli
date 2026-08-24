## 2024-05-18 - Hoist Regex Compilation
**Learning:** In Go, calling `regexp.MustCompile` inside a function that is executed repeatedly causes unnecessary overhead due to recompiling the regular expression on every invocation.
**Action:** Extract `regexp.MustCompile` calls outside of functions (e.g., as global/package-level variables) so they are evaluated only once at startup.
