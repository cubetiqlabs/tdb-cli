## 2024-08-06 - Credential Leak via Shallow Copy in Config Serialization
**Vulnerability:** `tdb config show` exposed API Keys because a shallow copy (`*env.Config`) was serialized; only `AdminSecret` was masked, and nested map items (API keys) were copied by reference and unmasked.
**Learning:** Shallow dereferencing (`*cfg`) does not deeply copy maps. If secrets reside inside nested collections, they leak when serialized (e.g. YAML) if not explicitly reallocated and masked.
**Prevention:** Always implement a dedicated deep clone method (e.g., `MaskedClone()`) that shallow copies struct fields first, then deeply iterates, reallocates, and masks secrets within nested maps.
