## 2024-05-18 - Optimize coerceJSONValue with json.RawMessage
**Learning:** In Go, unmarshaling a raw JSON string into a generic `interface{}` to wrap it for re-marshaling is very slow and loses large number precision (due to `float64` conversion).
**Action:** Use `json.Valid()` alongside `json.RawMessage()` to validate and defer processing of raw JSON strings that only need to be re-marshaled. This avoids intermediate map/slice allocations and preserves number precision.
