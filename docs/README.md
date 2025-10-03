# TinyDB CLI Documentation

Welcome to the TinyDB CLI documentation! This guide will help you get started with `tdb` and master its powerful features.

## 📚 Documentation Structure

### Getting Started
- **[Quickstart Guide](QUICKSTART.md)** - Get up and running in minutes
  - Installation and configuration
  - Common workflows
  - Command reference
  - Tips & tricks

### Feature Guides
- **[Snapshot Management](SNAPSHOT_CLI.md)** - Backup and restore your data
  - Full and incremental backups
  - Encryption and compression
  - Multi-storage support (local, S3)
  - Scheduled backups
  
- **[CLI Enhancements](CLI_ENHANCEMENTS.md)** - Comprehensive command documentation
  - All enhanced commands with examples
  - Common patterns and usage tips
  - Best practices

- **[Command Reference](COMMAND_REFERENCE.md)** - Complete command documentation
  - All commands with detailed examples
  - Flags and options reference
  - Environment variables
  - Scripting tips

### Reference
- **[Enhancement Summary](CLI_ENHANCEMENT_SUMMARY.md)** - Overview of CLI improvements
  - Before/after comparison
  - User benefits
  - Impact analysis

## 🚀 Quick Links

### For New Users
Start here if you're new to TinyDB CLI:

1. [Installation](QUICKSTART.md#installation)
2. [Configuration](QUICKSTART.md#configuration)
3. [Your First Collection](QUICKSTART.md#working-with-collections)
4. [Your First Document](QUICKSTART.md#working-with-documents)

### For Existing Users
Enhance your workflow:

- [Bulk Operations](QUICKSTART.md#bulk-operations)
- [Backup & Restore](SNAPSHOT_CLI.md)
- [Audit Logs](QUICKSTART.md#monitoring-and-audit-logs)
- [Advanced Queries](QUICKSTART.md#tips--tricks)

### For Developers
Contributing to TinyDB CLI:

- [CLI Enhancement Patterns](CLI_ENHANCEMENTS.md#common-patterns)
- [Testing Guide](../README.md#testing)
- [Build Instructions](../README.md#building)

## 🎯 Common Tasks

| Task | Command | Documentation |
|------|---------|---------------|
| List collections | `tdb tenant collections list` | [Collections Guide](QUICKSTART.md#working-with-collections) |
| Create document | `tdb tenant documents create` | [Documents Guide](QUICKSTART.md#working-with-documents) |
| Bulk import | `tdb tenant documents sync` | [Bulk Operations](QUICKSTART.md#bulk-operations) |
| Create backup | `tdb tenant snapshots create` | [Snapshot Guide](SNAPSHOT_CLI.md) |
| View audit logs | `tdb tenant audit` | [Audit Guide](QUICKSTART.md#monitoring-and-audit-logs) |
| Query documents | `tdb tenant queries get` | [Query Guide](QUICKSTART.md#tips--tricks) |

## 💡 Key Features

### Dynamic Schema Management
Create and modify collections on-the-fly without downtime.
```bash
tdb tenant collections create users \
  --schema '{"fields":{"email":{"type":"string","required":true}}}' \
  --primary-key email
```

### Real-Time Sync
Bulk import thousands of documents efficiently.
```bash
tdb tenant documents sync users \
  --file users.jsonl \
  --mode patch
```

### Backup & Restore
Protect your data with snapshots.
```bash
tdb tenant snapshots create \
  --name daily-backup \
  --encrypt
```

### Audit Trail
Track all document operations.
```bash
tdb tenant audit \
  --collection users \
  --since 2024-01-01
```

## 🆘 Getting Help

- **Command Help**: Run any command with `--help` flag
  ```bash
  tdb tenant documents create --help
  ```

- **Examples**: Each command includes practical examples
  ```bash
  tdb tenant collections --help
  # Shows 6 commands with multiple examples each
  ```

- **Common Issues**: Check [Tips & Tricks](QUICKSTART.md#tips--tricks)

## 🔗 Additional Resources

- [TinyDB Main Documentation](https://github.com/cubetiqlabs/tinydb)
- [REST API Reference](https://github.com/cubetiqlabs/tinydb#api-documentation)
- [Client SDKs](https://github.com/cubetiqlabs/tinydb/tree/main/clients)

## 📝 Documentation Updates

This documentation reflects the comprehensive CLI enhancements completed in 2025, including:
- ✅ 26 commands enhanced with detailed examples
- ✅ Snapshot management system
- ✅ Comprehensive help text for all operations
- ✅ Practical workflows and patterns

Last updated: January 2025
