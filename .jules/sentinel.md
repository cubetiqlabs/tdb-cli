## 2024-10-08 - API Key Exposure in CLI Config Output
**Vulnerability:** The raw configuration output commands (`tdb config show` and `tdb config list --raw`) were exposing raw API keys and the Admin Secret in plaintext without masking.
**Learning:** While the custom JSON serialization or YAML output handlers process the config objects, raw map printing and shallow copying allowed deep objects containing secrets to leak directly to `cmd.OutOrStdout()`.
**Prevention:** Always implement a dedicated `MaskedClone()` method for sensitive structures that performs deep copying of any nested maps and pointers containing credentials before marshalling for output.

## 2026-10-11 - Prevent Decompression Bombs (Zip Bombs) in Upgrade Command
**Vulnerability:** The `io.Copy(out, tarReader)` and `io.Copy(out, rc)` operations during binary extraction in the `upgrade` command didn't limit the number of bytes read. This makes it susceptible to a DoS vulnerability (G110) via decompression bombs (like zip bombs).
**Learning:** During extraction of compressed binaries, an attacker could provide a highly compressed payload that expands to fill all available disk space or exhaust memory when extracted.
**Prevention:** Always use `io.LimitReader` when reading from compressed archives or untrusted sources to enforce a maximum read limit and prevent resource exhaustion.

## 2024-10-15 - Unbounded Read from Stdin Vulnerability
**Vulnerability:** Reading from `cmd.InOrStdin()` using `io.ReadAll` without a limit allowed a potential DoS vulnerability where malicious input could exhaust memory.
**Learning:** Untrusted streams, even stdin, can provide arbitrarily large data. `io.ReadAll` will try to read until EOF, potentially consuming all available memory.
**Prevention:** Always wrap untrusted streams (like stdin) with `io.LimitReader` (e.g., a 10MB limit) before passing them to `io.ReadAll` to prevent memory exhaustion and DoS.
