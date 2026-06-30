## 2026-06-30 - [Hoist regexp compilations]
**Learning:** In Go, calling `regexp.MustCompile` inside a frequently executed function causes repeated expensive compilations.
**Action:** Always hoist `regexp.MustCompile` out of functions and into global package scope variables so they are compiled only once at program startup.
