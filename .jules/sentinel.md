## 2024-05-18 - Prevent Credential Leakage in CLI Config Display
**Vulnerability:** API keys were exposed in plaintext when running `tdb config show` because only the AdminSecret was masked, while APIKeys were included verbatim in a shallow copy.
**Learning:** Go's map assignment is by reference. A shallow copy of a struct containing maps of secrets is insufficient because the maps (and therefore the secrets) are still referenced and serialized in plaintext.
**Prevention:** Always perform a deep copy (e.g. `MaskedClone()`) to iterate through and mask secrets inside nested maps or pointers before serialization.
