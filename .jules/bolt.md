## 2026-06-25 - [Double JSON Encoding Avoidance]
**Learning:** Found a performance bottleneck where raw JSON payloads were being double-encoded (and thus wrapped in strings or incurring unnecessary serialization overhead) when passed as `interface{}` to `json.NewEncoder`.
**Action:** Used an interface type assertion with marker methods (`Bytes()` and `isJSONRaw()`) to bypass the reflection-based `json.Encoder` and use a zero-allocation/lightweight `bytes.NewReader(raw.Bytes())` for already encoded payloads.
