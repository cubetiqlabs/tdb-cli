## 2024-05-24 - API Key Leakage in CLI JSON Output
**Vulnerability:** The CLI `tdb config list --raw` command printed the full, unmasked API key entries by serializing the raw tenant configurations using `printJSON`.
**Learning:** Shallow copies of structs for masking (e.g. `display := *env.Config`) do not correctly obscure deeply nested maps such as `map[string]APIKeyEntry` containing API keys, leading to credential leakage in structured outputs like YAML or JSON.
**Prevention:** Implement deep cloning with masking logic (e.g., a `MaskedClone()` method) when passing complex configuration structures with sensitive secrets to serialization commands, ensuring no nested fields leak values.
