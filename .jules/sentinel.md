## 2024-07-04 - Credential Leakage in CLI Config Serialization
**Vulnerability:** The CLI `tdb config show` command leaks plaintext API keys because it only masks the `AdminSecret` and uses a shallow copy, exposing the nested `APIKeyEntry.Key` values within the `Tenants` map.
**Learning:** When serializing configurations containing nested maps with secrets in Go (e.g., using `yaml.Marshal`), shallow copying is insufficient. Deep cloning must be used to ensure all nested secrets are masked.
**Prevention:** Always implement a `MaskedClone()` method for configuration structs that perform deep cloning and masks all sensitive fields (both root and nested) before displaying them to users.
