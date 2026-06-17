## 2025-06-17 - Fix credential leakage in config display
**Vulnerability:** API Keys nested within the Config.Tenants.Keys map were exposed in plaintext when using `tdb config show` because the `AdminSecret` was manually masked but the nested `Tenants` map was serialized via a shallow copy.
**Learning:** When displaying or serializing configurations containing nested maps with secrets, always perform a deep clone and mask all nested secrets rather than relying on shallow pointer dereferencing or selectively masking top-level fields only.
**Prevention:** Always implement a dedicated deep-copy masking method (like `MaskedClone`) for struct types holding multiple layered secrets, rather than manually altering exported fields on a copied value before serialization.
