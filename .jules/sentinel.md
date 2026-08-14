## 2024-08-14 - Fix API Key Leak in Config Output
**Vulnerability:** The `tdb config show` command leaked plaintext API keys when printing the configuration.
**Learning:** When displaying or serializing configurations that contain nested maps or structs with secrets, shallow copying or partial masking (like only masking `AdminSecret`) is insufficient. A full deep copy (`MaskedClone`) is necessary to traverse and mask all secrets before serialization.
**Prevention:** Always implement and use a deep cloning method to mask all sensitive fields before serializing structs containing credentials.
