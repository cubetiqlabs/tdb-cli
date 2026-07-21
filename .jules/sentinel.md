## 2024-07-21 - [Prevent API Key Leakage via Shallow Cloning in Config Show]
**Vulnerability:** The `tdb config show` command leaked sensitive tenant API keys because it only masked the `AdminSecret`, displaying API keys in plain text.
**Learning:** Using shallow copying for Go structures (e.g., `display := *c`) copies references to nested maps instead of cloning the map's contents. Consequently, iterating and modifying values in these maps was overlooked, resulting in credential leakage.
**Prevention:** Ensure deep cloning strategies (such as a `MaskedClone()` method) are utilized when dealing with nested structures containing secrets before serialization.
