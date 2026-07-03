## 2024-07-03 - Hoist regexp.MustCompile out of function scope
**Learning:** Calling `regexp.MustCompile` inside a function causes the regular expression to be recompiled every time the function is invoked, which is an expensive operation and a performance bottleneck.
**Action:** Always hoist `regexp.MustCompile` out of functions and into global package scope variables so they are compiled only once at program startup.
