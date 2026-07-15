## 2024-07-15 - Hoist Regex Compilation
**Learning:** In Go, `regexp.MustCompile` is expensive and should not be placed inside functions that are called repeatedly.
**Action:** Hoist `regexp.MustCompile` to global package scope variables so they are compiled only once at program startup.
