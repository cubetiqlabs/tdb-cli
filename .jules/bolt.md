## 2024-10-19 - Pre-calculating slice capacity
**Learning:** In Go, dynamically appending to a slice (`rows = append(rows, ...)`) without a pre-calculated capacity inside heavily nested loops can lead to repeated slice reallocations and memory copying.
**Action:** When iterating over maps and slices where the total count is determinable, iterate first to pre-calculate the exact capacity, then instantiate the slice with `make([]T, 0, capacity)` to eliminate reallocations.
