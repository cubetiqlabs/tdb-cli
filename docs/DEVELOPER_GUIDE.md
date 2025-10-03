# Developer Guide: TinyDB CLI

This guide helps developers understand the TinyDB CLI architecture, extend its functionality, and maintain code quality.

## 🏛️ Architecture Overview

### Component Layers

```
┌─────────────────────────────────────────┐
│         User Interface (CLI)            │
│    (Cobra commands in pkg/tdbcli/cli)   │
└─────────────┬───────────────────────────┘
              │
┌─────────────▼───────────────────────────┐
│      Command Layer (Cobra)              │
│  - Argument parsing                     │
│  - Flag validation                      │
│  - Help text generation                 │
│  - Example formatting                   │
└─────────────┬───────────────────────────┘
              │
┌─────────────▼───────────────────────────┐
│     Client Layer (HTTP)                 │
│  - API request construction             │
│  - Authentication (API keys)            │
│  - Response handling                    │
│  - Error mapping                        │
└─────────────┬───────────────────────────┘
              │
┌─────────────▼───────────────────────────┐
│      TinyDB REST API                    │
│  - Business logic                       │
│  - Data persistence                     │
│  - Multi-tenancy                        │
└─────────────────────────────────────────┘
```

### Directory Structure

```
tdb-cli/
├── cmd/tdb/main.go                 # Entry point, CLI root setup
├── pkg/tdbcli/
│   ├── cli/                        # Cobra command definitions
│   │   ├── root.go                 # Root command setup
│   │   ├── tenant_collections_cmd.go   # Collection management (6 commands)
│   │   ├── tenant_documents_cmd.go     # Document CRUD (7 commands)
│   │   ├── tenant_queries_cmd.go       # Saved queries (2 commands)
│   │   ├── tenant_audit_cmd.go         # Audit logs (1 command)
│   │   ├── tenant_snapshots_cmd.go     # Backup/restore (6 commands)
│   │   ├── admin_cmd.go                # Admin operations (2 commands)
│   │   └── config_cmd.go               # Configuration (2 commands)
│   ├── client/                     # HTTP client implementation
│   │   ├── client.go               # Core client with auth
│   │   ├── collections.go          # Collection API calls
│   │   ├── documents.go            # Document API calls
│   │   ├── queries.go              # Query API calls
│   │   ├── audit.go                # Audit API calls
│   │   ├── snapshots.go            # Snapshot API calls
│   │   └── types.go                # Shared types
│   └── config/                     # Configuration management
│       ├── config.go               # Config file handling
│       └── store.go                # API key storage
├── docs/                           # Documentation
│   ├── README.md                   # Docs index
│   ├── QUICKSTART.md               # User getting started
│   ├── CONTRIBUTING.md             # Contribution guide
│   ├── DEVELOPER_GUIDE.md          # This file
│   ├── CLI_ENHANCEMENTS.md         # Enhancement details
│   ├── CLI_ENHANCEMENT_SUMMARY.md  # Enhancement summary
│   └── SNAPSHOT_CLI.md             # Snapshot feature docs
└── scripts/                        # Build/deployment scripts
    ├── install.sh                  # Unix install script
    └── install.ps1                 # Windows install script
```

## 🔧 Key Technologies

### Cobra CLI Framework
- **Command structure**: Hierarchical commands with subcommands
- **Flag parsing**: Automatic type conversion and validation
- **Help generation**: Auto-generated help text from metadata
- **Aliases**: Command shortcuts

### HTTP Client
- **Standard library**: `net/http` for requests
- **Context support**: Timeout and cancellation
- **Error handling**: Structured error responses
- **Authentication**: API key in headers

### Configuration
- **YAML format**: Human-readable config files
- **Environment variables**: Override config with env vars
- **Secure storage**: API keys stored safely

## 📚 Command Development Lifecycle

### 1. Design Phase

**Identify the need:**
- What user problem does this solve?
- What API endpoint does it map to?
- What are the required and optional parameters?

**Plan the interface:**
```
tdb <resource> <action> [arguments] [flags]

Examples:
- tdb tenant collections create <name> --schema <json>
- tdb tenant documents list <collection> --limit 100
- tdb tenant snapshots create --name <name> --encrypt
```

### 2. Implementation Phase

**A. Create the command structure**

```go
// pkg/tdbcli/cli/tenant_feature_cmd.go
package cli

import (
    "context"
    "encoding/json"
    "fmt"
    "os"

    "github.com/spf13/cobra"
    "github.com/cubetiqlabs/tdb-cli/pkg/tdbcli/client"
)

var (
    // Command-specific flags
    featureParam1 string
    featureParam2 int
    featureParam3 bool
)

func init() {
    // Register with parent command
    tenantCmd.AddCommand(newFeatureCmd())
}

func newFeatureCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "feature",
        Short: "Manage feature resources",
        Long:  `Detailed description...`,
    }

    // Add subcommands
    cmd.AddCommand(newFeatureListCmd())
    cmd.AddCommand(newFeatureCreateCmd())
    // ... more subcommands

    return cmd
}

func newFeatureListCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "list",
        Short: "List all features",
        Long: `List all feature resources in your tenant.

This command retrieves all features and displays them in a structured format.
You can filter, sort, and limit the results using optional flags.`,
        Example: `  # List all features
  tdb tenant feature list --api-key $API_KEY

  # Limit to 10 results
  tdb tenant feature list --limit 10 --api-key $API_KEY

  # Output as JSON
  tdb tenant feature list --output json --api-key $API_KEY`,
        RunE: func(cmd *cobra.Command, args []string) error {
            return runFeatureList(cmd, args)
        },
    }

    // Add flags
    cmd.Flags().StringVar(&apiKey, "api-key", "", "API key for authentication (required)")
    cmd.MarkFlagRequired("api-key")
    cmd.Flags().IntVar(&limit, "limit", 50, "Maximum number of results")
    cmd.Flags().StringVar(&output, "output", "", "Output format (json)")

    return cmd
}

func runFeatureList(cmd *cobra.Command, args []string) error {
    ctx := context.Background()

    // Create client
    c, err := client.NewClient(endpoint, apiKey, nil)
    if err != nil {
        return fmt.Errorf("failed to create client: %w", err)
    }

    // Make API call
    features, err := c.ListFeatures(ctx, limit)
    if err != nil {
        return fmt.Errorf("failed to list features: %w", err)
    }

    // Output results
    if output == "json" {
        return json.NewEncoder(os.Stdout).Encode(features)
    }

    // Human-readable output
    fmt.Printf("Found %d features:\n", len(features.Items))
    for _, f := range features.Items {
        fmt.Printf("  - %s (ID: %s)\n", f.Name, f.ID)
    }

    return nil
}
```

**B. Implement the client method**

```go
// pkg/tdbcli/client/feature.go
package client

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
)

// Feature represents a feature resource
type Feature struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    Status    string    `json:"status"`
    CreatedAt string    `json:"created_at"`
}

// FeatureListResponse represents the API response
type FeatureListResponse struct {
    Items []Feature `json:"items"`
    Total int       `json:"total"`
}

// ListFeatures retrieves all features with pagination
func (c *Client) ListFeatures(ctx context.Context, limit int) (*FeatureListResponse, error) {
    url := fmt.Sprintf("%s/api/features?limit=%d", c.baseURL, limit)

    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }

    var resp FeatureListResponse
    if err := c.doRequest(req, &resp); err != nil {
        return nil, fmt.Errorf("execute request: %w", err)
    }

    return &resp, nil
}
```

### 3. Testing Phase

**A. Unit tests for client**

```go
// pkg/tdbcli/client/feature_test.go
package client

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestListFeatures(t *testing.T) {
    // Create mock server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify request
        assert.Equal(t, "GET", r.Method)
        assert.Equal(t, "/api/features", r.URL.Path)
        assert.Equal(t, "test-key", r.Header.Get("X-API-Key"))

        // Return mock response
        resp := FeatureListResponse{
            Items: []Feature{
                {ID: "1", Name: "Feature 1", Status: "active"},
                {ID: "2", Name: "Feature 2", Status: "inactive"},
            },
            Total: 2,
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(resp)
    }))
    defer server.Close()

    // Create client
    client := NewClient(server.URL, "test-key", nil)

    // Test
    result, err := client.ListFeatures(context.Background(), 50)
    require.NoError(t, err)
    assert.Len(t, result.Items, 2)
    assert.Equal(t, "Feature 1", result.Items[0].Name)
}

func TestListFeatures_Error(t *testing.T) {
    // Create error server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusUnauthorized)
        json.NewEncoder(w).Encode(map[string]string{
            "error": "invalid_api_key",
        })
    }))
    defer server.Close()

    client := NewClient(server.URL, "bad-key", nil)

    _, err := client.ListFeatures(context.Background(), 50)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "invalid_api_key")
}
```

**B. Integration tests (optional)**

```go
// pkg/tdbcli/cli/feature_integration_test.go
// +build integration

package cli

import (
    "os"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestFeatureList_Integration(t *testing.T) {
    // Requires TinyDB server running
    apiKey := os.Getenv("TINYDB_TEST_API_KEY")
    if apiKey == "" {
        t.Skip("TINYDB_TEST_API_KEY not set")
    }

    // Test actual command execution
    // ...
}
```

### 4. Documentation Phase

**A. Update command help**
- Ensure `Long` describes functionality clearly
- Add at least 3-7 practical examples
- Include edge cases and common pitfalls

**B. Update documentation files**
- Add to QUICKSTART.md if user-facing
- Update CLI_ENHANCEMENTS.md with details
- Update README.md command reference

## 🎯 Development Patterns

### Error Handling

```go
// Pattern 1: Wrap errors with context
if err := someOperation(); err != nil {
    return fmt.Errorf("failed to perform operation: %w", err)
}

// Pattern 2: User-friendly messages
if err := c.CreateCollection(ctx, name, schema); err != nil {
    if strings.Contains(err.Error(), "already_exists") {
        return fmt.Errorf("collection %q already exists", name)
    }
    return fmt.Errorf("failed to create collection: %w", err)
}

// Pattern 3: Validation errors
if name == "" {
    return fmt.Errorf("collection name is required")
}
if !isValidName(name) {
    return fmt.Errorf("invalid collection name %q: must be alphanumeric", name)
}
```

### Output Formatting

```go
// Pattern 1: JSON output option
if output == "json" {
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    return encoder.Encode(result)
}

// Pattern 2: Table output
fmt.Printf("%-20s %-36s %-10s\n", "NAME", "ID", "STATUS")
fmt.Println(strings.Repeat("-", 70))
for _, item := range items {
    fmt.Printf("%-20s %-36s %-10s\n", item.Name, item.ID, item.Status)
}

// Pattern 3: Detailed single-item output
fmt.Printf("Collection: %s\n", col.Name)
fmt.Printf("ID:         %s\n", col.ID)
fmt.Printf("Schema:     %s\n", prettyJSON(col.Schema))
fmt.Printf("Created:    %s\n", col.CreatedAt)
```

### Flag Management

```go
// Pattern 1: Required flags
cmd.Flags().StringVar(&apiKey, "api-key", "", "API key (required)")
cmd.MarkFlagRequired("api-key")

// Pattern 2: Optional with defaults
cmd.Flags().IntVar(&limit, "limit", 50, "Result limit (default 50)")
cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json")

// Pattern 3: Boolean flags
cmd.Flags().BoolVar(&verbose, "verbose", false, "Verbose output")
cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Simulate without making changes")

// Pattern 4: Short flags (use sparingly)
cmd.Flags().StringVarP(&file, "file", "f", "", "Input file")
cmd.Flags().StringVarP(&output, "output", "o", "", "Output file")
```

## 🧰 Useful Tools

### Development Tools

```bash
# Build
go build -o tdb cmd/tdb/main.go

# Build with version info
go build -ldflags "-X main.version=1.0.0" -o tdb cmd/tdb/main.go

# Run tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run integration tests
go test -tags=integration ./...

# Lint
golangci-lint run

# Format
go fmt ./...
goimports -w .
```

### Debugging

```bash
# Enable verbose logging
export TINYDB_DEBUG=1
./tdb tenant collections list --api-key $API_KEY

# Test with custom endpoint
./tdb tenant collections list \
  --endpoint http://localhost:8080 \
  --api-key $API_KEY

# Use jq for JSON inspection
./tdb tenant documents list mycol --api-key $API_KEY | jq '.items[0]'
```

## 📊 Performance Considerations

### Pagination

```go
// Implement pagination for large datasets
func (c *Client) ListAllFeatures(ctx context.Context) ([]Feature, error) {
    var allFeatures []Feature
    limit := 100
    offset := 0

    for {
        resp, err := c.ListFeatures(ctx, limit, offset)
        if err != nil {
            return nil, err
        }

        allFeatures = append(allFeatures, resp.Items...)

        if len(resp.Items) < limit {
            break
        }

        offset += limit
    }

    return allFeatures, nil
}
```

### Bulk Operations

```go
// Batch requests for efficiency
func (c *Client) BulkCreate(ctx context.Context, items []Item) error {
    const batchSize = 100

    for i := 0; i < len(items); i += batchSize {
        end := i + batchSize
        if end > len(items) {
            end = len(items)
        }

        batch := items[i:end]
        if err := c.createBatch(ctx, batch); err != nil {
            return fmt.Errorf("failed at batch %d: %w", i/batchSize, err)
        }
    }

    return nil
}
```

## 🔒 Security Best Practices

### API Key Handling

```go
// Never log API keys
log.Printf("Request to %s with key %s", url, apiKey) // ❌ BAD

// Redact sensitive data
log.Printf("Request to %s with key ***", url) // ✅ GOOD

// Or use first few characters
redacted := apiKey[:8] + "..." // ✅ GOOD
log.Printf("Request to %s with key %s", url, redacted)
```

### Input Validation

```go
// Validate user input
func validateCollectionName(name string) error {
    if len(name) == 0 {
        return fmt.Errorf("name cannot be empty")
    }
    if len(name) > 64 {
        return fmt.Errorf("name too long (max 64 characters)")
    }
    if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(name) {
        return fmt.Errorf("name must be alphanumeric with _ or -")
    }
    return nil
}
```

## 📚 Additional Resources

- [Cobra Documentation](https://github.com/spf13/cobra)
- [TinyDB API Reference](../../README.md)
- [Go HTTP Client Best Practices](https://golang.org/pkg/net/http/)
- [Testing in Go](https://golang.org/pkg/testing/)

## 🤝 Need Help?

- Check existing commands for patterns
- Review [CONTRIBUTING.md](CONTRIBUTING.md)
- Open a discussion for design questions
- Create an issue for bugs

Happy coding! 🚀
