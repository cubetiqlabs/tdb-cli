## 2024-07-20 - [Fix configuration secret leakage in YAML output]
**Vulnerability:** The `tdb config show` command leaked API keys and modifying nested maps in shallow copies of the configuration struct before serialization mutated the original in-memory structs.
**Learning:** When serializing configurations containing nested maps with secrets in Go (e.g., using yaml.Marshal or json.Marshal), ensure deep cloning is used rather than shallow pointer dereferencing to prevent credential leakage in serialized outputs.
**Prevention:** Use a `MaskedClone` method that performs a shallow copy first to preserve unmapped non-secret fields, before manually reallocating and deeply copying nested maps or pointers while applying masking logic to secrets.
