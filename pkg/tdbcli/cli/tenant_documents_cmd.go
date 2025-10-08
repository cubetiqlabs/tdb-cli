package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	clientpkg "github.com/cubetiqlabs/tdb-cli/pkg/tdbcli/client"
)

// Reconstructed missing commands (list/get) and cleaned export implementation.

func newTenantDocumentsGetCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var raw bool
	var rawPretty bool

	cmd := &cobra.Command{
		Use:   "get <collection> <id>",
		Short: "Get a document by ID",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}
			collection := strings.TrimSpace(args[0])
			id := strings.TrimSpace(args[1])
			if collection == "" || id == "" {
				return errors.New("collection and document ID are required")
			}
			doc, err := tenantClient.GetDocument(cmd.Context(), collection, id, auth.appID)
			if err != nil {
				return err
			}
			if raw || rawPretty {
				if rawPretty {
					return printJSON(cmd, makeDocumentPretty(*doc))
				}
				return printJSON(cmd, doc)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "ID: %s\nKEY: %s\nCOLLECTION: %s\nCREATED: %s\nUPDATED: %s\n",
				doc.ID,
				doc.Key,
				collection,
				formatTime(doc.CreatedAt),
				formatTime(doc.UpdatedAt),
			)
			if doc.DeletedAt != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "DELETED: %s\n", formatTime(*doc.DeletedAt))
			}
			fmt.Fprintln(cmd.OutOrStdout(), "DATA:")
			return printJSON(cmd, jsonStringToInterface(doc.Data))
		},
	}
	auth.bindWithApp(cmd)
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON response")
	cmd.Flags().BoolVar(&rawPretty, "raw-pretty", false, "Print pretty JSON response")
	return cmd
}

func newTenantDocumentsListCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var limit int
	var offset int
	var cursor string
	var includeDeleted bool
	var filters []string
	var selectFields string
	var selectOnly bool
	var sortFields string
	var raw bool
	var rawPretty bool

	cmd := &cobra.Command{
		Use:   "list <collection>",
		Short: "List documents in a collection",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil { return err }
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil { return err }
			collection := strings.TrimSpace(args[0])
			if collection == "" { return errors.New("collection name cannot be empty") }
			pageLimit := limit
			if pageLimit <= 0 { pageLimit = 50 }
			filterMap := map[string]string{}
			for _, f := range filters {
				parts := strings.SplitN(f, "=", 2)
				if len(parts) != 2 { return fmt.Errorf("invalid filter %q (expected key=value)", f) }
				k := strings.TrimSpace(parts[0])
				v := strings.TrimSpace(parts[1])
				if k == "" { return fmt.Errorf("filter key cannot be empty: %q", f) }
				filterMap[k] = v
			}
			params := clientpkg.ListDocumentsParams{AppID: auth.appID, Limit: pageLimit, Offset: offset, Cursor: strings.TrimSpace(cursor), IncludeDeleted: includeDeleted, Filters: filterMap}
			if trimmed := strings.TrimSpace(selectFields); trimmed != "" { params.SelectFields = splitCommaList(trimmed) }
			params.SelectOnly = selectOnly
			if trimmed := strings.TrimSpace(sortFields); trimmed != "" { sortTokens, err := normalizeDocumentSortTokens(splitCommaList(trimmed)); if err != nil { return err }; params.Sort = sortTokens }
			resp, err := tenantClient.ListDocuments(cmd.Context(), collection, params)
			if err != nil { return err }
			if raw || rawPretty {
				if rawPretty { return printJSON(cmd, resp) }
				return printJSON(cmd, resp)
			}
			if len(resp.Items) == 0 { fmt.Fprintln(cmd.OutOrStdout(), "No documents found"); return nil }
			rows := make([][]string, 0, len(resp.Items))
			for _, item := range resp.Items {
				rows = append(rows, []string{
					item.ID,
					item.Key,
					formatTime(item.CreatedAt),
					formatTime(item.UpdatedAt),
				})
			}
			renderTable(cmd, []string{"ID", "KEY", "CREATED", "UPDATED"}, rows)
			p := resp.Pagination
			fmt.Fprintf(cmd.OutOrStdout(), "COUNT: %d  LIMIT: %d  OFFSET: %d\n", p.Count, p.Limit, p.Offset)
			return nil
		},
	}
	auth.bindWithApp(cmd)
	cmd.Flags().IntVar(&limit, "limit", 50, "Maximum number of documents to return")
	cmd.Flags().IntVar(&offset, "offset", 0, "Offset for pagination")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Cursor for pagination")
	cmd.Flags().BoolVar(&includeDeleted, "include-deleted", false, "Include soft-deleted documents")
	cmd.Flags().StringArrayVar(&filters, "filter", nil, "Filter predicate field=value (repeatable)")
	cmd.Flags().StringVar(&selectFields, "select", "", "Comma-separated list of fields to project")
	cmd.Flags().BoolVar(&selectOnly, "select-only", false, "Restrict output to selected fields only (omit implicit metadata fields)")
	cmd.Flags().StringVar(&sortFields, "sort", "-created_at", "Comma-separated sort fields (prefix with - for descending)")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON response")
	cmd.Flags().BoolVar(&rawPretty, "raw-pretty", false, "Print pretty JSON response")
	return cmd
}

func newTenantDocumentsCreateCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var data string
	var file string
	var stdin bool
	var raw bool
	var rawPretty bool

	cmd := &cobra.Command{
		Use:   "create <collection>",
		Short: "Create a new document",
		Long: `Create a new document in a collection from JSON data.

The document data can be provided inline via --data, from a file via --file, or from stdin via --stdin.

If the collection has a primary key configured with auto-generation, the ID will be generated automatically. Otherwise, you must include the primary key field in the document data.`,
		Example: `  # Create from inline JSON
  tdb tenant documents create users \
    --data '{"email":"user@example.com","name":"John Doe"}' \
    --api-key $API_KEY

  # Create from file
  tdb tenant documents create products --file product.json --api-key $API_KEY

  # Create from stdin
  echo '{"title":"New Post","content":"..."}' | \
    tdb tenant documents create posts --stdin --api-key $API_KEY

  # Create with auto-generated ID
  tdb tenant documents create events \
    --data '{"type":"click","timestamp":"2025-01-15T10:30:00Z"}' \
    --api-key $API_KEY

  # Create for a specific app
  tdb tenant documents create logs \
    --data '{"level":"info","message":"Server started"}' \
    --app app_123 \
    --api-key $API_KEY`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}
			collection := strings.TrimSpace(args[0])
			if collection == "" {
				return errors.New("collection name cannot be empty")
			}
			payload, err := readJSONPayload(cmd, data, file, stdin, false)
			if err != nil {
				return err
			}
			doc, err := tenantClient.CreateDocument(cmd.Context(), collection, payload, auth.appID)
			if err != nil {
				return err
			}
			if raw || rawPretty {
				if rawPretty {
					return printJSON(cmd, makeDocumentPretty(*doc))
				}
				return printJSON(cmd, doc)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created document %s\n", doc.ID)
			return nil
		},
	}

	auth.bindWithApp(cmd)
	cmd.Flags().StringVar(&data, "data", "", "Inline JSON payload")
	cmd.Flags().StringVar(&file, "file", "", "Path to JSON payload file")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read JSON payload from stdin")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON response")
	cmd.Flags().BoolVar(&rawPretty, "raw-pretty", false, "Print pretty JSON response")

	return cmd
}

func newTenantDocumentsUpdateCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var data string
	var file string
	var stdin bool
	var raw bool
	var rawPretty bool

	cmd := &cobra.Command{
		Use:   "update <collection> <id>",
		Short: "Replace a document",
		Long: `Completely replace a document with new data.

This performs a full replacement - all existing fields will be replaced with the new document data. Use the 'patch' command instead if you want to partially update specific fields.`,
		Example: `  # Update document with inline JSON
  tdb tenant documents update users user_123 \
    --data '{"email":"newemail@example.com","name":"Jane Doe","role":"admin"}' \
    --api-key $API_KEY

  # Update from file
  tdb tenant documents update products prod_456 --file product-update.json --api-key $API_KEY

  # Update from stdin
  cat user-data.json | tdb tenant documents update users user_789 --stdin --api-key $API_KEY

  # Update for a specific app
  tdb tenant documents update configs cfg_001 \
    --data '{"theme":"dark","language":"en"}' \
    --app app_123 \
    --api-key $API_KEY`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}
			collection := strings.TrimSpace(args[0])
			id := strings.TrimSpace(args[1])
			if collection == "" || id == "" {
				return errors.New("collection and document ID are required")
			}
			payload, err := readJSONPayload(cmd, data, file, stdin, false)
			if err != nil {
				return err
			}
			doc, err := tenantClient.UpdateDocument(cmd.Context(), collection, id, payload, auth.appID)
			if err != nil {
				return err
			}
			if raw || rawPretty {
				if rawPretty {
					return printJSON(cmd, makeDocumentPretty(*doc))
				}
				return printJSON(cmd, doc)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated document %s\n", doc.ID)
			return nil
		},
	}

	auth.bindWithApp(cmd)
	cmd.Flags().StringVar(&data, "data", "", "Inline JSON payload")
	cmd.Flags().StringVar(&file, "file", "", "Path to JSON payload file")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read JSON payload from stdin")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON response")
	cmd.Flags().BoolVar(&rawPretty, "raw-pretty", false, "Print pretty JSON response")

	return cmd
}

func newTenantDocumentsPatchCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var data string
	var file string
	var stdin bool
	var raw bool
	var rawPretty bool

	cmd := &cobra.Command{
		Use:   "patch <collection> <id>",
		Short: "Patch a document",
		Long: `Partially update a document by merging the provided changes with the existing document.

This performs a JSON merge patch operation - only the fields you specify will be updated, and existing fields not mentioned in the patch will remain unchanged. To remove a field, set its value to null.`,
		Example: `  # Patch specific fields
  tdb tenant documents patch users user_123 \
    --data '{"status":"active","last_login":"2025-01-15T10:00:00Z"}' \
    --api-key $API_KEY

  # Remove a field by setting it to null
  tdb tenant documents patch users user_456 \
    --data '{"temp_field":null}' \
    --api-key $API_KEY

  # Patch from file
  tdb tenant documents patch products prod_789 --file changes.json --api-key $API_KEY

  # Patch nested fields
  tdb tenant documents patch users user_001 \
    --data '{"preferences":{"theme":"dark","notifications":true}}' \
    --api-key $API_KEY

  # Patch for a specific app
  tdb tenant documents patch settings cfg_001 \
    --data '{"enabled":true}' \
    --app app_123 \
    --api-key $API_KEY`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}
			collection := strings.TrimSpace(args[0])
			id := strings.TrimSpace(args[1])
			if collection == "" || id == "" {
				return errors.New("collection and document ID are required")
			}
			payload, err := readJSONPayload(cmd, data, file, stdin, false)
			if err != nil {
				return err
			}
			doc, err := tenantClient.PatchDocument(cmd.Context(), collection, id, payload, auth.appID)
			if err != nil {
				return err
			}
			if raw || rawPretty {
				if rawPretty {
					return printJSON(cmd, makeDocumentPretty(*doc))
				}
				return printJSON(cmd, doc)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Patched document %s\n", doc.ID)
			return nil
		},
	}

	auth.bindWithApp(cmd)
	cmd.Flags().StringVar(&data, "data", "", "Inline JSON payload")
	cmd.Flags().StringVar(&file, "file", "", "Path to JSON payload file")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read JSON payload from stdin")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON response")
	cmd.Flags().BoolVar(&rawPretty, "raw-pretty", false, "Print pretty JSON response")

	return cmd
}

func newTenantDocumentsDeleteCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var purge bool
	var confirm bool

	cmd := &cobra.Command{
		Use:   "delete <collection> <id>",
		Short: "Delete or purge a document",
		Long: `Delete a document (soft delete) or permanently purge it from the database.

By default, documents are soft-deleted and can be recovered. Use --purge to permanently remove the document from the database (this cannot be undone).

Soft-deleted documents can still be queried with the --include-deleted flag.`,
		Example: `  # Soft delete a document (can be recovered)
  tdb tenant documents delete users user_123 --api-key $API_KEY

  # Permanently purge a document (cannot be undone)
  tdb tenant documents delete users user_456 --purge --api-key $API_KEY

  # Purge with confirmation prompt
  tdb tenant documents delete orders order_789 --purge --confirm --api-key $API_KEY

  # Delete from a specific app
  tdb tenant documents delete logs log_001 --app app_123 --api-key $API_KEY`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}
			collection := strings.TrimSpace(args[0])
			id := strings.TrimSpace(args[1])
			if collection == "" || id == "" {
				return errors.New("collection and document ID are required")
			}
			if purge {
				if !confirm {
					return errors.New("use --confirm to acknowledge irreversible purge")
				}
				if err := tenantClient.PurgeDocument(cmd.Context(), collection, id, true, auth.appID); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Purged document %s\n", id)
				return nil
			}
			if err := tenantClient.DeleteDocument(cmd.Context(), collection, id, auth.appID); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted document %s\n", id)
			return nil
		},
	}

	auth.bindWithApp(cmd)
	cmd.Flags().BoolVar(&purge, "purge", false, "Permanently purge the document")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "Confirm irreversible purge")

	return cmd
}

func newTenantDocumentsBulkCreateCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var data string
	var file string
	var stdin bool
	var raw bool
	var rawPretty bool

	cmd := &cobra.Command{
		Use:   "bulk-create <collection>",
		Short: "Bulk insert documents",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}
			collection := strings.TrimSpace(args[0])
			if collection == "" {
				return errors.New("collection name cannot be empty")
			}
			payload, err := readJSONPayload(cmd, data, file, stdin, true)
			if err != nil {
				return err
			}
			resp, err := tenantClient.BulkCreateDocuments(cmd.Context(), collection, payload, auth.appID)
			if err != nil {
				return err
			}
			if raw || rawPretty {
				if rawPretty {
					payload := makeDocumentBulkPretty(resp)
					return printJSON(cmd, payload)
				}
				return printJSON(cmd, resp)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Inserted %d documents\n", len(resp.Items))
			return nil
		},
	}

	auth.bindWithApp(cmd)
	cmd.Flags().StringVar(&data, "data", "", "Inline JSON array payload")
	cmd.Flags().StringVar(&file, "file", "", "Path to JSON array payload file")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read JSON array payload from stdin")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON response")
	cmd.Flags().BoolVar(&rawPretty, "raw-pretty", false, "Print pretty JSON response")

	return cmd
}

func normalizeDocumentSortTokens(tokens []string) ([]string, error) {
	if len(tokens) == 0 {
		tokens = []string{"-created_at"}
	}
	expanded := make([]string, 0, len(tokens))
	for _, v := range tokens {
		parts := strings.Split(v, ",")
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				expanded = append(expanded, trimmed)
			}
		}
	}
	if len(expanded) == 0 {
		expanded = append(expanded, "-created_at")
	}
	allowed := map[string]struct{}{
		"created_at":  {},
		"updated_at":  {},
		"version":     {},
		"id":          {},
		"key":         {},
		"key_numeric": {},
		"deleted_at":  {},
	}
	result := make([]string, 0, len(expanded))
	for _, token := range expanded {
		desc := false
		field := token
		if strings.HasPrefix(field, "-") {
			desc = true
			field = field[1:]
		} else if strings.HasPrefix(field, "+") {
			field = field[1:]
		}
		field = strings.ToLower(strings.TrimSpace(field))
		if _, ok := allowed[field]; !ok || field == "" {
			return nil, fmt.Errorf("unsupported sort field %q", token)
		}
		if desc {
			result = append(result, "-"+field)
		} else {
			result = append(result, field)
		}
	}
	if len(result) == 0 {
		result = append(result, "-created_at")
	}
	return result, nil
}

func newTenantDocumentsCountCommand(env *Environment) *cobra.Command {
	var auth authFlags
	cmd := &cobra.Command{
		Use:   "count <collection>",
		Short: "Count documents in a collection",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}
			collection := strings.TrimSpace(args[0])
			if collection == "" {
				return errors.New("collection name cannot be empty")
			}
			count, err := tenantClient.CountDocuments(cmd.Context(), collection, auth.appID)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Documents: %d\n", count)
			return nil
		},
	}
	auth.bindWithApp(cmd)
	return cmd
}

// buildReportBody merges explicit body JSON (if any) with CLI flag derived groupBy / aggregates.
func buildReportBody(base map[string]any, groupBy []string, aggregates []string) map[string]any {
	if base == nil { base = map[string]any{} }
	if len(groupBy) > 0 {
		if _, ok := base["groupBy"]; !ok { base["groupBy"] = groupBy }
	}
	if len(aggregates) > 0 {
		var specs []map[string]any
		for _, spec := range aggregates {
			trim := strings.TrimSpace(spec)
			if trim == "" { continue }
			parts := strings.Split(trim, ":")
			var op, field, alias string
			distinct := false
			if len(parts) > 0 { op = strings.ToLower(strings.TrimSpace(parts[0])) }
			if len(parts) > 1 { field = strings.TrimSpace(parts[1]) }
			if len(parts) > 2 { alias = strings.TrimSpace(parts[2]) }
			if strings.HasSuffix(op, "!distinct") { op = strings.TrimSuffix(op, "!distinct"); distinct = true }
			if op == "" { continue }
			agg := map[string]any{"operation": op}
			if field != "" { agg["field"] = field }
			if alias != "" { agg["alias"] = alias }
			if distinct { agg["distinct"] = true }
			specs = append(specs, agg)
		}
		if len(specs) > 0 { if _, ok := base["aggregate"]; !ok { base["aggregate"] = specs } }
	}
	return base
}

// decideStreamingExport returns whether to stream given current flags & reasons for fallback.
func decideStreamingExport(requested bool, filters []string, includeDeleted bool, format string) (bool, string) {
	if !requested { return false, "" }
	if includeDeleted { return false, "include-deleted not supported in streaming" }
	if len(filters) > 0 { return false, "filters not supported in streaming" }
	if strings.ToLower(strings.TrimSpace(format)) == "json" { return false, "json array format not supported in streaming" }
	return true, ""
}

// aggregateSpec represents a parsed aggregate clause (internal to CLI helpers)
type aggregateSpecCLI struct {
	Operation string
	Field     string
	Alias     string
	Distinct  bool
}

// dedupeAggregateSpecs removes duplicate aggregate specs (same op, field, distinct) keeping the first occurrence.
// Alias differences do not create uniqueness; the earliest alias is preserved. Returns warnings for any removed duplicates.
func dedupeAggregateSpecs(specs []aggregateSpecCLI) ([]aggregateSpecCLI, []string) {
	seen := make(map[string]struct{})
	var out []aggregateSpecCLI
	var warnings []string
	for _, s := range specs {
		key := s.Operation + "|" + s.Field + "|" + strconv.FormatBool(s.Distinct)
		if _, exists := seen[key]; exists {
			// duplicate; emit warning referencing alias if present
			if s.Alias != "" {
				warnings = append(warnings, fmt.Sprintf("duplicate aggregate ignored (%s %s distinct=%v alias=%s)", s.Operation, s.Field, s.Distinct, s.Alias))
			} else {
				warnings = append(warnings, fmt.Sprintf("duplicate aggregate ignored (%s %s distinct=%v)", s.Operation, s.Field, s.Distinct))
			}
			continue
		}
		seen[key] = struct{}{}
		out = append(out, s)
	}
	return out, warnings
}

// parseAggregateSpecs converts user raw specs (op[:field][:alias][!distinct]) to structured specs, collecting warnings.
func parseAggregateSpecs(raw []string) ([]aggregateSpecCLI, []string) {
	var specs []aggregateSpecCLI
	var warnings []string
	for _, r := range raw {
		trim := strings.TrimSpace(r)
		if trim == "" { continue }
		parts := strings.Split(trim, ":")
		var op, field, alias string
		distinct := false
		if len(parts) > 0 { op = strings.ToLower(strings.TrimSpace(parts[0])) }
		if op == "" { warnings = append(warnings, "ignored empty operation") ; continue }
		if strings.HasSuffix(op, "!distinct") { op = strings.TrimSuffix(op, "!distinct"); distinct = true }
		if len(parts) > 1 { field = strings.TrimSpace(parts[1]) }
		if len(parts) > 2 { alias = strings.TrimSpace(parts[2]) }
		switch op { // basic validation; backend will enforce deeper rules
		case "count","sum","min","max","avg":
		default:
			warnings = append(warnings, fmt.Sprintf("unsupported aggregate op '%s'", op))
			continue
		}
		if op != "count" && field == "" { warnings = append(warnings, fmt.Sprintf("aggregate %s requires a field", op)); continue }
		specs = append(specs, aggregateSpecCLI{Operation: op, Field: field, Alias: alias, Distinct: distinct})
	}
	return specs, warnings
}

// expandAggregateSugar turns sugar flags into aggregateSpecCLI entries.
func expandAggregateSugar(count bool, countDistinct string, sums, mins, maxes, avgs []string) []aggregateSpecCLI {
	var specs []aggregateSpecCLI
	if count { specs = append(specs, aggregateSpecCLI{Operation: "count"}) }
	if cd := strings.TrimSpace(countDistinct); cd != "" { specs = append(specs, aggregateSpecCLI{Operation: "count", Field: cd, Distinct: true, Alias: "count_distinct_"+cd}) }
	for _, f := range sums { if t:=strings.TrimSpace(f); t!="" { specs = append(specs, aggregateSpecCLI{Operation:"sum", Field:t}) } }
	for _, f := range mins { if t:=strings.TrimSpace(f); t!="" { specs = append(specs, aggregateSpecCLI{Operation:"min", Field:t}) } }
	for _, f := range maxes { if t:=strings.TrimSpace(f); t!="" { specs = append(specs, aggregateSpecCLI{Operation:"max", Field:t}) } }
	for _, f := range avgs { if t:=strings.TrimSpace(f); t!="" { specs = append(specs, aggregateSpecCLI{Operation:"avg", Field:t}) } }
	return specs
}
func newTenantDocumentsReportCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var data string
	var file string
	var stdin bool
	var limit int
	var offset int
	var cursor string
	var selectFields string
	var selectOnly bool
	var groupBy string
	var aggregates []string
	// sugar flags
	var aggCount bool
	var aggCountDistinct string
	var aggSums []string
	var aggMins []string
	var aggMaxes []string
	var aggAvgs []string
	var raw bool
	var rawPretty bool

	cmd := &cobra.Command{
		Use:   "report <collection>",
		Short: "Run a report / analytics query for a collection",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}
			collection := strings.TrimSpace(args[0])
			if collection == "" {
				return errors.New("collection name cannot be empty")
			}

			var body map[string]any
			hasPayload := cmd.Flags().Lookup("data").Changed || cmd.Flags().Lookup("file").Changed || cmd.Flags().Lookup("stdin").Changed
			if hasPayload {
				payload, err := readJSONPayload(cmd, data, file, stdin, false)
				if err != nil {
					return err
				}
				if err := json.Unmarshal(payload, &body); err != nil {
					return fmt.Errorf("invalid report query payload: %w", err)
				}
			}
			if body == nil {
				body = make(map[string]any)
			}
			body["collection"] = collection

			params := clientpkg.ReportQueryParams{
				AppID:      auth.appID,
				Collection: collection,
				Limit:      limit,
				Offset:     offset,
				Cursor:     strings.TrimSpace(cursor),
				Body:       body,
			}
			if trimmed := strings.TrimSpace(selectFields); trimmed != "" {
				params.SelectFields = splitCommaList(trimmed)
			}
			params.SelectOnly = selectOnly
			if gb := strings.TrimSpace(groupBy); gb != "" {
				fields := splitCommaList(gb)
				if len(fields) > 0 {
					if _, ok := body["groupBy"]; !ok { body["groupBy"] = fields }
				}
			}

			// Parse explicit aggregate specs
			parsedExplicit, warnings := parseAggregateSpecs(aggregates)
			// Sugar expansions
			sugar := expandAggregateSugar(aggCount, aggCountDistinct, aggSums, aggMins, aggMaxes, aggAvgs)
			parsedAll := append(parsedExplicit, sugar...)
			// Dedupe (explicit specs take precedence because they appear first)
			parsedAll, dupWarnings := dedupeAggregateSpecs(parsedAll)
			if len(parsedAll) > 0 {
				var aggSpecs []map[string]any
				for _, s := range parsedAll {
					agg := map[string]any{"operation": s.Operation}
					if s.Field != "" { agg["field"] = s.Field }
					if s.Alias != "" { agg["alias"] = s.Alias }
					if s.Distinct { agg["distinct"] = true }
					aggSpecs = append(aggSpecs, agg)
				}
				if len(aggSpecs) > 0 { if _, ok := body["aggregate"]; !ok { body["aggregate"] = aggSpecs } }
			}
			for _, w := range warnings { fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", w) }
			for _, w := range dupWarnings { fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", w) }
			if limit > 0 || limit == -1 {
				if _, ok := body["limit"]; !ok {
					body["limit"] = limit
				}
			}
			if offset > 0 {
				if _, ok := body["offset"]; !ok {
					body["offset"] = offset
				}
			}
			if params.Cursor != "" {
				if _, ok := body["cursor"]; !ok {
					body["cursor"] = params.Cursor
				}
			}
			if len(params.SelectFields) > 0 {
				if _, ok := body["select"]; !ok {
					body["select"] = params.SelectFields
				}
			}
			if params.SelectOnly {
				if _, ok := body["selectOnly"]; !ok {
					body["selectOnly"] = true
				}
			}

			resp, err := tenantClient.ReportQuery(cmd.Context(), params)
			if err != nil {
				return err
			}
			if raw || rawPretty {
				if rawPretty {
					return printJSON(cmd, map[string]any{"data": resp.Data, "pagination": resp.Pagination})
				}
				return printJSON(cmd, resp)
			}
			result := &clientpkg.SavedQueryExecutionResult{Items: resp.Data}
			if err := renderSavedQueryResult(cmd, result); err != nil {
				return err
			}
			pagination := resp.Pagination
			fmt.Fprintf(cmd.OutOrStdout(), "TOTAL: %d  LIMIT: %d  OFFSET: %d\n", pagination.Total, pagination.Limit, pagination.Offset)
			if trimmed := strings.TrimSpace(pagination.NextCursor); trimmed != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "NEXT_CURSOR: %s\n", trimmed)
			}
			return nil
		},
	}

	auth.bindWithApp(cmd)
	cmd.Flags().StringVar(&data, "data", "", "Inline JSON report query payload")
	cmd.Flags().StringVar(&file, "file", "", "Path to report query JSON payload")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read report query JSON payload from stdin")
	cmd.Flags().IntVar(&limit, "limit", 0, "Override limit for the report query")
	cmd.Flags().IntVar(&offset, "offset", 0, "Override offset for the report query")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Cursor token for paginated reports")
	cmd.Flags().StringVar(&selectFields, "select", "", "Comma-separated list of fields to project")
	cmd.Flags().StringVar(&groupBy, "group-by", "", "Comma-separated list of fields to group by (report mode)")
	cmd.Flags().StringArrayVar(&aggregates, "aggregate", nil, "Aggregate spec op[:field][:alias][!distinct] (repeatable, e.g. --aggregate count --aggregate sum:price:total_sales)")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON response")
	cmd.Flags().BoolVar(&rawPretty, "raw-pretty", false, "Print pretty JSON response")
	// Sugar aggregate flags
	cmd.Flags().BoolVar(&aggCount, "count", false, "Add COUNT(*) aggregate")
	cmd.Flags().StringVar(&aggCountDistinct, "count-distinct", "", "Add COUNT(DISTINCT <field>) aggregate")
	cmd.Flags().StringArrayVar(&aggSums, "sum", nil, "Add SUM(field) aggregate (repeatable)")
	cmd.Flags().StringArrayVar(&aggMins, "min", nil, "Add MIN(field) aggregate (repeatable)")
	cmd.Flags().StringArrayVar(&aggMaxes, "max", nil, "Add MAX(field) aggregate (repeatable)")
	cmd.Flags().StringArrayVar(&aggAvgs, "avg", nil, "Add AVG(field) aggregate (repeatable)")

 	return cmd
}

func newTenantDocumentsExportCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var filters []string
	var selectFields string
	var selectOnly bool
	var includeDeleted bool
	var outPath string
	var format string
	var pretty bool
	var includeMeta bool
	var pageSize int
	var stream bool
	var cursor string

	cmd := &cobra.Command{
		Use:   "export <collection>",
		Short: "Export documents (supports streaming NDJSON)",
		Long: `Export documents from a collection.

Modes:
  - Paginated (default): uses ListDocuments API (supports filters, include-deleted, JSON array output)
  - Streaming (--stream): uses server NDJSON export endpoint for efficient full scans (no filters, jsonl only)

Examples:
  # Stream all documents as NDJSON
  tdb tenant documents export users --stream --api-key $API_KEY

  # Paginated export to a file (JSONL)
  tdb tenant documents export events --filter type=click --out events.jsonl --api-key $API_KEY

  # JSON array pretty output (paginated mode)
  tdb tenant documents export products --format json --pretty --api-key $API_KEY`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil { return err }
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil { return err }
			collection := strings.TrimSpace(args[0])
			if collection == "" { return errors.New("collection name cannot be empty") }

			mode := strings.ToLower(strings.TrimSpace(format))
			if mode == "" { mode = "jsonl" }
			if mode != "jsonl" && mode != "json" { return fmt.Errorf("unsupported format %q (choose json or jsonl)", mode) }

			// Decide streaming usage via helper
			if ok, reason := decideStreamingExport(stream, filters, includeDeleted, mode); stream && !ok {
				fmt.Fprintf(cmd.ErrOrStderr(), "Streaming disabled: %s; falling back to paginated export\n", reason)
				stream = false
			} else if stream && mode != "jsonl" { // defensive (helper already checks json format keyword only)
				fmt.Fprintln(cmd.ErrOrStderr(), "Streaming only supports jsonl format; falling back to paginated export")
				stream = false
			}

			selector := []string{}
			if trimmed := strings.TrimSpace(selectFields); trimmed != "" { selector = splitCommaList(trimmed) }

			// Streaming path
			if stream {
				body, headers, err := tenantClient.StreamExport(cmd.Context(), collection, selector, selectOnly, strings.TrimSpace(cursor), pageSize, auth.appID)
				if err != nil { return err }
				defer body.Close()
				var out *bufio.Writer
				var file *os.File
				if trimmed := strings.TrimSpace(outPath); trimmed != "" {
					clean := filepath.Clean(trimmed)
					if dir := filepath.Dir(clean); dir != "." && dir != "" { if err := os.MkdirAll(dir, 0o755); err != nil { return err } }
					file, err = os.Create(clean)
					if err != nil { return err }
					defer func(){ _ = file.Close() }()
					out = bufio.NewWriter(file)
					defer out.Flush()
				} else {
					out = bufio.NewWriter(cmd.OutOrStdout())
					defer out.Flush()
				}
				// Stream line by line to output; optionally transform if includeMeta false and line has 'data'.
				reader := bufio.NewReader(body)
				lines := 0
				for {
					line, readErr := reader.ReadBytes('\n')
					if len(line) > 0 {
						trim := bytes.TrimSpace(line)
						if len(trim) > 0 {
							if !includeMeta {
								var parsed map[string]any
								if json.Unmarshal(trim, &parsed) == nil {
									if dataVal, ok := parsed["data"]; ok {
										if pretty {
											b, _ := json.MarshalIndent(dataVal, "", "  ")
											trim = b
										} else {
											b, _ := json.Marshal(dataVal)
											trim = b
										}
									}
								}
							}
							if _, err := out.Write(trim); err != nil { return err }
							if _, err := out.WriteString("\n"); err != nil { return err }
							lines++
						}
					}
					if readErr != nil {
						if readErr == io.EOF { break }
						return readErr
					}
				}
				if next := headers.Get("X-Next-Cursor"); next != "" { fmt.Fprintf(cmd.ErrOrStderr(), "NEXT_CURSOR: %s\n", strings.TrimSpace(next)) }
				fmt.Fprintf(cmd.ErrOrStderr(), "Streamed %d documents\n", lines)
				return nil
			}

			// Paginated path
			page := pageSize
			if page <= 0 { page = 100 }
			filterMap := map[string]string{}
			for _, f := range filters {
				parts := strings.SplitN(f, "=", 2)
				if len(parts) != 2 { return fmt.Errorf("invalid filter %q (expected key=value)", f) }
				k := strings.TrimSpace(parts[0])
				v := strings.TrimSpace(parts[1])
				if k == "" { return fmt.Errorf("filter key cannot be empty: %q", f) }
				filterMap[k] = v
			}

			var out *bufio.Writer
			var file *os.File
			if trimmed := strings.TrimSpace(outPath); trimmed != "" {
				clean := filepath.Clean(trimmed)
				if dir := filepath.Dir(clean); dir != "." && dir != "" { if err := os.MkdirAll(dir, 0o755); err != nil { return err } }
				file, err = os.Create(clean)
				if err != nil { return err }
				defer func(){ _ = file.Close() }()
				out = bufio.NewWriter(file)
				defer out.Flush()
			} else {
				out = bufio.NewWriter(cmd.OutOrStdout())
				defer out.Flush()
			}

			jsonArray := mode == "json"
			if jsonArray {
				if _, err := out.WriteString("["); err != nil { return err }
				if pretty { if _, err := out.WriteString("\n"); err != nil { return err } }
			}
			written := 0
			offset := 0
			first := true
			for {
				params := clientpkg.ListDocumentsParams{AppID: auth.appID, Limit: page, Offset: offset, IncludeDeleted: includeDeleted, Filters: map[string]string{}}
				for k,v := range filterMap { params.Filters[k] = v }
				if len(selector) > 0 { params.SelectFields = selector }
				params.SelectOnly = selectOnly
				resp, err := tenantClient.ListDocuments(cmd.Context(), collection, params)
				if err != nil { return err }
				if len(resp.Items) == 0 { break }
				for _, doc := range resp.Items {
					payload, err := buildExportPayload(doc, includeMeta, pretty)
					if err != nil { return fmt.Errorf("prepare document %s: %w", doc.ID, err) }
					if jsonArray {
						if !first {
							if pretty { if _, err := out.WriteString(",\n"); err != nil { return err } } else { if _, err := out.WriteString(","); err != nil { return err } }
						} else { first = false }
						if _, err := out.Write(payload); err != nil { return err }
						if pretty { if _, err := out.WriteString("\n"); err != nil { return err } }
					} else {
						if _, err := out.Write(payload); err != nil { return err }
						if _, err := out.WriteString("\n"); err != nil { return err }
					}
					written++
				}
				offset += len(resp.Items)
				if len(resp.Items) < page { break }
			}
			if jsonArray {
				if _, err := out.WriteString("]"); err != nil { return err }
				if pretty { if _, err := out.WriteString("\n"); err != nil { return err } }
			}
			if trimmed := strings.TrimSpace(outPath); trimmed != "" { fmt.Fprintf(cmd.ErrOrStderr(), "Exported %d documents to %s\n", written, trimmed) } else { fmt.Fprintf(cmd.ErrOrStderr(), "Exported %d documents\n", written) }
			return nil
		},
	}
	auth.bindWithApp(cmd)
	cmd.Flags().StringArrayVar(&filters, "filter", nil, "Filter predicate field=value (repeatable; disables streaming)")
	cmd.Flags().StringVar(&selectFields, "select", "", "Comma-separated list of fields to project")
	cmd.Flags().BoolVar(&selectOnly, "select-only", false, "Restrict output to only selected fields (omit implicit metadata)")
	cmd.Flags().BoolVar(&includeDeleted, "include-deleted", false, "Include soft-deleted documents (disables streaming)")
	cmd.Flags().StringVar(&outPath, "out", "", "Write output to the specified file (defaults to stdout)")
	cmd.Flags().StringVar(&format, "format", "jsonl", "Output format: jsonl or json (array)")
	cmd.Flags().BoolVar(&pretty, "pretty", false, "Pretty-print JSON values")
	cmd.Flags().BoolVar(&includeMeta, "include-meta", false, "Include document metadata alongside payload data (paginated mode)")
	cmd.Flags().IntVar(&pageSize, "page-size", 100, "Page size for paginated mode or limit hint for streaming")
	cmd.Flags().BoolVar(&stream, "stream", false, "Use streaming NDJSON export (no filters, no include-deleted, jsonl only)")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Cursor for streaming continuation (X-Next-Cursor emitted to stderr)")
	return cmd
}

func newTenantDocumentsSyncCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var data string
	var file string
	var stdin bool
	var mode string
	var keyField string
	var skipMissing bool

	cmd := &cobra.Command{
		Use:   "sync <collection>",
		Short: "Sync documents by primary key from JSON payload (create or update)",
		Long: `Synchronize documents in a collection by upserting (create or update) based on primary key values.

Accepts JSONL (JSON Lines) or JSON array format. Each document must include its primary key field. Documents that don't exist will be created; existing documents will be updated based on the mode.

Modes:
  - patch: Merge changes with existing documents (default)
  - update: Completely replace existing documents
  - create: Only create new documents, skip existing ones

Use --skip-missing to only update existing documents without creating new ones.`,
		Example: `  # Sync from JSONL file (patch mode)
  tdb tenant documents sync users --file users.jsonl --api-key $API_KEY

  # Sync from JSON array (update mode - full replacement)
  tdb tenant documents sync products \
    --file products.json \
    --mode update \
    --api-key $API_KEY

  # Sync from stdin
  cat orders.jsonl | tdb tenant documents sync orders --stdin --api-key $API_KEY

  # Only update existing documents (skip creation)
  tdb tenant documents sync users \
    --file updates.jsonl \
    --skip-missing \
    --api-key $API_KEY

  # Sync with custom primary key field
  tdb tenant documents sync products \
    --file products.jsonl \
    --key-field sku \
    --api-key $API_KEY

  # Example JSONL format (users.jsonl):
  # {"email":"user1@example.com","name":"Alice","role":"admin"}
  # {"email":"user2@example.com","name":"Bob","role":"user"}

  # Example JSON array format (products.json):
  # [
  #   {"sku":"ABC-123","name":"Widget","price":29.99},
  #   {"sku":"XYZ-789","name":"Gadget","price":49.99"}
  # ]

  # For a specific app
  tdb tenant documents sync logs \
    --file logs.jsonl \
    --app app_123 \
    --api-key $API_KEY`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}
			collection := strings.TrimSpace(args[0])
			if collection == "" {
				return errors.New("collection name cannot be empty")
			}
			modeValue := strings.ToLower(strings.TrimSpace(mode))
			if modeValue == "" {
				modeValue = "patch"
			}
			if modeValue != "patch" && modeValue != "update" {
				return fmt.Errorf("unsupported mode %q (choose patch or update)", mode)
			}
			payload, err := readJSONPayload(cmd, data, file, stdin, false)
			if err != nil {
				return err
			}
			docs, err := decodeDocumentSyncPayload(payload)
			if err != nil {
				return err
			}
			if len(docs) == 0 {
				return errors.New("no documents provided in payload")
			}
			col, err := tenantClient.GetCollection(cmd.Context(), collection, auth.appID)
			if err != nil {
				return err
			}
			pkField := strings.TrimSpace(keyField)
			if pkField == "" {
				pkField = strings.TrimSpace(col.PrimaryKeyField)
				if pkField == "" {
					pkField = "id"
				}
			}
			pkType := strings.TrimSpace(col.PrimaryKeyType)
			if pkType == "" {
				pkType = "string"
			}
			keepPrimary := modeValue == "update"
			var created, updated, unchanged, skipped, missing, failed int
			for idx, rawDoc := range docs {
				keyValue, err := extractDocumentKey(rawDoc, pkField, pkType)
				if err != nil || strings.TrimSpace(keyValue) == "" {
					fmt.Fprintf(cmd.ErrOrStderr(), "[%d] skipping: %v\n", idx, firstNonNil(err, errors.New("missing primary key value")))
					skipped++
					continue
				}
				existing, err := tenantClient.GetDocumentByPrimaryKey(cmd.Context(), collection, keyValue, auth.appID)
				if err != nil {
					if isNotFoundError(err) {
						if skipMissing {
							fmt.Fprintf(cmd.ErrOrStderr(), "[%d] document %s not found; skipping\n", idx, keyValue)
							missing++
							continue
						}
						createPayload := prepareDocumentCreatePayload(rawDoc, pkField)
						encoded, err := json.Marshal(createPayload)
						if err != nil {
							fmt.Fprintf(cmd.ErrOrStderr(), "[%d] encode %s failed: %v\n", idx, keyValue, err)
							failed++
							continue
						}
						result, err := tenantClient.CreateDocument(cmd.Context(), collection, encoded, auth.appID)
						if err != nil {
							fmt.Fprintf(cmd.ErrOrStderr(), "[%d] create %s failed: %v\n", idx, keyValue, err)
							failed++
							continue
						}
						fmt.Fprintf(cmd.OutOrStdout(), "Synced document %s (created %s)\n", keyValue, formatRelativeTime(result.CreatedAt, "just now"))
						created++
						continue
					}
					fmt.Fprintf(cmd.ErrOrStderr(), "[%d] lookup %s failed: %v\n", idx, keyValue, err)
					failed++
					continue
				}
				payloadMap := prepareDocumentSyncPayload(rawDoc, pkField, keepPrimary)
				if len(payloadMap) == 0 {
					fmt.Fprintf(cmd.ErrOrStderr(), "[%d] document %s has no mutable fields; skipping\n", idx, keyValue)
					skipped++
					continue
				}
				skipUpdate, cmpErr := shouldSkipDocumentSync(existing.Data, payloadMap, pkField, keepPrimary, modeValue)
				if cmpErr != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "[%d] compare %s failed: %v\n", idx, keyValue, cmpErr)
				} else if skipUpdate {
					fmt.Fprintf(cmd.OutOrStdout(), "Synced document %s (unchanged)\n", keyValue)
					unchanged++
					continue
				}
				encoded, err := json.Marshal(payloadMap)
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "[%d] encode %s failed: %v\n", idx, keyValue, err)
					failed++
					continue
				}
				var result *clientpkg.Document
				if modeValue == "patch" {
					result, err = tenantClient.PatchDocument(cmd.Context(), collection, existing.ID, encoded, auth.appID)
				} else {
					result, err = tenantClient.UpdateDocument(cmd.Context(), collection, existing.ID, encoded, auth.appID)
				}
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "[%d] sync %s failed: %v\n", idx, keyValue, err)
					failed++
					continue
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Synced document %s (updated %s)\n", keyValue, formatRelativeTime(result.UpdatedAt, "just now"))
				updated++
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "Documents synced: created %d, updated %d, unchanged %d, skipped %d, missing %d, failed %d\n", created, updated, unchanged, skipped, missing, failed)
			if failed > 0 {
				return fmt.Errorf("failed to sync %d document(s)", failed)
			}
			return nil
		},
	}

	auth.bindWithApp(cmd)
	cmd.Flags().StringVar(&data, "data", "", "Inline JSON payload containing document data")
	cmd.Flags().StringVar(&file, "file", "", "Path to JSON file containing document data")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read document data from stdin")
	cmd.Flags().StringVar(&mode, "mode", "patch", "Sync mode: patch (default) or update")
	cmd.Flags().StringVar(&keyField, "key-field", "", "Override primary key field name used for matching")
	cmd.Flags().BoolVar(&skipMissing, "skip-missing", false, "Skip documents that are not found instead of creating them")
	return cmd
}

func decodeDocumentSyncPayload(raw []byte) ([]map[string]any, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, nil
	}
	var docs []map[string]any
	switch trimmed[0] {
	case '[':
		if err := json.Unmarshal(trimmed, &docs); err != nil {
			return nil, fmt.Errorf("decode document array: %w", err)
		}
	case '{':
		var single map[string]any
		if err := json.Unmarshal(trimmed, &single); err == nil {
			docs = append(docs, single)
			break
		}
		return nil, fmt.Errorf("invalid document payload: expected JSON object or array")
	default:
		return nil, fmt.Errorf("invalid document payload: expected JSON object or array")
	}
	return docs, nil
}

func extractDocumentKey(doc map[string]any, pkField, pkType string) (string, error) {
	candidates := []string{pkField, "key", "id"}
	for _, field := range candidates {
		if strings.TrimSpace(field) == "" {
			continue
		}
		if value, ok := doc[field]; ok {
			key, err := stringifyKey(value, pkType)
			if err != nil {
				return "", err
			}
			if key != "" {
				return key, nil
			}
		}
		// also consider lowercase key variations
		if value, ok := doc[strings.ToLower(field)]; ok {
			key, err := stringifyKey(value, pkType)
			if err != nil {
				return "", err
			}
			if key != "" {
				return key, nil
			}
		}
	}
	return "", fmt.Errorf("primary key field %s not present", pkField)
}

func stringifyKey(value interface{}, pkType string) (string, error) {
	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return "", fmt.Errorf("primary key cannot be empty")
		}
		return strings.TrimSpace(v), nil
	case float64:
		if pkType == "number" {
			return strconv.FormatInt(int64(v), 10), nil
		}
		return strings.TrimSpace(fmt.Sprintf("%v", v)), nil
	case json.Number:
		if pkType == "number" {
			i, err := v.Int64()
			if err != nil {
				return "", err
			}
			return strconv.FormatInt(i, 10), nil
		}
		return v.String(), nil
	case nil:
		return "", fmt.Errorf("primary key is nil")
	default:
		s := strings.TrimSpace(fmt.Sprintf("%v", v))
		if s == "" {
			return "", fmt.Errorf("primary key cannot be empty")
		}
		return s, nil
	}
}

func prepareDocumentSyncPayload(source map[string]any, pkField string, keepPrimary bool) map[string]any {
	cleaned := make(map[string]any)
	for key, value := range source {
		lower := strings.ToLower(key)
		if _, forbidden := documentSyncReservedFields[lower]; forbidden {
			continue
		}
		if !keepPrimary && strings.EqualFold(key, pkField) {
			continue
		}
		cleaned[key] = value
	}
	if keepPrimary {
		if value, ok := source[pkField]; ok {
			cleaned[pkField] = value
		} else if value, ok := source[strings.ToLower(pkField)]; ok {
			cleaned[pkField] = value
		}
	}
	return cleaned
}

func prepareDocumentCreatePayload(source map[string]any, pkField string) map[string]any {
	cleaned := make(map[string]any)
	for key, value := range source {
		lower := strings.ToLower(key)
		if _, forbidden := documentSyncReservedFields[lower]; forbidden && !strings.EqualFold(key, pkField) {
			continue
		}
		cleaned[key] = value
	}
	if _, ok := cleaned[pkField]; !ok {
		if value, ok := source[strings.ToLower(pkField)]; ok {
			cleaned[pkField] = value
		}
	}
	return cleaned
}

var documentSyncReservedFields = map[string]struct{}{
	"id":            {},
	"tenant_id":     {},
	"collection_id": {},
	"key":           {},
	"key_numeric":   {},
	"created_at":    {},
	"updated_at":    {},
	"deleted_at":    {},
}

func shouldSkipDocumentSync(existingJSON string, payload map[string]any, pkField string, keepPrimary bool, mode string) (bool, error) {
	if len(payload) == 0 {
		return false, nil
	}
	trimmed := strings.TrimSpace(existingJSON)
	if trimmed == "" {
		return false, nil
	}
	var current map[string]any
	if err := json.Unmarshal([]byte(trimmed), &current); err != nil {
		return false, fmt.Errorf("decode existing document: %w", err)
	}
	existingComparable := sanitizeDocumentComparisonMap(current, pkField, keepPrimary)
	payloadComparable := sanitizeDocumentComparisonMap(payload, pkField, keepPrimary)
	if strings.EqualFold(mode, "patch") {
		for key, newVal := range payloadComparable {
			existingVal, ok := existingComparable[key]
			if !ok {
				return false, nil
			}
			if !reflect.DeepEqual(existingVal, newVal) {
				return false, nil
			}
		}
		return true, nil
	}
	if len(existingComparable) != len(payloadComparable) {
		return false, nil
	}
	if reflect.DeepEqual(existingComparable, payloadComparable) {
		return true, nil
	}
	return false, nil
}

func sanitizeDocumentComparisonMap(source map[string]any, pkField string, keepPrimary bool) map[string]any {
	cleaned := make(map[string]any)
	if source == nil {
		return cleaned
	}
	pkFieldTrim := strings.TrimSpace(pkField)
	for key, value := range source {
		lower := strings.ToLower(key)
		if _, reserved := documentSyncReservedFields[lower]; reserved && !(keepPrimary && pkFieldTrim != "" && strings.EqualFold(key, pkFieldTrim)) {
			continue
		}
		if !keepPrimary && pkFieldTrim != "" && strings.EqualFold(key, pkFieldTrim) {
			continue
		}
		cleaned[key] = normalizeDocumentValue(value)
	}
	if keepPrimary && pkFieldTrim != "" {
		if _, ok := cleaned[pkFieldTrim]; !ok {
			if value, ok := source[pkFieldTrim]; ok {
				cleaned[pkFieldTrim] = normalizeDocumentValue(value)
			} else if value, ok := source[strings.ToLower(pkFieldTrim)]; ok {
				cleaned[pkFieldTrim] = normalizeDocumentValue(value)
			}
		}
	}
	return cleaned
}

func normalizeDocumentValue(value any) any {
	switch v := value.(type) {
	case map[string]interface{}:
		return normalizeDocumentMap(v)
	case []interface{}:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = normalizeDocumentValue(item)
		}
		return result
	case json.Number:
		if strings.Contains(v.String(), ".") {
			if f, err := v.Float64(); err == nil {
				return f
			}
		}
		if i, err := v.Int64(); err == nil {
			return float64(i)
		}
		return v.String()
	default:
		return v
	}
}

func normalizeDocumentMap(source map[string]interface{}) map[string]any {
	result := make(map[string]any, len(source))
	for key, value := range source {
		result[key] = normalizeDocumentValue(value)
	}
	return result
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "404")
}

func firstNonNil(err error, fallback error) error {
	if err != nil {
		return err
	}
	return fallback
}

func splitCommaList(value string) []string {
	parts := strings.Split(value, ",")
	var result []string
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func jsonStringToInterface(raw string) interface{} {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return map[string]interface{}{}
	}
	var anyValue interface{}
	if err := json.Unmarshal([]byte(trimmed), &anyValue); err == nil {
		return anyValue
	}
	return trimmed
}

func buildExportPayload(doc clientpkg.Document, includeMeta bool, pretty bool) ([]byte, error) {
	if includeMeta {
		payload := map[string]any{
			"id":            doc.ID,
			"tenant_id":     doc.TenantID,
			"collection_id": doc.CollectionID,
			"key":           doc.Key,
			"created_at":    doc.CreatedAt.Format(time.RFC3339Nano),
			"updated_at":    doc.UpdatedAt.Format(time.RFC3339Nano),
			"data":          jsonStringToInterface(doc.Data),
		}
		if doc.KeyNumeric != nil {
			payload["key_numeric"] = *doc.KeyNumeric
		}
		if doc.DeletedAt != nil {
			payload["deleted_at"] = doc.DeletedAt.Format(time.RFC3339Nano)
		}
		if pretty {
			return json.MarshalIndent(payload, "", "  ")
		}
		return json.Marshal(payload)
	}
	if !pretty {
		trimmed := strings.TrimSpace(doc.Data)
		if trimmed == "" {
			return []byte("null"), nil
		}
		return []byte(trimmed), nil
	}
	value := jsonStringToInterface(doc.Data)
	return json.MarshalIndent(value, "", "  ")
}
