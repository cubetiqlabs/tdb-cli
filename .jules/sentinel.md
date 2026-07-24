## 2024-07-24 - Masking secrets in nested structs during serialization
**Vulnerability:** The CLI `config show` command leaked stored API keys in plain text because it only masked the top-level `AdminSecret` and performed a shallow copy before serialization, leaving nested map secrets (`APIKeyEntry.Key`) fully exposed.
**Learning:** Shallow copying a struct with nested maps or pointers does not deep copy the nested elements. Modifying or serializing them can leak sensitive data or unintentionally mutate shared state.
**Prevention:** Always implement and use a robust deep copy method (like `MaskedClone`) that first performs a shallow copy to preserve unmapped fields, then manually reallocates and deeply copies/masks any nested maps or pointers before serialization.
