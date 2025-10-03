# Command Reference

Complete reference for all `tdb` CLI commands with examples.

## Table of Contents

- [Configuration](#configuration)
- [Admin Commands](#admin-commands)
- [Collections](#collections)
- [Documents](#documents)
- [Queries](#queries)
- [Snapshots](#snapshots)
- [Audit Logs](#audit-logs)

---

## Configuration

### `tdb config show`

Display current CLI configuration.

**Usage:**
```bash
tdb config show
```

**Example:**
```bash
# Show all stored configuration
tdb config show

# Check API endpoint
tdb config show | grep endpoint
```

**Output:**
```yaml
endpoint: https://api.tinydb.com
default_tenant: tenant-123
default_app: app-456
keys:
  - name: production
    key: tdb_***
    tenant: tenant-123
```

---

### `tdb config store-key`

Store an API key for reuse across commands.

**Usage:**
```bash
tdb config store-key [flags]
```

**Flags:**
- `--key` - API key to store (required)
- `--name` - Friendly name for the key (default: "default")
- `--tenant` - Associate with specific tenant
- `--app` - Associate with specific application
- `--set-default` - Set as default key

**Examples:**
```bash
# Store a key with a name
tdb config store-key --key tdb_abc123 --name production

# Store and set as default
tdb config store-key --key tdb_xyz789 --name staging --set-default

# Store with tenant and app context
tdb config store-key \
  --key tdb_def456 \
  --name myproject \
  --tenant tenant-123 \
  --app app-456

# Read from stdin
echo "tdb_secret123" | tdb config store-key --name secure --stdin
```

---

## Admin Commands

### `tdb admin tenants list`

List all tenants (admin only).

**Usage:**
```bash
tdb admin tenants list --api-key ADMIN_KEY
```

**Flags:**
- `--api-key` - Admin API key (required)
- `--endpoint` - TinyDB API endpoint
- `--limit` - Maximum results (default: 50)

**Examples:**
```bash
# List all tenants
tdb admin tenants list --api-key $ADMIN_KEY

# Limit to 10 results
tdb admin tenants list --api-key $ADMIN_KEY --limit 10

# Output as JSON for processing
tdb admin tenants list --api-key $ADMIN_KEY | jq '.items[] | {id, name}'
```

---

### `tdb admin tenants create`

Create a new tenant (admin only).

**Usage:**
```bash
tdb admin tenants create NAME --api-key ADMIN_KEY
```

**Flags:**
- `--api-key` - Admin API key (required)
- `--endpoint` - TinyDB API endpoint
- `--generate-key` - Generate API key for tenant
- `--store-key` - Store generated key in config

**Examples:**
```bash
# Create a tenant
tdb admin tenants create acme-corp --api-key $ADMIN_KEY

# Create with API key generation
tdb admin tenants create acme-corp \
  --api-key $ADMIN_KEY \
  --generate-key

# Create and store the key locally
tdb admin tenants create acme-corp \
  --api-key $ADMIN_KEY \
  --generate-key \
  --store-key \
  --name acme-prod
```

---

## Collections

### `tdb tenant collections list`

List all collections in your tenant.

**Usage:**
```bash
tdb tenant collections list --api-key KEY
```

**Flags:**
- `--api-key` - API key (required)
- `--endpoint` - TinyDB API endpoint
- `--show-schema` - Display schema for each collection
- `--inspect-docs` - Sample documents and show field types
- `--inspect-limit` - Number of docs to sample (default: 10)
- `--describe` - Enable both schema and doc inspection

**Examples:**
```bash
# List all collections
tdb tenant collections list --api-key $API_KEY

# Show schemas
tdb tenant collections list --api-key $API_KEY --show-schema

# Inspect document structure
tdb tenant collections list --api-key $API_KEY --inspect-docs --inspect-limit 20

# Full inspection (schema + docs)
tdb tenant collections list --api-key $API_KEY --describe
```

---

### `tdb tenant collections get`

Get details about a specific collection.

**Usage:**
```bash
tdb tenant collections get COLLECTION --api-key KEY
```

**Examples:**
```bash
# Get collection details
tdb tenant collections get users --api-key $API_KEY

# Extract schema only
tdb tenant collections get users --api-key $API_KEY | jq '.schema'

# Check primary key configuration
tdb tenant collections get users --api-key $API_KEY | jq '.primary_key'
```

---

### `tdb tenant collections create`

Create a new collection.

**Usage:**
```bash
tdb tenant collections create NAME [flags]
```

**Flags:**
- `--api-key` - API key (required)
- `--schema` - JSON schema definition
- `--primary-key` - Primary key field name
- `--primary-key-type` - Primary key type (string, number, uuid)
- `--auto-generate` - Auto-generate primary keys

**Examples:**
```bash
# Simple collection with auto-generated UUIDs
tdb tenant collections create users \
  --api-key $API_KEY \
  --primary-key id \
  --primary-key-type uuid \
  --auto-generate

# With schema validation
tdb tenant collections create products \
  --api-key $API_KEY \
  --schema '{"fields":{"name":{"type":"string","required":true},"price":{"type":"number","min":0}}}' \
  --primary-key sku \
  --primary-key-type string

# From schema file
tdb tenant collections create orders \
  --api-key $API_KEY \
  --schema @schema.json \
  --primary-key id \
  --auto-generate
```

---

### `tdb tenant collections update`

Update an existing collection.

**Usage:**
```bash
tdb tenant collections update COLLECTION [flags]
```

**Flags:**
- `--api-key` - API key (required)
- `--schema` - New JSON schema
- `--add-index` - Add an index to a field

**Examples:**
```bash
# Update schema
tdb tenant collections update users \
  --api-key $API_KEY \
  --schema '{"fields":{"email":{"type":"string","required":true,"format":"email"}}}'

# Add index
tdb tenant collections update users \
  --api-key $API_KEY \
  --add-index email
```

---

### `tdb tenant collections delete`

Delete a collection and all its documents.

**Usage:**
```bash
tdb tenant collections delete COLLECTION --api-key KEY
```

**Flags:**
- `--force` - Skip confirmation prompt

**Examples:**
```bash
# Delete with confirmation
tdb tenant collections delete test-collection --api-key $API_KEY

# Force delete without prompt
tdb tenant collections delete old-data \
  --api-key $API_KEY \
  --force
```

---

### `tdb tenant collections sync`

Sync collection definitions from a file.

**Usage:**
```bash
tdb tenant collections sync --file FILE --api-key KEY
```

**Flags:**
- `--file` - JSON file with collection definitions
- `--stdin` - Read from stdin

**Examples:**
```bash
# Sync from file
tdb tenant collections sync --file collections.json --api-key $API_KEY

# From stdin
cat collections.json | tdb tenant collections sync --stdin --api-key $API_KEY

# Example JSON format
cat > collections.json <<EOF
[
  {
    "name": "users",
    "schema": {"fields": {"email": {"type": "string", "required": true}}},
    "primary_key": {"field": "id", "type": "uuid", "auto_generate": true}
  }
]
EOF
```

---

## Documents

### `tdb tenant documents list`

List documents in a collection.

**Usage:**
```bash
tdb tenant documents list COLLECTION --api-key KEY
```

**Flags:**
- `--api-key` - API key (required)
- `--limit` - Maximum results (default: 50)
- `--offset` - Pagination offset
- `--cursor` - Cursor for pagination
- `--filter` - Query filter JSON
- `--sort` - Sort field (can specify multiple)

**Examples:**
```bash
# List all documents
tdb tenant documents list users --api-key $API_KEY

# With pagination
tdb tenant documents list users --api-key $API_KEY --limit 100 --offset 200

# Cursor-based pagination
tdb tenant documents list users \
  --api-key $API_KEY \
  --limit 25 \
  --cursor "eyJjcmVhdGVkX2F0..."

# With filtering
tdb tenant documents list users \
  --api-key $API_KEY \
  --filter '{"where":{"and":[{"status":{"eq":"active"}}]}}'

# Sort by multiple fields
tdb tenant documents list orders \
  --api-key $API_KEY \
  --sort created_at:desc \
  --sort total:desc
```

---

### `tdb tenant documents get`

Get a specific document by ID.

**Usage:**
```bash
tdb tenant documents get COLLECTION DOCUMENT_ID --api-key KEY
```

**Examples:**
```bash
# Get document
tdb tenant documents get users user-123 --api-key $API_KEY

# Extract specific field
tdb tenant documents get users user-123 --api-key $API_KEY | jq '.email'

# Check if document exists
if tdb tenant documents get users user-123 --api-key $API_KEY >/dev/null 2>&1; then
  echo "Document exists"
fi
```

---

### `tdb tenant documents create`

Create a new document.

**Usage:**
```bash
tdb tenant documents create COLLECTION --data JSON --api-key KEY
```

**Flags:**
- `--data` - JSON document data (required)
- `--file` - Read data from file
- `--stdin` - Read from stdin

**Examples:**
```bash
# Create from inline JSON
tdb tenant documents create users \
  --data '{"name":"Alice","email":"alice@example.com"}' \
  --api-key $API_KEY

# From file
tdb tenant documents create users \
  --file user.json \
  --api-key $API_KEY

# From stdin
echo '{"name":"Bob","email":"bob@example.com"}' | \
  tdb tenant documents create users --stdin --api-key $API_KEY

# With specific ID (if primary key allows)
tdb tenant documents create products \
  --data '{"id":"SKU-001","name":"Widget","price":19.99}' \
  --api-key $API_KEY
```

---

### `tdb tenant documents update`

Replace a document completely.

**Usage:**
```bash
tdb tenant documents update COLLECTION DOCUMENT_ID --data JSON --api-key KEY
```

**Examples:**
```bash
# Update entire document
tdb tenant documents update users user-123 \
  --data '{"name":"Alice Updated","email":"alice.new@example.com","status":"active"}' \
  --api-key $API_KEY

# From file
tdb tenant documents update orders order-456 \
  --file order-updated.json \
  --api-key $API_KEY
```

---

### `tdb tenant documents patch`

Partially update a document (shallow merge).

**Usage:**
```bash
tdb tenant documents patch COLLECTION DOCUMENT_ID --data JSON --api-key KEY
```

**Examples:**
```bash
# Update single field
tdb tenant documents patch users user-123 \
  --data '{"status":"inactive"}' \
  --api-key $API_KEY

# Update multiple fields
tdb tenant documents patch products SKU-001 \
  --data '{"price":24.99,"stock":150}' \
  --api-key $API_KEY

# Add new field to existing document
tdb tenant documents patch users user-123 \
  --data '{"last_login":"2024-01-15T10:30:00Z"}' \
  --api-key $API_KEY
```

---

### `tdb tenant documents delete`

Delete a document (soft delete by default).

**Usage:**
```bash
tdb tenant documents delete COLLECTION DOCUMENT_ID --api-key KEY
```

**Flags:**
- `--purge` - Permanently delete (hard delete)
- `--force` - Skip confirmation

**Examples:**
```bash
# Soft delete (can be recovered)
tdb tenant documents delete users user-123 --api-key $API_KEY

# Permanent deletion
tdb tenant documents delete users user-123 \
  --api-key $API_KEY \
  --purge \
  --force

# Batch delete with loop
for id in user-{1..10}; do
  tdb tenant documents delete users $id --api-key $API_KEY --force
done
```

---

### `tdb tenant documents sync`

Bulk upsert documents from JSONL or JSON array.

**Usage:**
```bash
tdb tenant documents sync COLLECTION --file FILE --api-key KEY
```

**Flags:**
- `--file` - Input file (JSONL or JSON array)
- `--stdin` - Read from stdin
- `--mode` - Sync mode: patch, update, create (default: patch)
- `--skip-missing` - Only update existing documents

**Examples:**
```bash
# JSONL format (recommended for large files)
cat > users.jsonl <<EOF
{"id":"1","name":"Alice","email":"alice@example.com"}
{"id":"2","name":"Bob","email":"bob@example.com"}
{"id":"3","name":"Charlie","email":"charlie@example.com"}
EOF

tdb tenant documents sync users \
  --file users.jsonl \
  --mode patch \
  --api-key $API_KEY

# JSON array format
tdb tenant documents sync products \
  --file products.json \
  --mode update \
  --api-key $API_KEY

# Only update existing (don't create new)
tdb tenant documents sync users \
  --file updates.jsonl \
  --mode patch \
  --skip-missing \
  --api-key $API_KEY

# From stdin
cat large-dataset.jsonl | \
  tdb tenant documents sync orders --stdin --api-key $API_KEY
```

---

## Queries

### `tdb tenant queries list`

List all saved queries.

**Usage:**
```bash
tdb tenant queries list --api-key KEY
```

**Examples:**
```bash
# List queries
tdb tenant queries list --api-key $API_KEY

# Extract query names
tdb tenant queries list --api-key $API_KEY | jq '.items[] | .name'
```

---

### `tdb tenant queries get`

Get and optionally execute a saved query.

**Usage:**
```bash
tdb tenant queries get QUERY_ID --api-key KEY
```

**Flags:**
- `--execute` - Execute the query and return results
- `--params` - Query parameters as JSON

**Examples:**
```bash
# Get query definition
tdb tenant queries get query-123 --api-key $API_KEY

# Execute query
tdb tenant queries get query-123 \
  --api-key $API_KEY \
  --execute

# Execute with parameters
tdb tenant queries get active-users-query \
  --api-key $API_KEY \
  --execute \
  --params '{"status":"active","min_age":18}'

# By name instead of ID
tdb tenant queries get "Monthly Sales Report" \
  --api-key $API_KEY \
  --execute
```

---

## Snapshots

For complete snapshot documentation, see [SNAPSHOT_CLI.md](SNAPSHOT_CLI.md).

### `tdb tenant snapshots list`

List all snapshots.

**Examples:**
```bash
# List all
tdb tenant snapshots list --api-key $API_KEY

# Filter by collection
tdb tenant snapshots list --collection users --api-key $API_KEY
```

---

### `tdb tenant snapshots create`

Create a backup snapshot.

**Examples:**
```bash
# Full backup
tdb tenant snapshots create \
  --collection users \
  --name "Daily backup" \
  --api-key $API_KEY

# Encrypted incremental
tdb tenant snapshots create \
  --collection orders \
  --name "Hourly inc" \
  --incremental \
  --parent-snapshot snap-123 \
  --encrypt \
  --api-key $API_KEY
```

---

### `tdb tenant snapshots restore`

Restore from a snapshot.

**Examples:**
```bash
# Restore to original collection
tdb tenant snapshots restore snap-123 --api-key $API_KEY

# Restore to different collection
tdb tenant snapshots restore snap-123 \
  --target-collection users-restored \
  --api-key $API_KEY
```

---

## Audit Logs

### `tdb tenant audit`

Inspect audit log entries for document operations.

**Usage:**
```bash
tdb tenant audit --api-key KEY
```

**Flags:**
- `--collection` - Filter by collection ID
- `--document-id` - Filter by document ID
- `--operation` - Filter by operation (create, update, delete, purge)
- `--actor` - Filter by actor (API key prefix)
- `--since` - Start time (RFC3339 or duration like "24h")
- `--until` - End time (RFC3339 or duration)
- `--limit` - Maximum results (default: 100)
- `--sort` - Sort field (default: created_at)
- `--raw` - Output raw JSON (compact)
- `--raw-pretty` - Output pretty JSON

**Examples:**
```bash
# Recent audit logs
tdb tenant audit --api-key $API_KEY

# Last 24 hours for a collection
tdb tenant audit \
  --collection users \
  --since 24h \
  --api-key $API_KEY

# Specific document history
tdb tenant audit \
  --collection users \
  --document-id user-123 \
  --api-key $API_KEY

# Only delete operations
tdb tenant audit \
  --operation delete \
  --since 7d \
  --api-key $API_KEY

# Filter by actor (API key)
tdb tenant audit \
  --actor tdb_prod_*** \
  --since 2024-01-01T00:00:00Z \
  --until 2024-01-31T23:59:59Z \
  --api-key $API_KEY

# Combined filters with pretty JSON
tdb tenant audit \
  --collection orders \
  --operation update \
  --since 48h \
  --limit 50 \
  --raw-pretty \
  --api-key $API_KEY

# Sort by oldest first
tdb tenant audit \
  --sort created_at:asc \
  --limit 10 \
  --api-key $API_KEY
```

---

## Environment Variables

You can use environment variables to avoid repeating flags:

```bash
# Set default API key
export TINYDB_API_KEY="tdb_your_key_here"

# Set custom endpoint
export TINYDB_ENDPOINT="http://localhost:8080"

# Now use commands without --api-key flag
tdb tenant collections list
tdb tenant documents create users --data '{"name":"Alice"}'
```

---

## Tips & Tricks

### Use jq for JSON Processing

```bash
# Extract specific fields
tdb tenant documents list users --api-key $API_KEY | \
  jq '.items[] | {name, email}'

# Filter results
tdb tenant documents list users --api-key $API_KEY | \
  jq '.items[] | select(.status == "active")'

# Count results
tdb tenant documents list users --api-key $API_KEY | \
  jq '.items | length'
```

### Scripting

```bash
#!/bin/bash
# Daily backup script

API_KEY="tdb_your_key"
DATE=$(date +%Y-%m-%d)

for collection in users products orders; do
  echo "Backing up $collection..."
  tdb tenant snapshots create \
    --collection $collection \
    --name "Daily backup $DATE" \
    --encrypt \
    --api-key $API_KEY
done

echo "All backups complete!"
```

### Error Handling

```bash
# Check command success
if tdb tenant documents create users --data '{"invalid"}' --api-key $API_KEY 2>/dev/null; then
  echo "Success"
else
  echo "Failed: $?"
fi

# Capture output
OUTPUT=$(tdb tenant collections get users --api-key $API_KEY 2>&1)
if [ $? -eq 0 ]; then
  echo "Collection exists"
  echo "$OUTPUT" | jq '.schema'
else
  echo "Collection not found"
fi
```

---

For more information, see:
- [Quick Start Guide](QUICKSTART.md)
- [Snapshot Management](SNAPSHOT_CLI.md)
- [Developer Guide](DEVELOPER_GUIDE.md)
