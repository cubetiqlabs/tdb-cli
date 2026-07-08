## 2024-07-08 - Fix Credential Leakage in CLI Configuration Serialization
**Vulnerability:** The CLI commands `tdb config show` and `tdb config list --raw` serialize and display nested API keys in plaintext because only `AdminSecret` was masked and shallow cloning was used.
**Learning:** Serializing nested maps or structs containing secrets with `yaml.Marshal` or `json.Marshal` requires a deep copy where each nested secret is properly masked. Shallow copying leaves nested maps pointing to the original data, leaking credentials.
**Prevention:** Always implement and use a deep cloning method like `MaskedClone()` for configuration structures before displaying them to users.
