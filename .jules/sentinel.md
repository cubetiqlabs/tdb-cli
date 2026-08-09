## 2025-02-18 - Prevent Credential Leakage in Config Outputs
**Vulnerability:** API keys and nested credentials were leaked in plaintext during CLI output serialization (`tdb config show`, `tdb config list --raw`) because deep cloning was not performed.
**Learning:** When serializing configurations containing nested maps with secrets, simple pointer dereferencing (`display := *env.Config`) drops unmapped fields or keeps shallow copies of maps, leading to accidental plaintext leakage when masking logic assumes a distinct copy.
**Prevention:** Always implement a explicit deep clone method (like `MaskedClone()`) that reallocates nested maps and deeply copies configurations before masking and serializing output to standard streams.
