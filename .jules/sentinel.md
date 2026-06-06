## 2025-02-14 - Masked API Keys in CLI output
**Vulnerability:** API Keys (`X-API-Key`) stored in `APIKeyEntry` structs were printed in plain text during `tdb config show` execution. This caused a potential security risk of secret leakage to logs or unauthorized viewers of the terminal.
**Learning:** Only the `AdminSecret` was being masked before output, but tenant keys were missed. It is important to deeply sanitize all nested data structures containing secrets before displaying them in plain text or writing them to standard output.
**Prevention:** Implement recursive or dedicated secret masking techniques for any data structure printed directly to a terminal, ensuring that API keys, tokens, and passwords are never visible in plaintext commands like `config show`.
