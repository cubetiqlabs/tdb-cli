# TinyDB CLI (`tdb`)

[![Latest Release](https://img.shields.io/github/v/release/cubetiqlabs/tdb-cli?sort=semver)](https://github.com/cubetiqlabs/tdb-cli/releases)
[![Release workflow status](https://github.com/cubetiqlabs/tdb-cli/actions/workflows/release.yml/badge.svg)](https://github.com/cubetiqlabs/tdb-cli/actions/workflows/release.yml)

TinyDB CLI is a standalone command-line interface for managing TinyDB tenants, collections, documents, saved queries, and configuration.

## Features

-   Tenant and API key management
-   Collection & schema lifecycle commands
-   Document CRUD and bulk operations
-   Saved query lifecycle and execution helpers
-   Audit log inspection with filtering and sorting
-   Document version metadata for optimistic concurrency (timestamps + sequence headers)
-   Real-time authentication check via `/api/me`
-   Offline export helpers and install scripts
-   Self-upgrade via `tdb upgrade`

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

### Audit logs

List the most recent audit entries for a tenant, optionally filtering by collection, document, actor, or time window:

```bash
tdb tenant audit --collection users --since 48h --sort created_at --raw-pretty
```

Relative durations (`1h`, `2d`, etc.) are resolved against the current time; fallback to RFC3339 timestamps for absolute ranges. Use `--raw` for compact JSON and `--raw-pretty` for pretty-printed output.

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

1. Fork the repository and create a feature branch.
2. Run `go test ./...` and `go vet ./...` before opening a pull request.
3. Open a PR targeting the `main` branch.

## License

MIT © CUBIS Labs
