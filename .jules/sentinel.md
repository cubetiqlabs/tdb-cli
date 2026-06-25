## 2025-02-28 - [Credential Leakage in CLI Configuration Display]
**Vulnerability:** The `config show` command leaked nested API keys by only shallow copying the configuration object and masking the top-level `AdminSecret`. Nested maps within the config struct containing API keys were exposed in plaintext when serialized to YAML.
**Learning:** Shallow copies of structs with nested maps do not protect data within those maps. A deep clone is required when modifying a structure for display to avoid exposing sensitive data in fields containing maps or slices.
**Prevention:** Implement comprehensive deep cloning methods like `MaskedClone()` that recursively navigate and mask all sensitive strings, especially within deeply nested configurations or collections, before serialization and display.
