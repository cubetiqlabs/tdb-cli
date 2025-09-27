# TinyDB CLI (`tdb`)

The `tdb` command-line tool lets you administer TinyDB from your terminal. It wraps the admin and tenant APIs so you can provision tenants, issue API keys, manage applications, and keep local credentials synced.

## Features

-   Works with both admin and tenant-scoped endpoints
-   Persists CLI configuration and API keys under your user config directory
-   Colorized, column-aligned tables (with automatic `NO_COLOR` support)
-   First-class helpers for provisioning tenants, generating keys, and creating applications
-   Friendly error messages when configuration is missing or incomplete

## Installation

### 1. Download a release binary

Prebuilt archives are published for every tagged release under [`tdb-cli-v*` GitHub releases](https://github.com/cubetiqlabs/tinydb/releases).

1. Download the archive matching your platform (for example `tdb-cli_0.3.0_darwin_arm64.zip`).
2. Extract the archive and place the `tdb` (or `tdb.exe`) binary somewhere on your `PATH` (e.g. `/usr/local/bin`).
3. Make the binary executable if required:

    ```bash
    chmod +x /usr/local/bin/tdb
    ```

4. Verify the install:

    ```bash
    tdb version
    ```

### 2. Build from source with Go

Prerequisites: Go 1.22+.

```bash
git clone https://github.com/cubetiqlabs/tinydb.git
cd tinydb
go build -trimpath -ldflags "-s -w" -o tdb ./cmd/tdb
mv tdb /usr/local/bin/
```

Optionally embed the current Git version by reusing the project Makefile:

```bash
make tdb-cli
```

Artifacts will be written to `dist/tdb-cli/<os>_<arch>/` alongside `.zip`/`.tar.gz` archives.

### 3. Upgrade or reinstall

Simply replace the binary with the latest release or rebuild from source. Run `tdb version` to confirm the update.

## Configuration

The CLI keeps persistent state in a YAML file located at:

-   macOS/Linux: `~/.config/tdb/config.yaml`
-   Windows: `%AppData%\tdb\config.yaml`

You can override the path with `--config /custom/path.yaml` on any command.

Typical bootstrapping flow:

```bash
tdb config set endpoint https://tinydb-prod.ctdn.net
tdb config set admin-secret super-secret-token
```

Inspect the current config at any time:

```bash
tdb config show
```

### Storing API keys

Cache generated keys locally so tenant-scoped commands can authenticate automatically:

```bash
tdb config store-key tenant_123 primary --stdin <<<'TDB_API_KEY_...' --tenant-name "Marketing"
```

Flags:

-   `--key` / `--stdin` – provide the raw API key value
-   `--prefix` – optional recorded prefix for reference
-   `--app-id` – associate the key with an application
-   `--default` – mark the key as the default for the tenant
-   `--tenant-name` – friendly label stored in config

Remove or adjust entries later:

```bash
tdb config delete-key tenant_123 primary
tdb config set default-key tenant_123 read-only
```

Set a default tenant so tenant operations do not require `--tenant` each time:

```bash
tdb config set default-tenant tenant_123
```

## Global flags

Every command supports the following persistent flags:

-   `--config` – path to the CLI config file
-   `--endpoint` – override the API endpoint for the current invocation
-   `--admin-secret` – override the admin secret for the current invocation

Use these when scripting against multiple clusters without editing your stored config.

## Command overview

### Version

```bash
tdb version
```

Prints `tdb/<version>` using the embedded build metadata.

### Config commands

```bash
tdb config show
```

Display the current config (with the admin secret masked). Other sub-commands:

-   `tdb config set endpoint <url>`
-   `tdb config set admin-secret <secret>`
-   `tdb config set default-key <tenant_id> <alias>`
-   `tdb config set default-tenant <tenant_id>`
-   `tdb config set tenant-name <tenant_id> <label>`
-   `tdb config store-key <tenant_id> <alias> [flags...]`
-   `tdb config delete-key <tenant_id> <alias>`

### Admin commands

Admin commands require the admin secret to be configured. The CLI automatically sends it as `X-Admin-Secret`.

List tenants:

```bash
tdb admin tenants list
```

Create a tenant and generate an API key in one step:

```bash
tdb admin tenants create \
  --name "Acme Corp" \
  --description "North America region" \
  --with-key \
  --save-key-as acme-admin \
  --set-default \
  --tenant-name "Acme (NA)"
```

Manage keys:

```bash
tdb admin keys list --tenant tenant_123

tdb admin keys create \
  --tenant tenant_123 \
  --app-id webshop \
  --description "Checkout service" \
  --save-key-as checkout \
  --set-default

tdb admin keys revoke key_prefix
```

### Tenant commands

Tenant operations require an API key. The CLI will use, in priority order:

1. `--api-key` if supplied
2. Stored key referenced by `--key`
3. The tenant's configured default key

If no `--tenant` is provided the configured `default_tenant` is used.

List applications:

```bash
tdb tenant apps list --tenant tenant_123 --key checkout
```

Create an application and store the generated key:

```bash
tdb tenant apps create \
  --tenant tenant_123 \
  --key checkout \
  --name "Data Sync" \
  --description "Realtime sync client" \
  --with-key \
  --store-key-as data-sync \
  --set-default
```

Fetch details for a single application:

```bash
tdb tenant apps get app_456 --tenant tenant_123 --key checkout
```

## Output and color

The CLI renders unicode tables with alternating row styles when stdout is a TTY. Set `NO_COLOR=1` to disable ANSI colors, or `FORCE_COLOR=1` to force-enable them in pipelines.

## Troubleshooting

-   "endpoint not configured" – run `tdb config set endpoint <url>`
-   "admin secret not configured" – set the secret via `tdb config set admin-secret <secret>` or pass `--admin-secret`
-   "tenant ... not found in config" – store the tenant's API key (or pass `--api-key`) before running tenant commands
-   Add `--config /path/to/test.yaml` to work with disposable configs during automation

With the basics configured you can script `tdb` in CI/CD pipelines or on your workstation to automate TinyDB tenant onboarding and application management.
