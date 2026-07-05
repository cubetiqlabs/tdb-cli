## 2024-10-24 - Prevent credential leakage from config outputs
**Vulnerability:** The CLI `tdb config show` and `tdb config list --raw` commands leak API keys and admin secrets due to shallow copying (pointer dereference) of nested maps in Go structures when serializing with json/yaml.
**Learning:** Shallow copying `display := *env.Config` does not clone nested maps/pointers in Go. Thus, modifying the copy (or leaving nested maps unmasked) affects the original or leaks unmasked secrets via nested references.
**Prevention:** Always implement and use a `MaskedClone()` method that deeply copies and masks all sensitive fields (including those in nested maps) prior to serialization.
