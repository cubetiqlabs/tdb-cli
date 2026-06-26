## 2024-05-24 - [Avoid Double-Encoding Overhead in HTTP Requests]
**Learning:** The codebase was performing double encoding of `jsonRaw` objects by calling `json.NewEncoder(buf).Encode(payload)`, which took a byte slice of already-encoded JSON, re-encoded it as a JSON string, and placed it inside a buffer.
**Action:** Implemented a type assertion on `payload` to check for `interface{ Bytes() []byte; isJSONRaw() }`. For `jsonRaw`, bypassing `json.NewEncoder` and using `bytes.NewReader(raw.Bytes())` directly removes double encoding and string allocation overhead on every request.
