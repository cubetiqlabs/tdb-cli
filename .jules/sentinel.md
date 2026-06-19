
## 2025-05-18 - Prevent Credential Leakage in Serialized Maps
**Vulnerability:** The CLI `config show` and `config list --raw` commands leaked nested `APIKeyEntry` secrets because of shallow copying of the config state before passing to JSON or YAML marshalers. Only `AdminSecret` was explicitly masked.
**Learning:** Shallow copies do not duplicate inner pointer data such as maps. Serializing a "modified" shallow copy of a struct containing nested maps with secrets will leak the unmasked secrets.
**Prevention:** Implement a deep `Clone()` method (or `MaskedClone()`) that explicitly iterates through nested maps and creates new, masked objects before returning the state for serialization.
