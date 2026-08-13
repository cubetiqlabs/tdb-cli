## 2024-08-13 - Prevent credential leakage in serialized configurations
**Vulnerability:** Sensitive API keys inside nested maps in the configuration were leaked when printing the config to the console because only the top-level AdminSecret was masked and shallow copying was used.
**Learning:** Shallow copying structs with nested maps (e.g. `clone := *config`) doesn't create copies of the maps, causing the underlying references to remain. Masking nested maps requires explicit deep cloning to avoid modifying the original data or leaking secrets.
**Prevention:** Implement a deep cloning method like `MaskedClone()` to safely mask secrets in nested structures before serialization, rather than relying on shallow copies.
