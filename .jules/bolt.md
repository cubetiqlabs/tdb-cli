## 2024-05-18 - Optimize JSON Request Encoding
**Learning:** The codebase has a custom `jsonRaw` type (underlying `[]byte`) for pre-encoded JSON payloads. Previously, these were being passed to `json.NewEncoder`, which caused reflection and double-encoding overhead.
**Action:** When a request payload is of type `jsonRaw` or `[]byte`, directly type-assert it and pass it to `bytes.NewReader` instead of using `json.NewEncoder`.
