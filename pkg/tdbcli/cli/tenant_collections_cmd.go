package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	clientpkg "cubetiqlabs/tinydb/pkg/tdbcli/client"
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
