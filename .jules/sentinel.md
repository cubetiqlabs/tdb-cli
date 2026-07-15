## 2024-05-18 - Mask Secrets in Serialized Configurations
**Vulnerability:** The CLI exposed plain text API keys and shallow copied `AdminSecret` when dumping configurations using `tdb config list --raw` and `tdb config show`, risking credential leakage in logs or terminal outputs.
**Learning:** Using shallow copies or directly marshaling complex nested structs containing secrets leaks sensitive data. Deep cloning with secret masking is necessary to prevent accidental exposure during serialization.
**Prevention:** Implement deep-copy routines like `MaskedClone` on configuration structures that traverse and mask all nested secret fields before passing data to `json.Marshal` or `yaml.Marshal`.
