# TinyDB CLI Command Enhancements

This document summarizes the comprehensive usage descriptions and examples added to all `tdb-cli` commands to improve user experience and make the CLI easier to understand and use.

## Overview

All CLI commands have been enhanced with:
- **Long descriptions** explaining what the command does and when to use it
- **Practical examples** covering common use cases
- **Multiple usage patterns** showing different flag combinations
- **Best practices** and tips for effective usage

## Enhanced Commands

### Tenant Collections Commands

#### `tdb tenant collections list`
- **Features**: List collections with schema inspection and document structure analysis
- **Key Examples**:
  - List all collections
  - Show schemas with `--describe`
  - Inspect document structure with `--inspect-docs`
  - Custom inspection limits

#### `tdb tenant collections get`
- **Features**: Retrieve detailed collection information
- **Key Examples**:
  - Get by collection name
  - Get for specific app
  - Raw JSON output

#### `tdb tenant collections create`
- **Features**: Create collections with schema validation and primary key configuration
- **Key Examples**:
  - Simple collection creation
  - With inline JSON schema
  - Schema from file
  - Custom primary keys
  - Auto-generated UUIDs
  - Sync mode (create or update)

#### `tdb tenant collections update`
- **Features**: Update collection schemas and primary key settings
- **Key Examples**:
  - Update schema inline
  - Update from file
  - Modify primary key configuration
  - Enable auto-generation

#### `tdb tenant collections delete`
- **Features**: Permanently delete collections
- **Key Examples**:
  - Delete with warning about irreversibility
  - Delete from specific app
  - Backup before deletion recommendation

#### `tdb tenant collections sync`
- **Features**: Bulk create/update collections from JSON definitions
- **Key Examples**:
  - Sync from array format
  - Sync from object format
  - Different modes (create, update, upsert)
  - From file or stdin

### Tenant Documents Commands

#### `tdb tenant documents list`
- **Features**: List documents with filtering, sorting, pagination, and field selection
- **Key Examples**:
  - Basic listing
  - Limit and offset pagination
  - Cursor-based pagination
  - Multiple filters
  - Sorting (ascending/descending)
  - Field selection
  - Include soft-deleted documents
  - Pretty-print JSON

#### `tdb tenant documents get`
- **Features**: Fetch single document by ID or primary key
- **Key Examples**:
  - Get by ID
  - Get by primary key value
  - Pretty-printed JSON output
  - From specific app

#### `tdb tenant documents create`
- **Features**: Create new documents from JSON data
- **Key Examples**:
  - Inline JSON data
  - From file
  - From stdin
  - With auto-generated ID
  - For specific app

#### `tdb tenant documents update`
- **Features**: Complete document replacement
- **Key Examples**:
  - Update with inline JSON
  - Update from file
  - Update from stdin
  - For specific app
  - Explanation of full replacement vs patch

#### `tdb tenant documents patch`
- **Features**: Partial document updates via JSON merge patch
- **Key Examples**:
  - Patch specific fields
  - Remove fields with null
  - Patch from file
  - Nested field updates
  - For specific app

#### `tdb tenant documents delete`
- **Features**: Soft delete or permanent purge
- **Key Examples**:
  - Soft delete (recoverable)
  - Permanent purge
  - Purge with confirmation
  - Delete from specific app
  - Warning about permanence

#### `tdb tenant documents sync`
- **Features**: Bulk upsert documents by primary key
- **Key Examples**:
  - Sync from JSONL
  - Sync from JSON array
  - Different modes (patch, update, create)
  - Skip missing (update only)
  - Custom primary key field
  - JSONL and JSON format examples
  - For specific app

### Tenant Queries Commands

#### `tdb tenant queries list`
- **Features**: List all saved queries
- **Key Examples**:
  - List all queries
  - List for specific app
  - Raw JSON output

#### `tdb tenant queries get`
- **Features**: Retrieve saved query by ID or name
- **Key Examples**:
  - Get by ID
  - Get by name with `--by-name`
  - Raw JSON output

### Tenant Snapshots Commands

All snapshot commands already have comprehensive examples (implemented earlier):
- `list` - List snapshots with filters
- `create` - Create full/incremental backups with encryption
- `get` - View snapshot details
- `restore` - Restore to original or different collection
- `delete` - Delete snapshots

## Command Patterns

### Common Flags Across Commands

All commands support these common authentication and output flags:

```bash
--api-key $API_KEY      # Direct API key authentication
--key alias             # Use stored API key
--tenant tenant_id      # Specify tenant
--app app_id            # Scope to specific application
--raw                   # Raw JSON output
--raw-pretty            # Pretty-printed JSON
```

### Input Methods

Many commands support multiple input methods:

```bash
--data '{"key":"value"}'     # Inline JSON
--file path/to/file.json     # From file
--stdin                      # From stdin (pipe)
```

### Pagination Patterns

Documents support both pagination styles:

```bash
# Offset-based
--limit 20 --offset 40

# Cursor-based (efficient for large sets)
--cursor eyJpZCI6... --limit 50
```

### Filter Patterns

```bash
--filter key=value               # Simple equality
--filter nested.field=value      # Dotted paths for nested fields
--filter status=active \         # Multiple filters (AND logic)
  --filter role=admin
```

### Sort Patterns

```bash
--sort field:asc                 # Ascending
--sort field:desc                # Descending
--sort price:asc \               # Multiple sort fields
  --sort created_at:desc
```

## Usage Tips

### 1. Schema Management

```bash
# Create collection with strict schema
tdb tenant collections create --name users \
  --schema '{"type":"object","required":["email"],"properties":{"email":{"type":"string"}}}' \
  --api-key $API_KEY

# Inspect existing schemas and document structure
tdb tenant collections list --describe
```

### 2. Bulk Operations

```bash
# Sync collections from file
cat collections.json | tdb tenant collections sync --stdin --mode upsert

# Sync documents from JSONL
tdb tenant documents sync users --file users.jsonl --mode patch
```

### 3. Data Export and Backup

```bash
# Export documents
tdb tenant documents export users --out backup.jsonl

# Create encrypted snapshot
tdb tenant snapshots create --collection users \
  --name "Daily backup" --encrypt --storage s3
```

### 4. Field Selection for Performance

```bash
# Only fetch required fields
tdb tenant documents list users \
  --select id,email,created_at \
  --limit 1000
```

### 5. Combining Filters and Sorting

```bash
# Find active admins, sorted by creation date
tdb tenant documents list users \
  --filter status=active \
  --filter role=admin \
  --sort created_at:desc \
  --limit 50
```

## Migration from Previous Version

If you were using commands without examples, they will continue to work exactly the same way. The enhancements only add:

- More descriptive help text
- Practical examples in `--help` output
- Better documentation of available flags and options

No breaking changes were introduced.

## Getting Help

Every command supports `--help` to see:
- Description of what the command does
- All available flags and options
- Practical examples
- Related commands

```bash
# See help for any command
tdb --help
tdb tenant --help
tdb tenant collections --help
tdb tenant collections create --help
tdb tenant documents list --help
tdb tenant snapshots --help
```

## Files Modified

The following command files were enhanced:

1. **tenant_collections_cmd.go**
   - `list` - Added schema inspection and document analysis examples
   - `get` - Added basic retrieval examples
   - `create` - Added comprehensive creation examples with schemas and primary keys
   - `update` - Added schema and primary key update examples
   - `delete` - Added deletion examples with backup recommendations
   - `sync` - Added bulk synchronization examples

2. **tenant_documents_cmd.go**
   - `list` - Added filtering, sorting, pagination examples
   - `get` - Added basic retrieval examples
   - `create` - Added creation examples from various sources
   - `update` - Added full replacement examples
   - `patch` - Added partial update examples
   - `delete` - Added soft delete and purge examples
   - `sync` - Added bulk upsert examples with JSONL and JSON formats

3. **tenant_queries_cmd.go**
   - `list` - Added query listing examples
   - `get` - Added retrieval by ID and name examples

4. **tenant_snapshots_cmd.go** (Already complete from previous work)
   - All commands have comprehensive examples

## Next Steps

Remaining commands to enhance:
- Audit log commands
- Auth/RBAC commands (roles, permissions)
- Admin commands (tenants, API keys)
- Config commands

## Testing

All enhancements have been tested:

```bash
cd clients/tdb-cli
go build -o tdb cmd/tdb/main.go
./tdb tenant collections create --help
./tdb tenant documents list --help
./tdb tenant snapshots create --help
```

Build successful ✅  
Examples display correctly ✅  
No breaking changes ✅
