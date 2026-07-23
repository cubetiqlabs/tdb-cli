## 2024-05-24 - Credential Leakage in Serialized Outputs
**Vulnerability:** The CLI `tdb config show` and `tdb config list --raw` commands leak unmasked API keys because secrets in nested structures are not masked during JSON/YAML serialization.
**Learning:** When serializing configurations containing nested maps with secrets in Go, shallow pointer dereferencing or only masking top-level secrets leaves nested secrets exposed.
**Prevention:** Ensure a deep copy (e.g., a `MaskedClone()` method) is used to manually allocate and deeply copy nested structures, masking all secrets before serialization to prevent credential leakage.
