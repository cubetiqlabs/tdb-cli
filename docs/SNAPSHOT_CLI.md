# Snapshot Management CLI

This document describes the snapshot management commands added to `tdb-cli` for quick backup and restore operations.

## Overview

The snapshot commands provide a complete CLI interface for managing TinyDB collection snapshots, including:
- Creating full and incremental backups
- Listing and filtering snapshots
- Restoring snapshots to original or different collections
- Viewing detailed snapshot information
- Deleting snapshots

## Commands

All snapshot commands are available under `tdb tenant snapshots` (aliases: `snapshot`, `backup`, `backups`).

### List Snapshots

List all snapshots for the tenant, optionally filtered by collection:

```bash
# List all snapshots
tdb tenant snapshots list --api-key $API_KEY

# List snapshots for a specific collection
tdb tenant snapshots list --api-key $API_KEY --collection users

# List with pagination
tdb tenant snapshots list --api-key $API_KEY --limit 10 --offset 20

# Get raw JSON output
tdb tenant snapshots list --api-key $API_KEY --raw
```

**Flags:**
- `--collection`: Filter by collection ID
- `--limit`: Maximum number of snapshots to return (default: 50)
- `--offset`: Number of snapshots to skip
- `--raw`: Print raw JSON response

### Create Snapshot

Create a full or incremental snapshot of a collection:

```bash
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
    --description "Daily production backup" \
    --encrypt \
    --storage s3

# Create an incremental snapshot
tdb tenant snapshots create \
    --api-key $API_KEY \
    --collection users \
    --name "Incremental backup" \
    --incremental \
    --parent-snapshot snap-parent-123
```

**Flags:**
- `--collection`: Collection ID (required)
- `--name`: Snapshot name (required)
- `--description`: Snapshot description
- `--incremental`: Create incremental snapshot
- `--parent-snapshot`: Parent snapshot ID for incremental snapshots
- `--encrypt`: Encrypt snapshot data
- `--storage`: Storage provider (local, s3, gcs)
- `--raw`: Print raw JSON response

### Get Snapshot Details

View detailed information about a specific snapshot:

```bash
tdb tenant snapshots get --api-key $API_KEY --snapshot snap-123

# Get raw JSON output
tdb tenant snapshots get --api-key $API_KEY --snapshot snap-123 --raw
```

**Flags:**
- `--snapshot`: Snapshot ID (required)
- `--raw`: Print raw JSON response

**Output Example:**
```
Snapshot Details:
  ID:                snap-abc123
  Name:              Daily backup
  Collection:        users
  Type:              full
  Status:            completed
  Documents:         10,523
  Size:              2.5 MiB
  Encrypted:         Yes
  Storage:           s3
  Created:           2025-01-15 14:30:00
  Completed:         2025-01-15 14:32:15
  Compression:       gzip
  Compression Ratio: 3.2x
```

### Restore Snapshot

Restore a snapshot to its original collection or a different collection:

```bash
# Restore to original collection
tdb tenant snapshots restore \
    --api-key $API_KEY \
    --snapshot snap-123

# Restore to a different collection
tdb tenant snapshots restore \
    --api-key $API_KEY \
    --snapshot snap-123 \
    --target-collection users-restored

# Get raw JSON output
tdb tenant snapshots restore \
    --api-key $API_KEY \
    --snapshot snap-123 \
    --raw
```

**Flags:**
- `--snapshot`: Snapshot ID (required)
- `--target-collection`: Target collection ID (defaults to original)
- `--raw`: Print raw JSON response

### Delete Snapshot

Delete a snapshot:

```bash
# Delete with confirmation prompt
tdb tenant snapshots delete \
    --api-key $API_KEY \
    --snapshot snap-123

# Force delete without confirmation
tdb tenant snapshots delete \
    --api-key $API_KEY \
    --snapshot snap-123 \
    --force

# Get raw JSON output
tdb tenant snapshots delete \
    --api-key $API_KEY \
    --snapshot snap-123 \
    --force \
    --raw
```

**Flags:**
- `--snapshot`: Snapshot ID (required)
- `--force`: Skip confirmation prompt
- `--raw`: Print raw JSON response

## Authentication

All snapshot commands support the same authentication options as other tenant commands:

- `--api-key`: Raw API key to authenticate with (overrides stored keys)
- `--key`: Stored key alias to authenticate with
- `--tenant`: Tenant ID (defaults to configured value)

## Common Use Cases

### Daily Backup

```bash
#!/bin/bash
DATE=$(date +%Y-%m-%d)
tdb tenant snapshots create \
    --api-key $API_KEY \
    --collection users \
    --name "Daily backup $DATE" \
    --encrypt \
    --storage s3
```

### Incremental Backup Chain

```bash
# Create initial full snapshot
PARENT=$(tdb tenant snapshots create \
    --api-key $API_KEY \
    --collection users \
    --name "Full backup" \
    --raw | jq -r '.id')

# Create incremental snapshots
tdb tenant snapshots create \
    --api-key $API_KEY \
    --collection users \
    --name "Incremental 1" \
    --incremental \
    --parent-snapshot $PARENT
```

### Disaster Recovery

```bash
# Find the latest snapshot
SNAPSHOT=$(tdb tenant snapshots list \
    --api-key $API_KEY \
    --collection users \
    --limit 1 \
    --raw | jq -r '.snapshots[0].id')

# Restore to a recovery collection
tdb tenant snapshots restore \
    --api-key $API_KEY \
    --snapshot $SNAPSHOT \
    --target-collection users-recovery
```

### Cleanup Old Snapshots

```bash
# List snapshots older than 30 days and delete them
tdb tenant snapshots list \
    --api-key $API_KEY \
    --collection users \
    --raw | \
jq -r '.snapshots[] | select(.created_at < (now - 2592000)) | .id' | \
while read snapshot_id; do
    tdb tenant snapshots delete \
        --api-key $API_KEY \
        --snapshot $snapshot_id \
        --force
done
```

## Implementation Details

### Files Modified

1. **`pkg/tdbcli/cli/tenant_snapshots_cmd.go`** (402 lines)
   - 6 snapshot commands with full flag support
   - Table rendering for list output
   - Detailed text output for get command
   - Confirmation prompts for destructive operations

2. **`pkg/tdbcli/client/types.go`** (+55 lines)
   - `Snapshot` struct (20 fields)
   - `CreateSnapshotRequest`
   - `RestoreSnapshotRequest`
   - `RestoreSnapshotResponse`
   - `SnapshotListResponse`

3. **`pkg/tdbcli/client/tenant.go`** (+100 lines)
   - `ListSnapshots()`: GET /api/snapshots
   - `GetSnapshot()`: GET /api/snapshots/:id
   - `CreateSnapshot()`: POST /api/snapshots
   - `RestoreSnapshot()`: POST /api/snapshots/:id/restore
   - `DeleteSnapshot()`: DELETE /api/snapshots/:id

4. **`pkg/tdbcli/cli/tenant_cmd.go`** (+3 lines)
   - Registered snapshots command in tenant command hierarchy

5. **`README.md`** (updated)
   - Added snapshots to features list
   - Added comprehensive snapshots section with examples

### API Endpoints

The CLI commands map to the following TinyDB API endpoints:

- `GET /api/snapshots?collection_id=&limit=&offset=` - List snapshots
- `GET /api/snapshots/:id` - Get snapshot details
- `POST /api/snapshots` - Create snapshot
- `POST /api/snapshots/:id/restore` - Restore snapshot
- `DELETE /api/snapshots/:id` - Delete snapshot

All endpoints require authentication via the `X-API-Key` header.

## Testing

Build and test the CLI:

```bash
cd clients/tdb-cli
go build -o tdb cmd/tdb/main.go
./tdb tenant snapshots --help
go test ./...
```

All existing tests pass, and the new commands integrate seamlessly with the existing CLI architecture.
