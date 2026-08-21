## 2024-05-24 - API Key Leak in Config Output
**Vulnerability:** The CLI `config list --raw` command leaked full plaintext API keys by directly marshalling the configuration structs to JSON without masking them.
**Learning:** Raw JSON struct serialization features often bypass specific presentation-layer logic (like masking secrets for UI tables). It's crucial to ensure secrets are sanitized or masked in the structures being serialized before passing them to JSON formatters.
**Prevention:** Always deep copy and sanitize configuration structs containing credentials before returning them as raw JSON output.
