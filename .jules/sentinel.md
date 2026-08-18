## 2026-08-18 - Prevented Credential Leak in Config Serialization
**Vulnerability:** `tdb config show` and `tdb config list --raw` were leaking raw API keys from nested config maps.
**Learning:** Using a shallow struct copy (`display := *env.Config`) in Go copies maps by reference. Attempting to mask inner map keys without deep cloning or just omitting the masking logic leaves deep credentials fully exposed during YAML/JSON serialization.
**Prevention:** When serializing configurations containing nested maps with secrets, always implement and use a deep cloning method (e.g., `MaskedClone()`) to ensure deeply nested secrets are masked or removed before passing the object to `yaml.Marshal` or `json.Marshal`.
