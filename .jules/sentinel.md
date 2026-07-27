## 2024-07-27 - Configuration Secret Leakage
**Vulnerability:** The CLI `config show` command was leaking tenant API keys because it performed a shallow copy of the config and only masked the `AdminSecret`, leaving deeply nested API keys exposed when marshaling to YAML.
**Learning:** In Go, a shallow copy of a struct (e.g., `clone := *config`) does not duplicate nested maps or pointers. Consequently, any secrets stored in nested maps remain unmasked unless explicitly deep-copied and sanitized.
**Prevention:** Always implement a dedicated deep cloning method (e.g., `MaskedClone()`) for configuration objects that manually reallocates and copies nested maps while masking all contained secrets before serialization for display.
