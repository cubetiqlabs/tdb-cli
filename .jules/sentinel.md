## 2025-02-15 - Masking API Keys in Config Display
**Vulnerability:** The `tdb config show` command masks the `admin_secret` but does not mask the actual `key` values in the `tenants` configuration, potentially leaking sensitive API keys to the terminal or logs.
**Learning:** Config structures that contain nested secrets (like API keys within tenant configurations) must deep clone and mask all secrets before serialization, not just top-level secrets.
**Prevention:** Implement deep cloning with a `MaskedClone()` method on configuration structs to ensure all nested secrets are properly redacted before outputting configuration.
