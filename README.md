# TinyDB CLI (`tdb`)

[![Latest Release](https://img.shields.io/github/v/release/cubetiqlabs/tdb-cli?sort=semver)](https://github.com/cubetiqlabs/tdb-cli/releases)
[![Release workflow status](https://github.com/cubetiqlabs/tdb-cli/actions/workflows/release.yml/badge.svg)](https://github.com/cubetiqlabs/tdb-cli/actions/workflows/release.yml)

TinyDB CLI is a standalone command-line interface for managing TinyDB tenants, collections, documents, saved queries, and configuration.

## Features

- Tenant and API key management
- Collection & schema lifecycle commands
- Document CRUD and bulk operations
- Saved query lifecycle and execution helpers
- Real-time authentication check via `/api/me`
- Offline export helpers and install scripts
- Self-upgrade via `tdb upgrade`

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

MIT © CUBETIQ Labs
