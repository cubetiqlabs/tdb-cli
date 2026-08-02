## 2025-02-28 - Credential leakage due to shallow copy of nested maps
**Vulnerability:** The CLI `config show` and `config list --raw` commands leaked plaintext tenant API keys.
**Learning:** In Go, dereferencing a struct pointer (`*c`) only performs a shallow copy. Nested structures like maps and slices continue to reference the original data. When modifying the copy for masking secrets, the nested structures remain unmasked.
**Prevention:** Implement deep cloning (e.g., a `MaskedClone()` method) for structures with nested maps or pointers before serialization to safely mask secrets without affecting the original in-memory instance.
