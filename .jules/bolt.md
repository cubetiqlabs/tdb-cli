## 2024-06-22 - Optimize JSON request encoding
**Learning:** The codebase uses a custom `jsonRaw` type (underlying `[]byte`) for pre-encoded JSON payloads. When this is passed to `json.NewEncoder`, it causes reflection and double-encoding overhead.
**Action:** Type-assert for `jsonRaw` and pass it directly using `bytes.NewReader` to avoid reflection and double-encoding overhead.
