package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/spf13/cobra"

	clientpkg "github.com/cubetiqlabs/tdb-cli/pkg/tdbcli/client"
)

func newTenantCollectionsListCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var raw bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List collections for a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}
			collections, err := tenantClient.ListCollections(cmd.Context(), auth.appID)
			if err != nil {
				return err
			}
			if raw {
				return printJSON(cmd, collections)
			}
			if len(collections) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No collections found")
				return nil
			}
			rows := make([][]string, 0, len(collections))
			for _, col := range collections {
				app := "-"
				if col.AppID != nil && strings.TrimSpace(*col.AppID) != "" {
					app = *col.AppID
				}
				rows = append(rows, []string{
					col.Name,
					app,
					summarizePrimaryKey(col.PrimaryKeyField, col.PrimaryKeyType, col.PrimaryKeyAuto),
					formatTime(col.CreatedAt),
					formatTime(col.UpdatedAt),
				})
			}
			renderTable(cmd, []string{"NAME", "APP", "PRIMARY KEY", "CREATED", "UPDATED"}, rows)
			return nil
		},
	}
	auth.bindWithApp(cmd)
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON response")
	return cmd
}

func newTenantCollectionsCountCommand(env *Environment) *cobra.Command {
	var auth authFlags
	cmd := &cobra.Command{
		Use:   "count",
		Short: "Count collections for a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}
			total, err := tenantClient.CountCollections(cmd.Context(), auth.appID)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Collections: %d\n", total)
			return nil
		},
	}
	auth.bindWithApp(cmd)
	return cmd
}

func newTenantCollectionsGetCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var raw bool
	cmd := &cobra.Command{
		Use:   "get <name>",
		Short: "Fetch a collection by name",
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
			col, err := tenantClient.GetCollection(cmd.Context(), strings.TrimSpace(args[0]), auth.appID)
			if err != nil {
				return err
			}
			if raw {
				return printJSON(cmd, col)
			}
			app := "-"
			if col.AppID != nil && strings.TrimSpace(*col.AppID) != "" {
				app = *col.AppID
			}
			fmt.Fprintf(cmd.OutOrStdout(), "NAME: %s\nID: %s\nAPP: %s\nPRIMARY KEY: %s\nCREATED: %s\nUPDATED: %s\n",
				col.Name,
				col.ID,
				app,
				summarizePrimaryKey(col.PrimaryKeyField, col.PrimaryKeyType, col.PrimaryKeyAuto),
				formatTime(col.CreatedAt),
				formatTime(col.UpdatedAt),
			)
			schema := strings.TrimSpace(col.SchemaJSON)
			if schema != "" {
				var pretty map[string]any
				if err := json.Unmarshal([]byte(schema), &pretty); err == nil {
					fmt.Fprintln(cmd.OutOrStdout(), "SCHEMA:")
					if err := printJSON(cmd, pretty); err != nil {
						return err
					}
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "SCHEMA: %s\n", schema)
				}
			}
			return nil
		},
	}
	auth.bindWithApp(cmd)
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON response")
	return cmd
}

func newTenantCollectionsCreateCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var name string
	var schema string
	var schemaFile string
	var pkField string
	var pkType string
	var pkAuto bool
	var sync bool
	var raw bool

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a collection",
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}
			nameTrim := strings.TrimSpace(name)
			if nameTrim == "" {
				return errors.New("--name is required")
			}
			schemaContent, err := resolveSchemaInput(schema, schemaFile)
			if err != nil {
				return err
			}
			req := clientpkg.CreateCollectionRequest{
				Name:   nameTrim,
				Schema: schemaContent,
				AppID:  strings.TrimSpace(auth.appID),
			}
			if pkFieldTrim := strings.TrimSpace(pkField); pkFieldTrim != "" {
				spec := &clientpkg.PrimaryKeySpec{Field: pkFieldTrim}
				if typeTrim := strings.TrimSpace(pkType); typeTrim != "" {
					spec.Type = typeTrim
				}
				if cmd.Flags().Lookup("primary-key-auto").Changed {
					auto := pkAuto
					spec.Auto = &auto
				}
				req.PrimaryKey = spec
			} else if strings.TrimSpace(pkType) != "" || cmd.Flags().Lookup("primary-key-auto").Changed {
				return errors.New("--primary-key-field is required when configuring a primary key")
			}
			if cmd.Flags().Lookup("sync").Changed && sync {
				req.Sync = boolPtr(true)
			}
			col, err := tenantClient.CreateCollection(cmd.Context(), req)
			if err != nil {
				return err
			}
			if raw {
				return printJSON(cmd, col)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created collection %s (%s)\n", col.Name, col.ID)
			return nil
		},
	}

	auth.bindWithApp(cmd)
	cmd.Flags().StringVar(&name, "name", "", "Collection name")
	cmd.Flags().StringVar(&schema, "schema", "", "Inline JSON schema string")
	cmd.Flags().StringVar(&schemaFile, "schema-file", "", "Path to JSON schema file")
	cmd.Flags().StringVar(&pkField, "primary-key-field", "", "Primary key field name")
	cmd.Flags().StringVar(&pkType, "primary-key-type", "", "Primary key data type")
	cmd.Flags().BoolVar(&pkAuto, "primary-key-auto", false, "Enable auto-increment primary key")
	cmd.Flags().BoolVar(&sync, "sync", false, "Update the collection if it already exists")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON response")

	return cmd
}

func newTenantCollectionsUpdateCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var schema string
	var schemaFile string
	var pkField string
	var pkType string
	var pkAuto bool
	var raw bool

	cmd := &cobra.Command{
		Use:   "update <name>",
		Short: "Update a collection's schema",
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
			name := strings.TrimSpace(args[0])
			if name == "" {
				return errors.New("collection name cannot be empty")
			}
			var schemaProvided bool
			if cmd.Flags().Lookup("schema").Changed || cmd.Flags().Lookup("schema-file").Changed {
				schemaProvided = true
			}
			schemaContent, err := resolveSchemaInput(schema, schemaFile)
			if err != nil {
				return err
			}
			if !schemaProvided && strings.TrimSpace(pkField) == "" && !cmd.Flags().Lookup("primary-key-auto").Changed && strings.TrimSpace(pkType) == "" {
				return errors.New("provide schema or primary key updates")
			}
			req := clientpkg.UpdateCollectionRequest{}
			if schemaProvided {
				req.Schema = schemaContent
			}
			if pkFieldTrim := strings.TrimSpace(pkField); pkFieldTrim != "" {
				spec := &clientpkg.PrimaryKeySpec{Field: pkFieldTrim}
				if typeTrim := strings.TrimSpace(pkType); typeTrim != "" {
					spec.Type = typeTrim
				}
				if cmd.Flags().Lookup("primary-key-auto").Changed {
					auto := pkAuto
					spec.Auto = &auto
				}
				req.PrimaryKey = spec
			} else if strings.TrimSpace(pkType) != "" || cmd.Flags().Lookup("primary-key-auto").Changed {
				return errors.New("--primary-key-field is required when configuring a primary key")
			}
			col, err := tenantClient.UpdateCollection(cmd.Context(), name, auth.appID, req)
			if err != nil {
				return err
			}
			if raw {
				return printJSON(cmd, col)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated collection %s\n", col.Name)
			return nil
		},
	}

	auth.bindWithApp(cmd)
	cmd.Flags().StringVar(&schema, "schema", "", "Inline JSON schema string")
	cmd.Flags().StringVar(&schemaFile, "schema-file", "", "Path to JSON schema file")
	cmd.Flags().StringVar(&pkField, "primary-key-field", "", "Primary key field name")
	cmd.Flags().StringVar(&pkType, "primary-key-type", "", "Primary key data type")
	cmd.Flags().BoolVar(&pkAuto, "primary-key-auto", false, "Enable auto-increment primary key")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON response")

	return cmd
}

func newTenantCollectionsDeleteCommand(env *Environment) *cobra.Command {
	var auth authFlags
	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a collection",
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
			name := strings.TrimSpace(args[0])
			if name == "" {
				return errors.New("collection name cannot be empty")
			}
			if err := tenantClient.DeleteCollection(cmd.Context(), name, auth.appID); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted collection %s\n", name)
			return nil
		},
	}
	auth.bindWithApp(cmd)
	return cmd
}

func newTenantCollectionsSyncCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var data string
	var file string
	var stdin bool

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync collections from JSON definitions (create or update)",
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}
			payload, err := readJSONPayload(cmd, data, file, stdin, false)
			if err != nil {
				return err
			}
			entries, err := decodeCollectionSyncPayload(payload)
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				return errors.New("no collections provided in payload")
			}
			var created, updated, unchanged, skipped, failed int
			appID := strings.TrimSpace(auth.appID)
			for _, entry := range entries {
				name := strings.TrimSpace(entry.Name)
				if name == "" {
					fmt.Fprintln(cmd.ErrOrStderr(), "Skipping collection with empty name in payload")
					skipped++
					continue
				}
				schemaStr, err := entry.schemaString()
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "Skipping %s: invalid schema: %v\n", name, err)
					skipped++
					continue
				}
				pkSpec := (*clientpkg.PrimaryKeySpec)(nil)
				if entry.PrimaryKey != nil {
					pkSpec = &clientpkg.PrimaryKeySpec{Field: strings.TrimSpace(entry.PrimaryKey.Field), Type: strings.TrimSpace(entry.PrimaryKey.Type)}
					if entry.PrimaryKey.Auto != nil {
						pkSpec.Auto = boolPtr(*entry.PrimaryKey.Auto)
					}
					if strings.TrimSpace(pkSpec.Field) == "" && strings.TrimSpace(pkSpec.Type) == "" && pkSpec.Auto == nil {
						pkSpec = nil
					}
				}
				createReq := clientpkg.CreateCollectionRequest{
					Name:       name,
					Schema:     schemaStr,
					AppID:      appID,
					PrimaryKey: pkSpec,
				}
				col, err := tenantClient.GetCollection(cmd.Context(), name, auth.appID)
				if err != nil {
					if !isNotFoundError(err) {
						fmt.Fprintf(cmd.ErrOrStderr(), "Failed to sync %s: %v\n", name, err)
						failed++
						continue
					}
					if strings.TrimSpace(createReq.Schema) == "" && createReq.PrimaryKey == nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "Skipping %s: nothing to create\n", name)
						skipped++
						continue
					}
					if _, err := tenantClient.CreateCollection(cmd.Context(), createReq); err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "Failed to create %s: %v\n", name, err)
						failed++
						continue
					}
					fmt.Fprintf(cmd.OutOrStdout(), "Synced collection %s (created)\n", name)
					created++
					continue
				}

				updateReq := clientpkg.UpdateCollectionRequest{}
				schemaProvided := len(entry.Schema) > 0 && strings.TrimSpace(schemaStr) != ""
				if schemaProvided {
					equal, cmpErr := jsonEquivalent(schemaStr, col.SchemaJSON)
					if cmpErr != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "Skipping %s: schema comparison failed: %v\n", name, cmpErr)
						failed++
						continue
					}
					if !equal {
						updateReq.Schema = schemaStr
					}
				}
				if pkSpec != nil && primaryKeyNeedsUpdate(pkSpec, col) {
					updateReq.PrimaryKey = pkSpec
				}
				if updateReq.Schema == "" && updateReq.PrimaryKey == nil {
					fmt.Fprintf(cmd.OutOrStdout(), "Synced collection %s (unchanged)\n", name)
					unchanged++
					continue
				}
				if _, err := tenantClient.UpdateCollection(cmd.Context(), name, auth.appID, updateReq); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "Failed to update %s: %v\n", name, err)
					failed++
					continue
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Synced collection %s (updated)\n", name)
				updated++
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "Collections synced: created %d, updated %d, unchanged %d, skipped %d, failed %d\n", created, updated, unchanged, skipped, failed)
			if failed > 0 {
				return fmt.Errorf("failed to sync %d collection(s)", failed)
			}
			return nil
		},
	}

	auth.bindWithApp(cmd)
	cmd.Flags().StringVar(&data, "data", "", "Inline JSON payload containing collection definitions")
	cmd.Flags().StringVar(&file, "file", "", "Path to JSON file containing collection definitions")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read collection definitions from stdin")
	return cmd
}

type collectionSyncPayload struct {
	Name       string                `json:"name"`
	Schema     json.RawMessage       `json:"schema"`
	PrimaryKey *collectionPrimaryKey `json:"primary_key"`
}

type collectionPrimaryKey struct {
	Field string `json:"field"`
	Type  string `json:"type"`
	Auto  *bool  `json:"auto"`
}

func (p *collectionSyncPayload) schemaString() (string, error) {
	if p == nil || len(p.Schema) == 0 {
		return "", nil
	}
	trimmed := bytes.TrimSpace(p.Schema)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return "", nil
	}
	if trimmed[0] == '"' {
		var s string
		if err := json.Unmarshal(trimmed, &s); err != nil {
			return "", err
		}
		return strings.TrimSpace(s), nil
	}
	return string(trimmed), nil
}

func decodeCollectionSyncPayload(raw []byte) ([]collectionSyncPayload, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, nil
	}
	var entries []collectionSyncPayload
	switch trimmed[0] {
	case '[':
		if err := json.Unmarshal(trimmed, &entries); err != nil {
			return nil, fmt.Errorf("decode collection array: %w", err)
		}
	case '{':
		var single collectionSyncPayload
		if err := json.Unmarshal(trimmed, &single); err == nil && (strings.TrimSpace(single.Name) != "" || len(single.Schema) > 0 || single.PrimaryKey != nil) {
			entries = append(entries, single)
			break
		}
		var keyed map[string]collectionSyncPayload
		if err := json.Unmarshal(trimmed, &keyed); err != nil {
			return nil, fmt.Errorf("decode collection map: %w", err)
		}
		for name, spec := range keyed {
			if strings.TrimSpace(spec.Name) == "" {
				spec.Name = name
			}
			entries = append(entries, spec)
		}
	default:
		return nil, fmt.Errorf("invalid collections payload: expected JSON object or array")
	}
	for i := range entries {
		entries[i].Name = strings.TrimSpace(entries[i].Name)
	}
	return entries, nil
}

func resolveSchemaInput(inline, filePath string) (string, error) {
	trimmedInline := strings.TrimSpace(inline)
	trimmedFile := strings.TrimSpace(filePath)
	if trimmedInline != "" && trimmedFile != "" {
		return "", errors.New("use either --schema or --schema-file")
	}
	if trimmedFile != "" {
		content, err := readFileContent(trimmedFile)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(content), nil
	}
	return trimmedInline, nil
}

func boolPtr(v bool) *bool {
	b := v
	return &b
}

func jsonEquivalent(a, b string) (bool, error) {
	if strings.TrimSpace(a) == "" && strings.TrimSpace(b) == "" {
		return true, nil
	}
	lhs, err := normalizeJSON(a)
	if err != nil {
		return false, err
	}
	rhs, err := normalizeJSON(b)
	if err != nil {
		return false, err
	}
	return reflect.DeepEqual(lhs, rhs), nil
}

func normalizeJSON(raw string) (interface{}, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	decoder := json.NewDecoder(strings.NewReader(trimmed))
	decoder.UseNumber()
	var value interface{}
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

func primaryKeyNeedsUpdate(spec *clientpkg.PrimaryKeySpec, col *clientpkg.Collection) bool {
	if spec == nil || col == nil {
		return false
	}
	if field := strings.TrimSpace(spec.Field); field != "" && !strings.EqualFold(field, strings.TrimSpace(col.PrimaryKeyField)) {
		return true
	}
	if typ := strings.TrimSpace(spec.Type); typ != "" && !strings.EqualFold(typ, strings.TrimSpace(col.PrimaryKeyType)) {
		return true
	}
	if spec.Auto != nil && *spec.Auto != col.PrimaryKeyAuto {
		return true
	}
	return false
}
