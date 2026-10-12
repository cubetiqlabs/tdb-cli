## 2024-10-08 - API Key Exposure in CLI Config Output
**Vulnerability:** The raw configuration output commands (`tdb config show` and `tdb config list --raw`) were exposing raw API keys and the Admin Secret in plaintext without masking.
**Learning:** While the custom JSON serialization or YAML output handlers process the config objects, raw map printing and shallow copying allowed deep objects containing secrets to leak directly to `cmd.OutOrStdout()`.
**Prevention:** Always implement a dedicated `MaskedClone()` method for sensitive structures that performs deep copying of any nested maps and pointers containing credentials before marshalling for output.

## 2024-05-24 - Prevent Decompression Bomb (G110) with io.LimitReader
**Vulnerability:** Unrestricted `io.Copy` operations when extracting `tar.gz` and `zip` archives can lead to decompression bombs, crashing or filling the disk of the machine running the upgrade command.
**Learning:** `io.LimitReader` combined with `io.Copy` is a common fix, but relying solely on `io.LimitReader` silently truncates the file if it exceeds the limit without returning an error.
**Prevention:** Always check if the number of written bytes equals the limit (or read one extra byte) and return an explicit error. Do not rely only on `io.LimitReader` for bounded `io.Copy`.
