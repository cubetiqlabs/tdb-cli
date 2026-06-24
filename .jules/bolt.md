## 2026-06-24 - Avoid double-encoding for pre-encoded JSON payloads
**Learning:** The codebase uses a custom jsonRaw type for pre-encoded JSON payloads. When passed to the newJSONRequest HTTP request builder, json.NewEncoder creates unnecessary overhead. We can implement a Bytes() []byte method on jsonRaw and use a type assertion for interface{ Bytes() []byte } to stream it directly to the HTTP request without intermediate buffer allocation or duplicate encoding.
**Action:** Implemented Bytes() []byte on jsonRaw and added the interface type assertion to newJSONRequest.
