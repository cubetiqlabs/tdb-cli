## 2024-05-24 - Prevent Credential Leakage in Raw Output
**Vulnerability:** Raw JSON output of configuration commands exposed plaintext API keys stored in maps.
**Learning:** When serializing configurations containing nested maps with secrets in Go, direct serialization leaks credentials. Deep cloning is required.
**Prevention:** Ensure deep cloning (such as a MaskedClone() method) is used rather than shallow pointer dereferencing to manually reallocate and deeply copy nested maps while masking secrets before JSON serialization.
