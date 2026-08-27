## 2025-02-14 - Secure Config Directory Permissions
**Vulnerability:** The configuration directory `~/.config/tdb/` and configuration file `~/.config/tdb/config.yaml` containing API keys and admin secrets were created with overly permissive `0755` (directory) permissions, allowing other users on the system to potentially read the file if standard file mask logic doesn't prevent access to the file.
**Learning:** Config directories holding secrets should always use `0700` to prevent any read/write access from other system users, regardless of file permissions (which were `0600`).
**Prevention:** Always verify `os.MkdirAll` permissions for paths that will store sensitive data, ensuring they are restricted to `0700` (`os.FileMode(0700)`).
