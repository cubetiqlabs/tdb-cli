## 2024-05-24 - Masking secrets properly
**Vulnerability:** Credential leakage in serialized configurations when using shallow copying.
**Learning:** Shallow dereferencing (e.g. `display := *env.Config`) is insufficient for deep structures containing secrets (like nested maps of API keys). `yaml.Marshal` will encode the actual API keys since they reside in referenced maps.
**Prevention:** Implement a deep-copying `MaskedClone()` method on configuration structs to ensure all nested secrets (API keys, etc.) are properly masked before serialization.
