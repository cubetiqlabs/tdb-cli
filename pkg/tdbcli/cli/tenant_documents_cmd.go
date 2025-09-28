package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	clientpkg "github.com/cubetiqlabs/tdb-cli/pkg/tdbcli/client"
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
