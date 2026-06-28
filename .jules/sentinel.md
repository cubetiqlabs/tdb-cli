## 2024-06-28 - Credential leakage in configuration serialization
**Vulnerability:** The CLI `config list --raw` and `config show` commands leaked unmasked sensitive API keys to output because deep maps containing secrets were serialized directly without deep cloning and masking.
**Learning:** When serializing structs that contain nested maps or pointers to secrets, making a shallow pointer dereference or copying only top-level fields leaves nested references pointing to the original sensitive data, causing credential leaks when serialized.
**Prevention:** Ensure deep cloning strategies (like a `MaskedClone()` method) are used for any configurations containing nested secrets before outputting or serializing them.
