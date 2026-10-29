## 2024-10-08 - API Key Exposure in CLI Config Output
**Vulnerability:** The raw configuration output commands (`tdb config show` and `tdb config list --raw`) were exposing raw API keys and the Admin Secret in plaintext without masking.
**Learning:** While the custom JSON serialization or YAML output handlers process the config objects, raw map printing and shallow copying allowed deep objects containing secrets to leak directly to `cmd.OutOrStdout()`.
**Prevention:** Always implement a dedicated `MaskedClone()` method for sensitive structures that performs deep copying of any nested maps and pointers containing credentials before marshalling for output.

## 2026-10-11 - Prevent Decompression Bombs (Zip Bombs) in Upgrade Command
**Vulnerability:** The `io.Copy(out, tarReader)` and `io.Copy(out, rc)` operations during binary extraction in the `upgrade` command didn't limit the number of bytes read. This makes it susceptible to a DoS vulnerability (G110) via decompression bombs (like zip bombs).
**Learning:** During extraction of compressed binaries, an attacker could provide a highly compressed payload that expands to fill all available disk space or exhaust memory when extracted.
**Prevention:** Always use `io.LimitReader` when reading from compressed archives or untrusted sources to enforce a maximum read limit and prevent resource exhaustion.

## 2026-10-12 - Prevent Memory Exhaustion (DoS) in Unbounded Reads
**Vulnerability:** Unbounded calls to `io.ReadAll(cmd.InOrStdin())` were reading entire input streams into memory, making the CLI vulnerable to memory exhaustion (DoS) when fed excessively large input files or streams.
**Learning:** Using `io.ReadAll` on untrusted input sources without limits can allow attackers to crash the process by providing large payloads.
**Prevention:** Always wrap the reader with `io.LimitReader` and read up to `limit + 1` bytes. Explicitly check if the read length exceeds the limit and return an error to avoid silently truncating the input.

## 2026-10-13 - Correct io.LimitReader Bounds Checking
**Vulnerability:** When combining `io.LimitReader` and `io.Copy` to prevent DoS via decompression bombs, checking if `written == limit` incorrectly blocked valid files that exactly match the limit size.
**Learning:** Using `io.LimitReader(r, limit)` and failing on `written == limit` is a bug. `io.LimitReader` returns `EOF` after exactly `limit` bytes. To properly detect overflow, you must read up to `limit + 1` bytes and check if `written > limit`.
**Prevention:** Always configure `io.LimitReader` to `limit + 1` when paired with `io.Copy`, and check `written > limit` to detect explicit overflow rather than matching the valid limit bound.
