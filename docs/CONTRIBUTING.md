# Contributing to TinyDB CLI

Thank you for your interest in contributing to TinyDB CLI! This guide will help you add new features and maintain consistency with existing patterns.

## 🏗️ CLI Architecture

```
tdb-cli/
├── cmd/tdb/           # Main entry point
│   └── main.go        # CLI initialization
├── pkg/tdbcli/
│   ├── cli/           # Command definitions (Cobra)
│   │   ├── tenant_collections_cmd.go
│   │   ├── tenant_documents_cmd.go
│   │   ├── tenant_queries_cmd.go
│   │   ├── tenant_audit_cmd.go
│   │   ├── tenant_snapshots_cmd.go
│   │   ├── admin_cmd.go
│   │   └── config_cmd.go
│   └── client/        # API client logic
│       ├── client.go
│       ├── collections.go
│       ├── documents.go
│       ├── queries.go
│       ├── audit.go
│       └── snapshots.go
└── docs/              # Documentation
```

## 📝 Adding a New Command

### Step 1: Define the Command Structure

Follow this proven pattern for consistency:

```go
cmd := &cobra.Command{
    Use:   "command-name [args]",
    Short: "Brief one-line description",
    Long: `Detailed multi-paragraph explanation of what this command does.

Include:
- What problem it solves
- How it works
- Key concepts
- Important notes or warnings`,
    Example: `  # Example 1: Basic usage
  tdb command-name --flag value

  # Example 2: Common pattern
  tdb command-name --flag1 value1 --flag2 value2

  # Example 3: Advanced usage
  tdb command-name --all-flags with-context
  
  # Example 4: Edge case or special scenario
  tdb command-name --special-case`,
    Args: cobra.ExactArgs(1), // or NoArgs, MinimumNArgs(n), etc.
    RunE: func(cmd *cobra.Command, args []string) error {
        // Implementation
        return nil
    },
}
```

### Step 2: Add Flags

Use consistent flag naming and descriptions:

```go
// Required flags
cmd.Flags().StringVar(&apiKey, "api-key", "", "API key for authentication (required)")
cmd.MarkFlagRequired("api-key")

// Optional flags with sensible defaults
cmd.Flags().StringVar(&endpoint, "endpoint", "https://api.tinydb.com", "TinyDB API endpoint")
cmd.Flags().IntVar(&limit, "limit", 50, "Maximum number of results")

// Boolean flags
cmd.Flags().BoolVar(&verbose, "verbose", false, "Enable verbose output")

// File flags
cmd.Flags().StringVarP(&file, "file", "f", "", "Input file path")
```

### Step 3: Write Quality Examples

Include at least 3-7 examples covering:

1. **Basic usage** - Simplest form
2. **Common pattern** - Typical use case
3. **With filters** - Using query/filter options
4. **With output options** - JSON formatting, output files
5. **Advanced scenario** - Complex real-world usage
6. **Piping/scripting** - Shell integration
7. **Edge cases** - Error handling, special cases

Example template:

```go
Example: `  # Basic: List all items
  tdb tenant items list --api-key $API_KEY

  # Filter by status
  tdb tenant items list --status active --api-key $API_KEY

  # Limit results and output to file
  tdb tenant items list --limit 100 --output items.json --api-key $API_KEY

  # Use with jq for processing
  tdb tenant items list --api-key $API_KEY | jq '.items[] | select(.priority == "high")'

  # Combine multiple filters
  tdb tenant items list --status active --created-after 2024-01-01 --api-key $API_KEY

  # Verbose mode for debugging
  tdb tenant items list --verbose --api-key $API_KEY`,
```

### Step 4: Implement the Client Method

In `pkg/tdbcli/client/`:

```go
// ListItems retrieves items with optional filters
func (c *Client) ListItems(ctx context.Context, filters map[string]string) (*ItemsResponse, error) {
    // Build URL with query params
    u, _ := url.Parse(c.baseURL + "/api/items")
    q := u.Query()
    for k, v := range filters {
        q.Set(k, v)
    }
    u.RawQuery = q.Encode()

    // Make request
    req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
    if err != nil {
        return nil, err
    }

    var resp ItemsResponse
    if err := c.doRequest(req, &resp); err != nil {
        return nil, err
    }

    return &resp, nil
}
```

### Step 5: Add Tests

Create tests in `pkg/tdbcli/client/`:

```go
func TestListItems(t *testing.T) {
    // Setup mock server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify request
        assert.Equal(t, "GET", r.Method)
        assert.Equal(t, "/api/items", r.URL.Path)
        
        // Return mock response
        json.NewEncoder(w).Encode(ItemsResponse{
            Items: []Item{{ID: "1", Name: "test"}},
        })
    }))
    defer server.Close()

    client := NewClient(server.URL, "test-key", nil)
    resp, err := client.ListItems(context.Background(), nil)
    
    assert.NoError(t, err)
    assert.Len(t, resp.Items, 1)
}
```

## 🎨 Documentation Standards

### Long Description Format

```go
Long: `Brief opening sentence explaining the command.

Longer explanation paragraph that provides context. Explain what problem
this solves and when users should use it.

Key points:
- Important detail 1
- Important detail 2
- Important detail 3

Notes:
- Any warnings or gotchas
- Performance considerations
- Related commands`,
```

### Example Format Rules

1. **Use comments** - Each example should have a descriptive comment
2. **Show real values** - Use realistic data, not "foo" and "bar"
3. **Environment variables** - Use `$API_KEY` for sensitive values
4. **Progressive complexity** - Start simple, get more complex
5. **Include output** - When helpful, show expected results
6. **Shell integration** - Show piping with `jq`, `grep`, etc.

## 🧪 Testing Guidelines

### Before Submitting

1. **Build successfully**
   ```bash
   go build -o tdb cmd/tdb/main.go
   ```

2. **Run tests**
   ```bash
   go test ./...
   ```

3. **Verify help text**
   ```bash
   ./tdb your-command --help
   ```

4. **Test actual usage** (against dev server)
   ```bash
   ./tdb your-command --api-key $DEV_API_KEY
   ```

### Test Coverage

Aim for:
- Unit tests for client methods
- Integration tests for commands (where applicable)
- Error case coverage
- Edge case handling

## 📋 Pull Request Checklist

- [ ] Command follows the standard pattern
- [ ] Long description is clear and comprehensive
- [ ] At least 3 examples provided
- [ ] Client method implemented with error handling
- [ ] Tests added with good coverage
- [ ] Help text verified (`--help` output)
- [ ] All tests pass (`go test ./...`)
- [ ] Build succeeds (`go build`)
- [ ] Documentation updated if needed

## 🔍 Code Review Focus Areas

Reviewers will check:

1. **Consistency** - Follows established patterns
2. **Examples** - Practical and diverse
3. **Error handling** - Graceful failures with helpful messages
4. **Documentation** - Clear, accurate, complete
5. **Testing** - Adequate coverage
6. **User experience** - Intuitive and helpful

## 💡 Best Practices

### Error Messages

```go
// Good: Helpful, actionable
return fmt.Errorf("failed to create collection %q: %w (check schema syntax)", name, err)

// Bad: Vague, not actionable
return fmt.Errorf("error: %v", err)
```

### Flag Naming

- Use kebab-case: `--created-after`, not `--createdAfter`
- Be descriptive: `--collection`, not `--col`
- Match API parameters when possible
- Use short flags sparingly (`-f`, `-o`)

### Output Formatting

```go
// Support JSON output
if outputJSON {
    return json.NewEncoder(os.Stdout).Encode(result)
}

// Human-readable by default
fmt.Printf("Created collection: %s\n", result.Name)
fmt.Printf("ID: %s\n", result.ID)
```

### Progress Indicators

```go
// For long operations
spinner := NewSpinner("Creating snapshot...")
defer spinner.Stop()

// Or progress bars for known work
bar := NewProgressBar(totalItems)
```

## 🌟 Examples of Excellence

See these commands for reference:

- `tenant_snapshots_cmd.go` - Comprehensive snapshot management
- `tenant_documents_cmd.go` - Complex CRUD with bulk operations
- `tenant_collections_cmd.go` - Schema management patterns
- `tenant_audit_cmd.go` - Filtering and time range queries

## 🤝 Getting Help

- Questions? Open a discussion
- Bug reports? Create an issue
- Feature ideas? Start with a proposal

Thank you for contributing to TinyDB CLI! 🚀
