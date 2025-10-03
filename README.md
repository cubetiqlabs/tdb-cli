# TinyDB CLI (`tdb`)

[![Latest Release](https://img.shields.io/github/v/release/cubetiqlabs/tdb-cli?sort=semver)](https://github.com/cubetiqlabs/tdb-cli/releases)
[![Release workflow status](https://github.com/cubetiqlabs/tdb-cli/actions/workflows/release.yml/badge.svg)](https://github.com/cubetiqlabs/tdb-cli/actions/workflows/release.yml)

TinyDB CLI is a standalone command-line interface for managing TinyDB tenants, collections, documents, saved queries, and configuration.

## Features

-   **Tenant & API Key Management** - Secure multi-tenant operations
-   **Collection & Schema Lifecycle** - Dynamic schema management with validation
-   **Document CRUD & Bulk Operations** - Efficient data manipulation at scale
-   **Saved Query Management** - Store and execute complex queries
-   **Report Query Execution** - Ad-hoc aggregations and analytics
-   **Audit Log Inspection** - Comprehensive filtering and sorting
-   **Snapshot Management** - Full and incremental backups with encryption
-   **Version Metadata** - Optimistic concurrency control
-   **Real-Time Auth Check** - `/api/me` endpoint integration
-   **Offline Export Helpers** - Cross-platform install scripts
-   **Self-Upgrade** - `tdb upgrade` command for easy updates

## 📚 Documentation

-   **[Quick Start Guide](docs/QUICKSTART.md)** - Get started in minutes
-   **[Snapshot Management](docs/SNAPSHOT_CLI.md)** - Backup and restore guide
-   **[CLI Enhancements](docs/CLI_ENHANCEMENTS.md)** - Comprehensive command reference
-   **[Developer Guide](docs/DEVELOPER_GUIDE.md)** - Extend and contribute to the CLI
-   **[Contributing](docs/CONTRIBUTING.md)** - Contribution guidelines

All commands include detailed help text with examples:
```bash
tdb tenant collections --help  # See all collection commands
tdb tenant documents create --help  # Get detailed examples
```

## Installation

Prebuilt archives are available on the [`tdb-cli` Releases page](https://github.com/cubetiqlabs/tdb-cli/releases). Pushing a tag that matches `v*` triggers the GitHub Actions release workflow, bundling binaries for macOS (arm64/amd64), Linux (arm64/amd64), and Windows (arm64/amd64).

### macOS & Linux

```bash
curl -fsSL https://raw.githubusercontent.com/cubetiqlabs/tdb-cli/main/scripts/install.sh | bash
```

### Windows (PowerShell)

```powershell
iwr https://raw.githubusercontent.com/cubetiqlabs/tdb-cli/main/scripts/install.ps1 -UseBasicParsing | iex
```

### From Source

```bash
git clone https://github.com/cubetiqlabs/tdb-cli.git
cd tdb-cli
go build -trimpath -ldflags "-s -w" -o tdb ./cmd/tdb
```

Or install the latest tagged version directly:

```bash
go install github.com/cubetiqlabs/tdb-cli/cmd/tdb@latest
```

## Usage

```bash
tdb --help
```

See `tdb <command> --help` for details on each command. Configuration is stored under `~/.config/tdb/config.yaml` by default.

### Quick Examples

```bash
# List collections
tdb tenant collections list --api-key $API_KEY

# Create a document
tdb tenant documents create users \
  --data '{"name":"Alice","email":"alice@example.com"}' \
  --api-key $API_KEY

# Create a backup
tdb tenant snapshots create \
  --collection users \
  --name "Daily backup" \
  --encrypt \
  --api-key $API_KEY

# View audit logs
tdb tenant audit --collection users --since 24h --api-key $API_KEY
```

For comprehensive examples and workflows, see the [Quick Start Guide](docs/QUICKSTART.md).

### Collection inspection

Use `tdb tenant collections list` with the new inspection flags to understand stored schemas and document shapes:

-   `--show-schema` prints the persisted JSON schema for each collection after the tabular summary.
-   `--inspect-docs` samples up to `--inspect-limit` documents (default 10) and infers field types, highlighting gaps versus the stored schema.
-   `--describe` is a shortcut that enables both of the above flags.

Example:

```bash
tdb tenant collections list \
    --describe \
    --tenant TENANT_ID \
    --app app_123
```

Fields marked with `*` are present in sampled documents but missing from the stored schema—handy for spotting drift or undocumented fields.

### Audit logs

List the most recent audit entries for a tenant, optionally filtering by collection, document, actor, or time window:

```bash
tdb tenant audit --collection users --since 48h --sort created_at --raw-pretty
```

Relative durations (`1h`, `2d`, etc.) are resolved against the current time; fallback to RFC3339 timestamps for absolute ranges. Use `--raw` for compact JSON and `--raw-pretty` for pretty-printed output.

### Snapshots

Create, restore, list, and delete collection snapshots for backup and disaster recovery:

```bash
# List all snapshots
tdb tenant snapshots list --api-key $API_KEY

# List snapshots for a specific collection
tdb tenant snapshots list --api-key $API_KEY --collection users

# Create a full snapshot
tdb tenant snapshots create \
    --api-key $API_KEY \
    --collection users \
    --name "Daily backup"

# Create an encrypted snapshot with S3 storage
tdb tenant snapshots create \
    --api-key $API_KEY \
    --collection orders \
    --name "Production backup" \
    --encrypt \
    --storage s3

# Create an incremental snapshot
tdb tenant snapshots create \
    --api-key $API_KEY \
    --collection users \
    --name "Incremental" \
    --incremental \
    --parent-snapshot snap-parent-123

# Get snapshot details
tdb tenant snapshots get --api-key $API_KEY --snapshot snap-123

# Restore to original collection
tdb tenant snapshots restore \
    --api-key $API_KEY \
    --snapshot snap-123

# Restore to a different collection
tdb tenant snapshots restore \
    --api-key $API_KEY \
    --snapshot snap-123 \
    --target-collection users-restored

# Delete a snapshot
tdb tenant snapshots delete \
    --api-key $API_KEY \
    --snapshot snap-123 \
    --force
```

Snapshots support both full and incremental backups, optional encryption at rest, and multiple storage providers (local, S3, GCS). Use aliases like `backup` or `snapshot` for convenience.

## Syncing existing data

The CLI can upsert existing collections and documents from JSON definitions. Each command accepts inline JSON, a file path, or `--stdin`.

-   Create or update collection schemas and primary-key metadata:

    ```bash
    tdb tenant collections sync --file collections.json
    ```

    The payload can be an array or object keyed by collection name:

    ```json
    [
        {
            "name": "users",
            "schema": { "type": "object" },
            "primary_key": { "field": "id", "type": "string" }
        }
    ]
    ```

    Collections that don’t already exist are provisioned automatically using the supplied schema and primary-key definition; existing collections are updated in place.

-   Patch or replace documents by primary key:

    ```bash
    tdb tenant documents sync users --mode patch --stdin < users.jsonl
    ```

    Each document must include the primary key (defaults to the collection key). Reserved metadata fields such as `id`, `key`, and timestamps are stripped automatically when patching. If a document is missing it will be created by default; pass `--skip-missing` to keep the old “update only” behavior. Use `--mode update` to perform full replacements instead of JSON merge patches.

## Releases

Releases are published automatically when new tags are pushed (e.g. `v1.2.3`). Each release contains prebuilt binaries for macOS (arm64/amd64), Linux (arm64/amd64), and Windows (amd64/arm64).

Use the helper script to create and push a tag:

```bash
scripts/tag_release.sh 0.4.0 "Release v0.4.0"
```

Override the remote or branch by setting `REMOTE` / `BRANCH`. After the tag is pushed, GitHub Actions runs `.github/workflows/release.yml` to publish artifacts automatically.

## Contributing

We welcome contributions! Please see our [Contributing Guide](docs/CONTRIBUTING.md) and [Developer Guide](docs/DEVELOPER_GUIDE.md) for details.

Quick start for contributors:

1. Fork the repository and create a feature branch
2. Follow the [command development patterns](docs/DEVELOPER_GUIDE.md#command-development-lifecycle)
3. Run `go test ./...` and `go vet ./...` before opening a pull request
4. Open a PR targeting the `main` branch

All commands should include:
- Detailed `Long` description
- At least 3-7 practical examples
- Comprehensive error handling
- Unit tests

See existing commands in `pkg/tdbcli/cli/` for reference patterns.

## License

MIT © CUBIS Labs
