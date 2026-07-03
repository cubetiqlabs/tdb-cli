## 2024-07-03 - Shallow Copy Credential Leakage in CLI Config
**Vulnerability:** Nested API keys within `TenantConfig` maps were exposed in plaintext when `yaml.Marshal` serialized a shallow copy of the configuration in the `tdb config show` command.
**Learning:** Go's shallow copying of structs (e.g., `display := *cfg`) does not deep clone maps or pointers. Consequently, nested structures like tenant keys remained unmasked and susceptible to leakage in standard serialized output.
**Prevention:** Use deep cloning mechanisms, such as a custom `MaskedClone()` method, when preparing configurations with nested maps or secrets for display or serialization.
