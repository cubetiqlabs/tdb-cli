## 2024-05-20 - Hoist Regular Expressions
**Learning:** Calling `regexp.MustCompile` inside functions causes repeated expensive regex compilations that act as a performance bottleneck.
**Action:** Always hoist `regexp.MustCompile` to global package scope variables so they are compiled only once at program startup.
