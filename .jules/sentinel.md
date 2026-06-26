## 2024-06-26 - Masked Deep Copying for Credential Serialization
**Vulnerability:** API keys were being leaked when printing nested configuration to JSON via a raw tenant dump command, and `AdminSecret` was leaked during shallow copies.
**Learning:** When displaying configurations with nested maps containing secrets, a shallow copy of a struct is insufficient if nested structs (like tenant configs containing API keys) are directly manipulated or dumped. Modifying a shallow copy's maps will mutate the original state, and directly serializing raw fields will expose them.
**Prevention:** Always implement a method that performs a deep clone of the entire configuration hierarchy, proactively masking all secret fields, before sending to an output command like YAML or JSON serialization.
