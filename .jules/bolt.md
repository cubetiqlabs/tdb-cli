
## 2024-11-20 - [Optimize table rendering with strings.Builder and capacity pre-calculation]
**Learning:** In Go, heavy string manipulation routines that loop to construct UI elements (like borders and rows for CLI tables) often incur unnecessary performance overhead if intermediate structures (like slices) are allocated to leverage `strings.Join`.
**Action:** Instead of accumulating strings into slices and joining them, use `strings.Builder` with a pre-calculated capacity (`.Grow()`) to minimize dynamic memory reallocation overhead. Also ensure that you compute the capacity using fast, lightweight metrics (e.g., standard byte length) rather than expensive formatting functions (like `runewidth.StringWidth`) inside the capacity loop.
