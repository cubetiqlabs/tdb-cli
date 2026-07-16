## 2024-05-18 - Prevent Credential Leakage in Raw Config List
**Vulnerability:** The CLI prints the full, unmasked `envCtx.Config.Tenants` data using `printJSON(cmd, envCtx.Config.Tenants)` when the `tdb config list --raw` command is run. This exposes the unmasked `APIKeyEntry.Key` values in plain text.
**Learning:** Raw JSON output flags can easily bypass normal CLI presentation layers that might obfuscate or omit secrets, leading to credential leakage if deep copying/masking isn't applied before serialization.
**Prevention:** Implement a `MaskedClone` method on `Config` and use it to scrub sensitive fields out of the object before passing it to serialization handlers like `printJSON` or logging.
