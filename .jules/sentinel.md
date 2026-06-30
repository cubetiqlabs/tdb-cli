## 2024-05-15 - Credential leakage in serialized config output
**Vulnerability:** The CLI configuration output (via YAML or JSON serialization) exposed unmasked API keys from nested maps because only the top-level AdminSecret was masked or a shallow copy was used.
**Learning:** When serializing configurations containing nested maps with secrets in Go, ensure deep cloning (such as a `MaskedClone()` method) is used rather than shallow pointer dereferencing to prevent credential leakage in serialized outputs.
**Prevention:** Always implement and use deep cloning (`MaskedClone()`) for structures containing sensitive data before serialization, especially when nested maps or pointers are involved.
