## 2024-05-24 - Masking secrets during deep config serialization
**Vulnerability:** API Keys inside nested `TenantConfig` maps were being serialized into plain text when running `tdb config list --raw` and shallow pointers could lead to credential leakage.
**Learning:** Shallow copies of structs with nested maps (`*env.Config`) do not copy the maps; modifying them or passing them to `json.Marshal` exposes the raw secrets.
**Prevention:** Implement and use a deep cloning method like `MaskedClone()` that properly reallocates and masks all secrets inside nested data structures before passing them to display or serialization outputs.
