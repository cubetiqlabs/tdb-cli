## 2024-08-01 - Prevent Credential Leakage in CLI Config Display
**Vulnerability:** The `tdb config show` command serialized the configuration and exposed raw tenant API keys because only the `AdminSecret` was masked and the `Tenants` map was a shallow copy.
**Learning:** When serializing configurations containing nested maps with secrets in Go, shallow pointer dereferencing allows credential leakage.
**Prevention:** Ensure a deep cloning method (like `MaskedClone`) is used to manually reallocate nested maps and correctly mask all secret fields before displaying configuration data.
