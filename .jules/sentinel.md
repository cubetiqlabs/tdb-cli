## 2024-06-18 - Nested Secrets Serialization Leak
**Vulnerability:** Deeply nested secrets (API keys) in CLI configuration were leaked when serializing map values to stdout using `printJSON` and `yaml.Marshal`.
**Learning:** Shallow copying a top-level struct with nested maps does not clone the maps. Modifying values in the copied map modifies the original data, and failing to mask them results in credential leakage during serialization.
**Prevention:** Implement deep cloning patterns (e.g., `MaskedClone()`) for structures containing sensitive data before display, carefully iterating and copying all nested reference types.
