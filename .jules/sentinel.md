## 2024-05-14 - Fix API Key leakage in CLI config output
**Vulnerability:** The CLI `config show` and `config list --raw` commands leaked sensitive tenant API keys by directly serializing maps to YAML and JSON, relying on shallow copies.
**Learning:** When serializing configurations containing nested maps with secrets in Go (e.g., using `yaml.Marshal` or `json.Marshal`), deep cloning must be used rather than shallow pointer dereferencing to prevent credential leakage in serialized outputs.
**Prevention:** Implement deep copy methods (e.g. `MaskedClone`) for objects containing secrets within nested structs or maps, manually allocating new maps and copying elements while applying masks.
