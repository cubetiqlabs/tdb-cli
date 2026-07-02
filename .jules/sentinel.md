## 2024-05-24 - Serialization of Secrets
**Vulnerability:** Shallow copying configurations containing nested maps with secrets before serializing them (e.g. using `yaml.Marshal` or `json.Marshal`) leads to credential leakage because nested pointers/maps are shared.
**Learning:** In Go, struct assignment `display := *env.Config` does not deep copy nested maps (`env.Config.Tenants`), meaning modifications to `display` or accidental printing of `display.Tenants` will expose actual secret data if the inner values aren't masked, or if we modify inner values it affects the real config.
**Prevention:** Implement a `MaskedClone()` method that performs a deep clone of the configuration, ensuring all nested maps and structures are copied and secrets are properly masked within the copy.
