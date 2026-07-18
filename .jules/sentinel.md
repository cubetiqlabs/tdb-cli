## 2024-07-18 - Prevent Credential Leakage in Config Display
**Vulnerability:** The `tdb config show` command leaked tenant API keys by serializing the configuration directly after only masking the admin secret.
**Learning:** Serializing complex configuration structs with nested maps requires deep cloning to safely mask secrets, as shallow pointer copies or incomplete masking can still expose sensitive data.
**Prevention:** Always implement a comprehensive deep clone method (like `MaskedClone()`) that explicitly allocates and copies all nested maps/pointers while masking all secret fields before serialization.
