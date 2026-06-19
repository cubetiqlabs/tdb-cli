
## 2026-06-19 - [Optimize JSON payload encoding for jsonRaw type]
**Learning:** The codebase defines a custom `jsonRaw` type (`[]byte`) to represent pre-encoded JSON payloads. When generating HTTP requests, passing this type to `json.NewEncoder(buf).Encode(payload)` causes unnecessary reflection and double-encoding overhead.
**Action:** Always check for `jsonRaw` specifically with a type assertion (`raw, ok := payload.(jsonRaw)`) and pass it directly using `bytes.NewReader(raw)` to skip the JSON encoding overhead.
