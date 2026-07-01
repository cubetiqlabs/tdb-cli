## 2024-05-23 - Credential Leakage in Configuration Serialization
**Vulnerability:** The CLI commands `tdb config show` and `tdb config list --raw` leaked raw API keys due to shallow struct copying before JSON/YAML serialization.
**Learning:** When serializing objects containing nested maps or slices, shallow pointer dereferencing (like `*config`) only copies the top-level references. Nested map values remain referenced to the original object, leaking unmasked secrets when modified or serialized.
**Prevention:** Ensure deep cloning strategies (such as a `MaskedClone()` method) are utilized to explicitly copy and sanitize all nested map or slice fields before serialization.
