## 2023-10-25 - Mask API Keys in Config Show
**Vulnerability:** API keys stored in the config file were exposed in plaintext when running `tdb config show`.
**Learning:** Only the `AdminSecret` was being masked when displaying the config. We need to deep clone and mask all sensitive fields, such as API keys within tenant configurations, before outputting. Deep cloning struct with maps requires making a new map to avoid changing the original config reference when masking.
**Prevention:** Always consider the full scope of sensitive data stored in a configuration object, especially in nested structures, and ensure all such fields are masked before serialization for display.
