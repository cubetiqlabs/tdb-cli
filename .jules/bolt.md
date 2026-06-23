## 2024-05-24 - Pre-encoded JSON Overhead
**Learning:** Found that passing `jsonRaw` (underlying `[]byte`) into `newJSONRequest` triggers `json.NewEncoder(buf).Encode(payload)`, which causes reflection and double-encoding overhead for payloads that are already JSON bytes.
**Action:** Implemented a type assertion for `interface{ Bytes() []byte }` in `newJSONRequest` to pass the raw bytes directly using `bytes.NewReader` without re-encoding, and added `Bytes() []byte` to `jsonRaw`.
