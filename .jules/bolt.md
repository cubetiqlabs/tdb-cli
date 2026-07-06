## 2026-07-06 - Hoist regular expression compilation to global scope
**Learning:** Using `regexp.MustCompile` inside a function causes the regular expression to be recompiled every time the function is called, which introduces an unnecessary performance bottleneck.
**Action:** Hoist `regexp.MustCompile` out of functions and into global package scope variables so they are compiled only once at program startup.
