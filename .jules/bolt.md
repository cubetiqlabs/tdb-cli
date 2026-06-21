## 2025-05-18 - Optimize JSON serialization for jsonRaw type
**Learning:** Found a performance bottleneck where custom pre-encoded JSON slices (e.g., `jsonRaw` alias for `[]byte`) were passed to `json.NewEncoder(buf).Encode(payload)`. This caused unnecessary reflection overhead and potentially incorrect double-encoding.
**Action:** When accepting `interface{}` payloads for network requests, check if the payload is already `[]byte` or a custom alias like `jsonRaw`. Use type assertion to pass it directly via `bytes.NewReader()` instead of `json.NewEncoder()`.
