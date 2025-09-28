# TinyDB CLI (`tdb`)

[![Latest Release](https://img.shields.io/github/v/release/cubetiqlabs/tdb-cli?sort=semver)](https://github.com/cubetiqlabs/tdb-cli/releases)
[![Release workflow status](https://github.com/cubetiqlabs/tdb-cli/actions/workflows/release.yml/badge.svg)](https://github.com/cubetiqlabs/tdb-cli/actions/workflows/release.yml)

The `tdb` command-line tool lets you administer TinyDB from your terminal. It wraps the admin and tenant APIs so you can provision tenants, issue API keys, manage applications, and keep local credentials synced.

## Features

-   Works with both admin and tenant-scoped endpoints
-   Persists CLI configuration and API keys under your user config directory
-   Colorized, column-aligned tables (with automatic `NO_COLOR` support)
-   First-class helpers for provisioning tenants, generating keys, and creating applications
-   Friendly error messages when configuration is missing or incomplete

## Installation

### 1. Download a release binary

Prebuilt archives are published for every tagged release under [`tdb-cli` GitHub releases](https://github.com/cubetiqlabs/tdb-cli/releases). When a tag matching `v*` is pushed, GitHub Actions builds six platform archives and publishes them automatically—see [`.github/workflows/release.yml`](https://github.com/cubetiqlabs/tdb-cli/blob/main/.github/workflows/release.yml).

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
git clone https://github.com/cubetiqlabs/tdb-cli.git
cd tdb-cli
go build -trimpath -ldflags "-s -w" -o tdb ./cmd/tdb
mv tdb /usr/local/bin/
```

Optionally embed build metadata via the repo Makefile:

```bash
make build
```

Artifacts will be written to `dist/<target>/` alongside `.zip`/`.tar.gz` archives.

### 3. Upgrade or reinstall

Simply replace the binary with the latest release or rebuild from source. Run `tdb version` to confirm the update.

### 4. Tagging a release from the TinyDB monorepo

When changes are ready to publish, create and push a version tag directly from the TinyDB repository using the helper script:

```bash
scripts/tag_release.sh 0.4.0 "Release v0.4.0"
```

The script ensures your working tree is clean, fast-forwards the branch, prefixes the tag with `v` if missing, and pushes it to `origin` (override with `REMOTE=...`). Once the tag is pushed, rerun `scripts/export_tdb_cli.sh` to publish the latest CLI code to the public `tdb-cli` repository.

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

Manage keys (defaults to the configured tenant when `--tenant` is omitted and will print which tenant is selected):

```bash
tdb admin keys list

tdb admin keys create \
  --app-id webshop \
  --description "Checkout service" \
  --save-key-as checkout \
  --set-default

tdb admin keys revoke key_prefix
```

The `tdb admin keys list` table now includes **HAS APP**, **STATUS**, **CREATED**, **LAST USED**, and **REVOKED** columns with friendly timestamps. Pass `--hide-revoked` to omit revoked keys from the output when you only care about active credentials.

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

Manage collections:

```bash
# list collections (optionally scoped by app)
tdb tenant collections list --tenant tenant_123 --key checkout

# create from schema file and set a primary key
tdb tenant collections create \
  --tenant tenant_123 \
  --key checkout \
  --name inventory \
  --schema-file ./inventory.schema.json \
  --primary-key-field sku \
  --primary-key-type string

# update schema
tdb tenant collections update inventory --tenant tenant_123 --key checkout --schema '{"type":"object"}'

# delete
tdb tenant collections delete inventory --tenant tenant_123 --key checkout
```

Work with documents:

```bash
# list documents with filters and projections
tdb tenant documents list inventory \
  --tenant tenant_123 --key checkout \
  --filter status=active --select id,name,price

# insert from a JSON file
tdb tenant documents create inventory --tenant tenant_123 --key checkout --file ./doc.json

# patch a document from stdin
tdb tenant documents patch inventory doc_001 --tenant tenant_123 --key checkout --stdin <<<'{"price": 42.5}'

# bulk insert from array
tdb tenant documents bulk-create inventory --tenant tenant_123 --key checkout --file ./bulk.json

# purge a document permanently
tdb tenant documents delete inventory doc_001 --tenant tenant_123 --key checkout --purge --confirm

# export a collection to a file
tdb tenant documents export inventory \
  --tenant tenant_123 --key checkout \
  --out ./inventory.jsonl --format jsonl --include-meta

```

Manage saved queries:

```bash
# list saved queries
tdb tenant queries list --tenant tenant_123 --key checkout

# create or upsert from JSON file
tdb tenant queries create \
  --tenant tenant_123 --key checkout \
  --file ./queries/daily_sales.json

# replace an existing query by name
tdb tenant queries put daily_sales \
  --tenant tenant_123 --key checkout \
  --file ./queries/daily_sales.json

# execute by name with runtime parameters
tdb tenant queries execute daily_sales \
  --tenant tenant_123 --key checkout --by-name \
  --params '{"params":{"date_from":"2025-09-01","date_to":"2025-09-30"}}'

# delete or purge a saved query (by name or id)
tdb tenant queries delete daily_sales \
  --tenant tenant_123 --key checkout --by-name --purge --confirm

# scaffold a params payload for local editing
tdb tenant queries params-template daily_sales \
  --tenant tenant_123 --key checkout --by-name --out ./queries/daily_sales.params.json
```

Validate your API key configuration at any time:

```bash
tdb tenant auth --tenant tenant_123 --key checkout
```

Pass `--raw` to inspect the raw `/api/me` payload or `--app-id` to override the scoped application when testing aliases.

### Upgrade and Installation

Check for a newer CLI release and install it in-place:

```bash
tdb upgrade
```

Use `tdb upgrade --check` to only report availability without downloading.

Quick install (macOS/Linux):

```bash
curl -fsSL https://raw.githubusercontent.com/cubetiqlabs/tdb-cli/main/scripts/install.sh | bash
```

Quick install (Windows PowerShell):

```powershell
irm https://raw.githubusercontent.com/cubetiqlabs/tdb-cli/main/scripts/install.ps1 | iex
```

## Output and color

The CLI renders unicode tables with alternating row styles when stdout is a TTY. Set `NO_COLOR=1` to disable ANSI colors, or `FORCE_COLOR=1` to force-enable them in pipelines.

## Troubleshooting

-   "endpoint not configured" – run `tdb config set endpoint <url>`
-   "admin secret not configured" – set the secret via `tdb config set admin-secret <secret>` or pass `--admin-secret`
-   "tenant ... not found in config" – store the tenant's API key (or pass `--api-key`) before running tenant commands
-   Add `--config /path/to/test.yaml` to work with disposable configs during automation

With the basics configured you can script `tdb` in CI/CD pipelines or on your workstation to automate TinyDB tenant onboarding and application management.

## Shell Completion

The `tdb` CLI supports shell completion for Bash and Zsh.

### Bash

To enable completion for the current session:

```bash
tdb completion bash | source /dev/stdin
```

To enable completion for all sessions:

-   **Linux:**
    ```bash
    tdb completion bash > /etc/bash_completion.d/tdb
    ```
-   **macOS (Homebrew):**
    ```bash
    tdb completion bash > /usr/local/etc/bash_completion.d/tdb
    ```

### Zsh

To enable completion for the current session:

```zsh
autoload -U compinit; compinit
tdb completion zsh | source /dev/stdin
```

To enable completion for all sessions, add to your `~/.zshrc`:

```zsh
tdb completion zsh > "${fpath[1]}/_tdb"
```
