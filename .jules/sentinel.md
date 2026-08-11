## 2024-05-18 - Prevent Credential Leak in Raw Config Output
**Vulnerability:** Unmasked API secrets are exposed via the CLI when dumping raw configuration using `tdb config list --raw`.
**Learning:** Configurations containing nested maps with secrets must be deep cloned before serialization to prevent credential leakage without corrupting in-memory config.
**Prevention:** Implement deep copy cloning with masking for credentials when converting configuration files to raw display formats.
