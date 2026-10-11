## 2024-10-08 - API Key Exposure in CLI Config Output
**Vulnerability:** The raw configuration output commands (`tdb config show` and `tdb config list --raw`) were exposing raw API keys and the Admin Secret in plaintext without masking.
**Learning:** While the custom JSON serialization or YAML output handlers process the config objects, raw map printing and shallow copying allowed deep objects containing secrets to leak directly to `cmd.OutOrStdout()`.
**Prevention:** Always implement a dedicated `MaskedClone()` method for sensitive structures that performs deep copying of any nested maps and pointers containing credentials before marshalling for output.

## 2026-10-11 - Prevent Decompression Bombs (Zip Bombs) in Upgrade Command
**Vulnerability:** The `io.Copy(out, tarReader)` and `io.Copy(out, rc)` operations during binary extraction in the `upgrade` command didn't limit the number of bytes read. This makes it susceptible to a DoS vulnerability (G110) via decompression bombs (like zip bombs).
**Learning:** During extraction of compressed binaries, an attacker could provide a highly compressed payload that expands to fill all available disk space or exhaust memory when extracted.
**Prevention:** Always use `io.LimitReader` when reading from compressed archives or untrusted sources to enforce a maximum read limit and prevent resource exhaustion.
