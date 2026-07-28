## 2024-05-24 - API Key Leakage in CLI Config Display
**Vulnerability:** A shallow copy (`*env.Config`) of the config struct was used when displaying config via `tdb config show`. Because the nested `Tenants` map was not deep-copied, its contents (including plaintext API keys) were leaked in the CLI output.
**Learning:** In Go, shallow copying a struct only copies the map pointers. Any nested maps, slices, or pointers must be explicitly deeply copied before serializing or modifying the copied structure, to prevent credential leakage or unintended mutations.
**Prevention:** Always implement a custom deep copy method (like `MaskedClone()`) when duplicating configurations with nested maps containing secrets, and ensure all secret fields are masked during the copy.
