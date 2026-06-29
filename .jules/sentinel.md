## 2024-05-18 - Prevent Secret Leakage in Config Serialization
**Vulnerability:** API keys and credentials could be exposed when serializing the configuration object (e.g. `tdb config show`, `tdb config list --raw`) because shallow copying failed to mask nested map structures.
**Learning:** When dealing with nested structs or maps containing sensitive data, shallow copies (e.g., `*config = *original`) do not duplicate nested references. Modifications to or serialization of the copy can expose secrets from the original data structure.
**Prevention:** Always implement and use a deep cloning method (e.g., `MaskedClone()`) that recursively copies and masks all sensitive fields, particularly within maps and slices, before passing configuration data to output serialization functions.
