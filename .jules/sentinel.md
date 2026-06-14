## 2024-06-14 - Prevent API Key Leakage in CLI Output
**Vulnerability:** The `tdb config list --raw` and `tdb config show` commands were either shallow-copying or fully exposing API keys and tenant secrets in JSON output.
**Learning:** Shallow copies of structs containing maps (`Tenants map[string]TenantConfig`) will still share references to the underlying maps, leading to secrets remaining exposed if only the outer struct is passed.
**Prevention:** Always implement and use a deep-copy mechanism (like `MaskedCopy`) for configuration structs before marshalling them to output formats.
