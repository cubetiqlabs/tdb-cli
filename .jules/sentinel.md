## 2024-03-22 - Fix Config API Key Leakage
**Vulnerability:** The CLI `tdb config show` command leaked API keys in plaintext because it performed a shallow copy of the configuration structure and only masked the `AdminSecret` before serialization.
**Learning:** When serializing configurations containing nested maps with secrets in Go, shallow pointer dereferencing (`clone := *c`) drops unmapped non-secret fields or preserves pointers, leading to secret leakage in nested structures. Deep cloning is required.
**Prevention:** Ensure deep cloning (such as a `MaskedClone()` method) is used rather than shallow pointer dereferencing to prevent credential leakage in serialized outputs. Manually reallocate and deeply copy nested maps or pointers.
