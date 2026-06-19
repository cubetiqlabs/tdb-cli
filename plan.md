1. **Add `MaskedClone()` to `pkg/tdbcli/config/config.go`**
   - Implement `maskSecret(string)` helper function to reduce duplication.
   - Refactor `MaskedAdminSecret()` to use `maskSecret`.
   - Add `MaskedClone() Config` which creates a deep copy of the config and masks both `AdminSecret` and all `Key` fields in `APIKeyEntry` structs within `Tenants`.

2. **Update `config show` command**
   - In `pkg/tdbcli/cli/config_cmd.go`, update the `config show` command to use `display := env.Config.MaskedClone()` instead of shallow copying and manually assigning the masked admin secret.

3. **Update `config list --raw` command**
   - In `pkg/tdbcli/cli/config_cmd.go`, update the `config list` command to use `printJSON(cmd, envCtx.Config.MaskedClone().Tenants)` when outputting raw JSON, ensuring API keys are not leaked.

4. **Complete pre-commit steps to ensure proper testing, verification, review, and reflection are done.**
   - Run `pnpm test`, `make test`, `make lint`, etc. to verify everything passes.

5. **Submit the change.**
   - Commit and submit PR.
