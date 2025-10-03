# TinyDB CLI - Quick Start Guide

Welcome to TinyDB CLI! This guide will help you get started quickly with practical examples.

## Table of Contents

1. [Installation](#installation)
2. [Configuration](#configuration)
3. [Common Workflows](#common-workflows)
4. [Command Reference](#command-reference)
5. [Tips & Tricks](#tips--tricks)

## Installation

See the main [README.md](README.md) for installation instructions.

## Configuration

### Store Your API Key

```bash
# Store your API key for easy reuse
tdb config store-key your-tenant-id my-key \
  --key "tdb_your_api_key_here" \
  --set-default \
  --description "My production key"

# Now you can use --key my-key instead of --api-key
tdb tenant collections list --key my-key
```

### View Current Config

```bash
# See your stored configuration
tdb config show
```

## Common Workflows

### 1. Working with Collections

```bash
# List all collections
tdb tenant collections list --key my-key

# Create a collection with schema
tdb tenant collections create \
  --name users \
  --schema '{"type":"object","required":["email"],"properties":{"email":{"type":"string"}}}' \
  --pk-field user_id \
  --pk-type string \
  --pk-auto \
  --key my-key

# Inspect collection schema and documents
tdb tenant collections list --describe --key my-key

# Get specific collection details
tdb tenant collections get users --key my-key
```

### 2. Managing Documents

```bash
# Create a document
tdb tenant documents create users \
  --data '{"email":"alice@example.com","name":"Alice","role":"admin"}' \
  --key my-key

# List documents with filters
tdb tenant documents list users \
  --filter role=admin \
  --filter status=active \
  --sort created_at:desc \
  --limit 20 \
  --key my-key

# Get a specific document
tdb tenant documents get users user_123 --key my-key

# Update a document (partial)
tdb tenant documents patch users user_123 \
  --data '{"status":"active","last_login":"2025-01-15T10:00:00Z"}' \
  --key my-key

# Delete a document (soft delete)
tdb tenant documents delete users user_456 --key my-key

# Permanently purge a document
tdb tenant documents delete users user_789 --purge --key my-key
```

### 3. Bulk Operations

```bash
# Sync collections from file
cat << EOF > collections.json
[
  {
    "name": "users",
    "schema": {"type": "object"},
    "primary_key": {"field": "email", "type": "string"}
  },
  {
    "name": "orders",
    "schema": {"type": "object"}
  }
]
EOF

tdb tenant collections sync --file collections.json --mode upsert --key my-key

# Sync documents from JSONL
cat << EOF > users.jsonl
{"email":"user1@example.com","name":"Alice","role":"admin"}
{"email":"user2@example.com","name":"Bob","role":"user"}
{"email":"user3@example.com","name":"Charlie","role":"user"}
EOF

tdb tenant documents sync users --file users.jsonl --key my-key

# Export documents
tdb tenant documents export users \
  --out users-backup.jsonl \
  --filter status=active \
  --key my-key
```

### 4. Backup and Restore

```bash
# Create a full snapshot
tdb tenant snapshots create \
  --collection users \
  --name "Daily backup $(date +%Y-%m-%d)" \
  --description "Automated daily backup" \
  --key my-key

# Create an encrypted snapshot for production
tdb tenant snapshots create \
  --collection orders \
  --name "Production backup" \
  --encrypt \
  --storage s3 \
  --key my-key

# List snapshots
tdb tenant snapshots list --collection users --key my-key

# Restore a snapshot
tdb tenant snapshots restore --snapshot snap-123 --key my-key

# Restore to a different collection
tdb tenant snapshots restore \
  --snapshot snap-123 \
  --target-collection users-restored \
  --key my-key

# Delete old snapshots
tdb tenant snapshots delete --snapshot snap-old-456 --force --key my-key
```

### 5. Monitoring and Audit

```bash
# View recent audit logs
tdb tenant audit --limit 50 --key my-key

# Filter audit logs by collection
tdb tenant audit --collection orders --since 24h --key my-key

# Find all deletes in the last week
tdb tenant audit \
  --operation delete \
  --since 168h \
  --sort created_at:desc \
  --key my-key

# Track changes by a specific actor
tdb tenant audit --actor user@example.com --since 48h --key my-key

# Detailed audit investigation
tdb tenant audit \
  --collection users \
  --document user_123 \
  --since 7d \
  --raw-pretty \
  --key my-key
```

### 6. Admin Operations

```bash
# List all tenants
tdb admin tenants list --admin-secret $ADMIN_SECRET

# Create a new tenant with API key
tdb admin tenants create \
  --name "Production" \
  --description "Production environment" \
  --with-key \
  --save-key-as prod-key \
  --set-default \
  --admin-secret $ADMIN_SECRET

# Create a staging tenant
tdb admin tenants create \
  --name "Staging" \
  --with-key \
  --admin-secret $ADMIN_SECRET
```

## Command Reference

### Quick Command Lookup

| Task | Command |
|------|---------|
| List collections | `tdb tenant collections list` |
| Create collection | `tdb tenant collections create --name NAME` |
| List documents | `tdb tenant documents list COLLECTION` |
| Create document | `tdb tenant documents create COLLECTION --data '{...}'` |
| Patch document | `tdb tenant documents patch COLLECTION ID --data '{...}'` |
| Delete document | `tdb tenant documents delete COLLECTION ID` |
| Sync documents | `tdb tenant documents sync COLLECTION --file FILE` |
| Create snapshot | `tdb tenant snapshots create --collection COLLECTION --name NAME` |
| Restore snapshot | `tdb tenant snapshots restore --snapshot ID` |
| View audit logs | `tdb tenant audit` |
| Store API key | `tdb config store-key TENANT_ID ALIAS --key KEY` |

### Get Help for Any Command

```bash
# General help
tdb --help

# Category help
tdb tenant --help
tdb tenant collections --help
tdb tenant documents --help
tdb tenant snapshots --help
tdb admin --help
tdb config --help

# Specific command help
tdb tenant collections create --help
tdb tenant documents sync --help
tdb tenant snapshots create --help
```

## Tips & Tricks

### 1. Use Stored Keys

Instead of typing `--api-key` every time, store your key once:

```bash
tdb config store-key my-tenant my-key --key "tdb_..." --set-default
tdb tenant collections list --key my-key
```

### 2. Pretty Print JSON

Add `--raw-pretty` for readable JSON output:

```bash
tdb tenant documents get users user_123 --raw-pretty
```

### 3. Field Selection for Performance

Only fetch the fields you need:

```bash
tdb tenant documents list users \
  --select id,email,created_at \
  --limit 1000
```

### 4. Use Cursor Pagination for Large Datasets

More efficient than offset pagination:

```bash
tdb tenant documents list users --limit 100 --cursor TOKEN
```

### 5. Pipe Data with stdin

```bash
echo '{"email":"test@example.com"}' | \
  tdb tenant documents create users --stdin
  
cat users.jsonl | tdb tenant documents sync users --stdin
```

### 6. Combine Filters

Multiple filters use AND logic:

```bash
tdb tenant documents list users \
  --filter status=active \
  --filter role=admin \
  --filter verified=true
```

### 7. Sort Multiple Fields

```bash
tdb tenant documents list orders \
  --sort status:asc \
  --sort created_at:desc
```

### 8. Export and Backup Before Major Changes

```bash
# Export documents before bulk update
tdb tenant documents export users --out backup.jsonl

# Create snapshot before collection deletion
tdb tenant snapshots create --collection old-data --name "Before deletion"
tdb tenant collections delete old-data
```

### 9. Use Environment Variables

```bash
export API_KEY="tdb_your_key_here"
export TENANT_ID="your-tenant-id"

tdb tenant collections list --api-key $API_KEY --tenant $TENANT_ID
```

### 10. Check Audit Logs for Troubleshooting

```bash
# Find recent errors or unexpected changes
tdb tenant audit --since 1h --collection problematic-collection
```

## Common Patterns

### Daily Backup Script

```bash
#!/bin/bash
DATE=$(date +%Y-%m-%d)
COLLECTIONS=("users" "orders" "products")

for collection in "${COLLECTIONS[@]}"; do
  tdb tenant snapshots create \
    --collection "$collection" \
    --name "Daily backup $DATE" \
    --encrypt \
    --storage s3 \
    --key my-key
  echo "Backed up $collection"
done
```

### Data Migration

```bash
# Export from source
tdb tenant documents export users \
  --out users-export.jsonl \
  --tenant source-tenant \
  --key source-key

# Import to destination
tdb tenant documents sync users \
  --file users-export.jsonl \
  --mode patch \
  --tenant dest-tenant \
  --key dest-key
```

### Bulk Updates

```bash
# Export, modify, re-import
tdb tenant documents export users --out users.jsonl
# Edit users.jsonl with your changes
tdb tenant documents sync users --file users.jsonl --mode patch
```

## Need More Help?

- Run `tdb COMMAND --help` for detailed examples
- Check [README.md](README.md) for setup and installation
- See [CLI_ENHANCEMENTS.md](CLI_ENHANCEMENTS.md) for complete documentation
- See [SNAPSHOT_CLI.md](SNAPSHOT_CLI.md) for snapshot details

Every command has been enhanced with comprehensive examples - just add `--help` to see them!
