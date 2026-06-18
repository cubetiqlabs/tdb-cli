## 2025-03-01 - Avoid reflection on pre-encoded JSON payloads
**Learning:** The codebase uses a custom `jsonRaw` type (underlying `[]byte`) for pre-encoded JSON payloads. When passing this type to `json.NewEncoder().Encode()`, it incurs reflection overhead and double-encodes the bytes unless a fast-path is implemented.
**Action:** Always check if a payload is of type `jsonRaw` and pass it directly using `bytes.NewReader` rather than relying on `json.NewEncoder` to optimize performance.
