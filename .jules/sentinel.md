## 2024-05-24 - API Key Leakage in Config Outputs
**Vulnerability:** The CLI `config show` and `config list --raw` commands leaked raw API keys from the configuration because the serialization processes did not mask or filter out the secrets stored in the nested `Tenants` map.
**Learning:** When serializing configurations containing nested maps with secrets, shallow copying or partial masking is insufficient. Deep cloning (e.g., a `MaskedClone` method) must be implemented and consistently applied before rendering to prevent sensitive data exposure.
**Prevention:** Implement and mandate the use of a deep copy method (`MaskedClone`) for all configuration output paths to guarantee that both top-level and nested secrets are safely masked prior to serialization.
