## 2025-02-18 - [CRITICAL] Prevent Config Credential Leakage via Deep Masking
**Vulnerability:** The CLI commands `config show` and `config list --raw` printed the raw configuration struct, leaking API keys stored in nested maps (`Tenants` -> `Keys`).
**Learning:** Struct shallow-copying (e.g. `display := *env.Config`) for presentation modification is insufficient when sensitive data is nested within maps or pointers, as modifying the shallow copy's top-level fields ignores the nested references.
**Prevention:** Always implement and use a deep cloning strategy (like a recursive `MaskedClone` method) that correctly handles and masks all nested sensitive structures before serializing configuration states to stdout.
