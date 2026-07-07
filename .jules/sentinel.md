## 2024-05-24 - Credential leakage through shallow copies
**Vulnerability:** API keys within a nested map in the configuration struct were leaked when the config was serialized and printed, because a shallow copy (`*env.Config`) was used before serialization.
**Learning:** In Go, shallow copying a struct that contains nested maps or pointers does not copy the underlying data. Modifying or serializing the shallow copy will still access the original nested data, including any secrets.
**Prevention:** Always implement a deep copy method (e.g., `MaskedClone()`) that shallow copies the struct first, and then manually reallocates and deep copies all nested maps/pointers while masking any sensitive fields.
