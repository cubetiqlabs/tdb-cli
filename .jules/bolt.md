## 2026-06-15 - Optimize JSON Requests
**Learning:** The codebase uses a custom `jsonRaw` type (underlying `[]byte`) to avoid parsing and encoding JSON payloads when they are already bytes. However, `json.NewEncoder(buf).Encode(payload)` was still being used, leading to double encoding.
**Action:** Implemented a fast path in `newJSONRequest` by type asserting `jsonRaw` or `[]byte`, wrapping it directly in `bytes.NewReader` and bypassing the encoding phase altogether, yielding performance improvements.
