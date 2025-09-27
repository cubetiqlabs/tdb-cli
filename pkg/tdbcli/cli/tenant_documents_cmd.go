package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	clientpkg "cubetiqlabs/tinydb/pkg/tdbcli/client"
)

func newTenantDocumentsListCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var limit int
	var offset int
	var includeDeleted bool
	var selectFields string
	var filters []string
	var raw bool
	var cursor string

	cmd := &cobra.Command{
		Use:   "list <collection>",
		Short: "List documents in a collection",
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
			params := clientpkg.ListDocumentsParams{
				AppID:          auth.appID,
				Limit:          limit,
				Offset:         offset,
				Cursor:         strings.TrimSpace(cursor),
				IncludeDeleted: includeDeleted,
				Filters:        make(map[string]string),
			}
			if trimmed := strings.TrimSpace(selectFields); trimmed != "" {
				params.SelectFields = splitCommaList(trimmed)
			}
			for _, f := range filters {
				parts := strings.SplitN(f, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid filter %q, expected key=value", f)
				}
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				if key == "" {
					return fmt.Errorf("filter key cannot be empty: %q", f)
				}
				params.Filters[key] = value
			}
			resp, err := tenantClient.ListDocuments(cmd.Context(), collection, params)
			if err != nil {
				return err
			}
			if raw {
				return printJSON(cmd, resp)
			}
			if len(resp.Items) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No documents found")
				return nil
			}
			rows := make([][]string, 0, len(resp.Items))
			for _, doc := range resp.Items {
				deleted := "-"
				if doc.DeletedAt != nil {
					deleted = formatRelativeTimePtr(doc.DeletedAt, "-")
				}
				rows = append(rows, []string{
					doc.ID,
					doc.Key,
					formatRelativeTime(doc.CreatedAt, "-"),
					formatRelativeTime(doc.UpdatedAt, "-"),
					deleted,
					summarizeJSON(doc.Data, 60),
				})
			}
			renderTable(cmd, []string{"ID", "KEY", "CREATED", "UPDATED", "DELETED", "DATA"}, rows)
			return nil
		},
	}

	auth.bindWithApp(cmd)
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of documents to return")
	cmd.Flags().IntVar(&offset, "offset", 0, "Offset for paginated results")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Cursor token for paginated results")
	cmd.Flags().BoolVar(&includeDeleted, "include-deleted", false, "Include soft-deleted documents")
	cmd.Flags().StringVar(&selectFields, "select", "", "Comma-separated list of fields to project")
	cmd.Flags().StringArrayVar(&filters, "filter", nil, "Filter predicate in the form field=value (repeatable)")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON response")

	return cmd
}

func newTenantDocumentsGetCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var raw bool
	cmd := &cobra.Command{
		Use:   "get <collection> <id>",
		Short: "Fetch a document by ID",
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
			if raw {
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
	return cmd
}

func newTenantDocumentsCreateCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var data string
	var file string
	var stdin bool
	var raw bool

	cmd := &cobra.Command{
		Use:   "create <collection>",
		Short: "Create a document",
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
			payload, err := readJSONPayload(cmd, data, file, stdin, false)
			if err != nil {
				return err
			}
			doc, err := tenantClient.CreateDocument(cmd.Context(), collection, payload, auth.appID)
			if err != nil {
				return err
			}
			if raw {
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

	return cmd
}

func newTenantDocumentsUpdateCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var data string
	var file string
	var stdin bool
	var raw bool

	cmd := &cobra.Command{
		Use:   "update <collection> <id>",
		Short: "Replace a document",
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
			payload, err := readJSONPayload(cmd, data, file, stdin, false)
			if err != nil {
				return err
			}
			doc, err := tenantClient.UpdateDocument(cmd.Context(), collection, id, payload, auth.appID)
			if err != nil {
				return err
			}
			if raw {
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

	return cmd
}

func newTenantDocumentsPatchCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var data string
	var file string
	var stdin bool
	var raw bool

	cmd := &cobra.Command{
		Use:   "patch <collection> <id>",
		Short: "Patch a document",
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
			payload, err := readJSONPayload(cmd, data, file, stdin, false)
			if err != nil {
				return err
			}
			doc, err := tenantClient.PatchDocument(cmd.Context(), collection, id, payload, auth.appID)
			if err != nil {
				return err
			}
			if raw {
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

	return cmd
}

func newTenantDocumentsDeleteCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var purge bool
	var confirm bool

	cmd := &cobra.Command{
		Use:   "delete <collection> <id>",
		Short: "Delete or purge a document",
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
			if raw {
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

	return cmd
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

func newTenantDocumentsExportCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var filters []string
	var selectFields string
	var includeDeleted bool
	var outPath string
	var format string
	var pretty bool
	var includeMeta bool
	var pageSize int

	cmd := &cobra.Command{
		Use:   "export <collection>",
		Short: "Export documents to stdout or a file",
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
			page := pageSize
			if page <= 0 {
				page = 100
			}
			mode := strings.ToLower(strings.TrimSpace(format))
			if mode == "" {
				mode = "jsonl"
			}
			if mode != "jsonl" && mode != "json" {
				return fmt.Errorf("unsupported format %q (choose json or jsonl)", mode)
			}
			filtersMap := make(map[string]string)
			for _, f := range filters {
				parts := strings.SplitN(f, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid filter %q, expected key=value", f)
				}
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				if key == "" {
					return fmt.Errorf("filter key cannot be empty: %q", f)
				}
				filtersMap[key] = value
			}
			selector := splitCommaList(selectFields)
			targetPath := strings.TrimSpace(outPath)
			var writer *bufio.Writer
			var file *os.File
			if targetPath != "" {
				cleanPath := filepath.Clean(targetPath)
				dir := filepath.Dir(cleanPath)
				if dir != "." && dir != "" {
					if err := os.MkdirAll(dir, 0o755); err != nil {
						return fmt.Errorf("create output directory: %w", err)
					}
				}
				file, err = os.Create(cleanPath)
				if err != nil {
					return err
				}
				writer = bufio.NewWriter(file)
				defer func() {
					_ = writer.Flush()
					_ = file.Close()
				}()
			} else {
				writer = bufio.NewWriter(cmd.OutOrStdout())
				defer writer.Flush()
			}
			jsonFormat := mode == "json"
			first := true
			if jsonFormat {
				if _, err := writer.WriteString("["); err != nil {
					return err
				}
				if pretty {
					if _, err := writer.WriteString("\n"); err != nil {
						return err
					}
				}
			}
			written := 0
			offset := 0
			for {
				params := clientpkg.ListDocumentsParams{
					AppID:          auth.appID,
					Limit:          page,
					Offset:         offset,
					IncludeDeleted: includeDeleted,
					Filters:        make(map[string]string, len(filtersMap)),
				}
				for k, v := range filtersMap {
					params.Filters[k] = v
				}
				if len(selector) > 0 {
					params.SelectFields = selector
				}
				resp, err := tenantClient.ListDocuments(cmd.Context(), collection, params)
				if err != nil {
					return err
				}
				if len(resp.Items) == 0 {
					break
				}
				for _, doc := range resp.Items {
					payload, err := buildExportPayload(doc, includeMeta, pretty)
					if err != nil {
						return fmt.Errorf("prepare document %s: %w", doc.ID, err)
					}
					if jsonFormat {
						if !first {
							if pretty {
								if _, err := writer.WriteString(",\n"); err != nil {
									return err
								}
							} else {
								if _, err := writer.WriteString(","); err != nil {
									return err
								}
							}
						} else {
							first = false
						}
						if _, err := writer.Write(payload); err != nil {
							return err
						}
						if pretty {
							if _, err := writer.WriteString("\n"); err != nil {
								return err
							}
						}
					} else {
						if _, err := writer.Write(payload); err != nil {
							return err
						}
						if _, err := writer.WriteString("\n"); err != nil {
							return err
						}
					}
					written++
				}
				offset += len(resp.Items)
				if len(resp.Items) < page {
					break
				}
			}
			if jsonFormat {
				if !first && pretty {
					if _, err := writer.WriteString("]\n"); err != nil {
						return err
					}
				} else {
					if _, err := writer.WriteString("]"); err != nil {
						return err
					}
					if pretty {
						if _, err := writer.WriteString("\n"); err != nil {
							return err
						}
					}
				}
			}
			statusOut := cmd.ErrOrStderr()
			if targetPath != "" {
				fmt.Fprintf(statusOut, "Exported %d documents to %s\n", written, targetPath)
			} else {
				fmt.Fprintf(statusOut, "Exported %d documents\n", written)
			}
			return nil
		},
	}
	auth.bindWithApp(cmd)
	cmd.Flags().StringArrayVar(&filters, "filter", nil, "Filter predicate in the form field=value (repeatable)")
	cmd.Flags().StringVar(&selectFields, "select", "", "Comma-separated list of fields to project")
	cmd.Flags().BoolVar(&includeDeleted, "include-deleted", false, "Include soft-deleted documents")
	cmd.Flags().StringVar(&outPath, "out", "", "Write output to the specified file (defaults to stdout)")
	cmd.Flags().StringVar(&format, "format", "jsonl", "Output format: jsonl or json")
	cmd.Flags().BoolVar(&pretty, "pretty", false, "Pretty-print JSON output")
	cmd.Flags().BoolVar(&includeMeta, "include-meta", false, "Include document metadata alongside payload data")
	cmd.Flags().IntVar(&pageSize, "page-size", 100, "Number of documents to fetch per page")
	return cmd
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
