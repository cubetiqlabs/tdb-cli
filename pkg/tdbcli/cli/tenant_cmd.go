package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	clientpkg "cubetiqlabs/tinydb/pkg/tdbcli/client"
	configpkg "cubetiqlabs/tinydb/pkg/tdbcli/config"
	versionpkg "cubetiqlabs/tinydb/pkg/tdbcli/version"
)

func registerTenantCommands(root *cobra.Command, env *Environment) {
	tenantCmd := &cobra.Command{
		Use:   "tenant",
		Short: "Tenant-scoped operations",
	}

	appsCmd := &cobra.Command{
		Use:   "apps",
		Short: "Manage applications for a tenant",
	}
	appsCmd.AddCommand(newTenantAppsListCommand(env))
	appsCmd.AddCommand(newTenantAppsCreateCommand(env))
	appsCmd.AddCommand(newTenantAppsGetCommand(env))

	tenantCmd.AddCommand(appsCmd)

	collectionsCmd := &cobra.Command{
		Use:   "collections",
		Short: "Manage collections for a tenant",
	}
	collectionsCmd.AddCommand(newTenantCollectionsListCommand(env))
	collectionsCmd.AddCommand(newTenantCollectionsGetCommand(env))
	collectionsCmd.AddCommand(newTenantCollectionsCreateCommand(env))
	collectionsCmd.AddCommand(newTenantCollectionsUpdateCommand(env))
	collectionsCmd.AddCommand(newTenantCollectionsDeleteCommand(env))
	collectionsCmd.AddCommand(newTenantCollectionsCountCommand(env))
	tenantCmd.AddCommand(collectionsCmd)

	documentsCmd := &cobra.Command{
		Use:   "documents",
		Short: "Manage collection documents",
	}
	documentsCmd.AddCommand(newTenantDocumentsListCommand(env))
	documentsCmd.AddCommand(newTenantDocumentsGetCommand(env))
	documentsCmd.AddCommand(newTenantDocumentsCreateCommand(env))
	documentsCmd.AddCommand(newTenantDocumentsUpdateCommand(env))
	documentsCmd.AddCommand(newTenantDocumentsPatchCommand(env))
	documentsCmd.AddCommand(newTenantDocumentsDeleteCommand(env))
	documentsCmd.AddCommand(newTenantDocumentsBulkCreateCommand(env))
	documentsCmd.AddCommand(newTenantDocumentsCountCommand(env))
	tenantCmd.AddCommand(documentsCmd)

	root.AddCommand(tenantCmd)
}

type authFlags struct {
	tenantID string
	keyAlias string
	apiKey   string
	appID    string
}

func (a *authFlags) bind(cmd *cobra.Command) {
	cmd.Flags().StringVar(&a.tenantID, "tenant", "", "Tenant ID (defaults to configured value)")
	cmd.Flags().StringVar(&a.keyAlias, "key", "", "Stored key alias to authenticate with")
	cmd.Flags().StringVar(&a.apiKey, "api-key", "", "Raw API key to authenticate with (overrides stored keys)")
}

func (a *authFlags) bindWithApp(cmd *cobra.Command) {
	a.bind(cmd)
	cmd.Flags().StringVar(&a.appID, "app-id", "", "Application ID to scope requests (defaults to stored key scope when available)")
}

func (a *authFlags) resolveTenantClient(env *Environment, cmd *cobra.Command) (*clientpkg.TenantClient, configpkg.APIKeyEntry, string, error) {
	tenantID := strings.TrimSpace(a.tenantID)
	if tenantID == "" {
		envCtx, err := requireEnvironment(env)
		if err != nil {
			return nil, configpkg.APIKeyEntry{}, "", err
		}
		tenantID = strings.TrimSpace(envCtx.Config.DefaultTenant)
	}
	if tenantID == "" {
		return nil, configpkg.APIKeyEntry{}, "", errors.New("--tenant is required (set a default via `tdb config set default-tenant <tenant_id>`)")
	}
	client, entry, err := tenantClientFromEnv(env, tenantID, strings.TrimSpace(a.keyAlias), strings.TrimSpace(a.apiKey))
	if err != nil {
		return nil, configpkg.APIKeyEntry{}, "", err
	}
	if strings.TrimSpace(a.appID) == "" {
		if trimmed := strings.TrimSpace(entry.AppID); trimmed != "" {
			a.appID = trimmed
			if cmd != nil {
				if flag := cmd.Flags().Lookup("app-id"); flag != nil && !flag.Changed {
					fmt.Fprintf(cmd.OutOrStdout(), "Using stored app scope %s\n", trimmed)
				}
			}
		}
	}
	a.tenantID = tenantID
	return client, entry, tenantID, nil
}

func newTenantAppsListCommand(env *Environment) *cobra.Command {
	var auth authFlags
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List applications for a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}
			apps, err := tenantClient.ListApplications(cmd.Context())
			if err != nil {
				return err
			}
			if len(apps) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No applications found")
				return nil
			}
			rows := make([][]string, 0, len(apps))
			for _, app := range apps {
				rows = append(rows, []string{app.ID, app.Name, app.Description, formatTime(app.CreatedAt)})
			}
			renderTable(cmd, []string{"ID", "NAME", "DESCRIPTION", "CREATED"}, rows)
			return nil
		},
	}
	auth.bind(cmd)
	return cmd
}

func newTenantAppsCreateCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var name string
	var description string
	var withKey bool
	var storeAlias string
	var setDefault bool
	var tenantLabel string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an application for a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" {
				return errors.New("--name is required")
			}
			tenantClient, _, resolvedTenantID, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}
			desc := strings.TrimSpace(description)
			if desc == "" {
				desc = versionpkg.DefaultApplicationDescription()
			}
			req := clientpkg.CreateApplicationRequest{
				Name:        strings.TrimSpace(name),
				Description: desc,
				WithAPIKey:  withKey,
			}
			app, generatedKey, err := tenantClient.CreateApplication(cmd.Context(), req)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created application %s (%s)\n", app.Name, app.ID)
			if generatedKey != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "Generated key: %s (prefix %s)\n", generatedKey.APIKey, generatedKey.Prefix)
				if strings.TrimSpace(storeAlias) != "" {
					entry := configpkg.APIKeyEntry{Key: generatedKey.APIKey, Prefix: generatedKey.Prefix, AppID: app.ID}
					if generatedKey.Description != nil {
						entry.Description = *generatedKey.Description
					}
					if err := storeAPIKey(envCtx, resolvedTenantID, storeAlias, entry, setDefault, strings.TrimSpace(tenantLabel)); err != nil {
						return fmt.Errorf("application created but failed to store key: %w", err)
					}
					fmt.Fprintf(cmd.OutOrStdout(), "Stored generated key as %s\n", storeAlias)
				}
			}
			tenantLabelTrim := strings.TrimSpace(tenantLabel)
			if tenantLabelTrim != "" && strings.TrimSpace(storeAlias) == "" {
				cfg := envCtx.Config
				tc := cfg.EnsureTenant(resolvedTenantID)
				tc.Name = tenantLabelTrim
				cfg.UpdateTenant(resolvedTenantID, tc)
				if err := envCtx.Save(); err != nil {
					return err
				}
			}
			return nil
		},
	}

	auth.bind(cmd)
	cmd.Flags().StringVar(&name, "name", "", "Application name")
	cmd.Flags().StringVar(&description, "description", "", "Application description (defaults to CLI identifier)")
	cmd.Flags().BoolVar(&withKey, "with-key", false, "Generate an API key for the application")
	cmd.Flags().StringVar(&storeAlias, "store-key-as", "", "Alias to store generated key in local config")
	cmd.Flags().BoolVar(&setDefault, "set-default", false, "Mark stored key as default for the tenant")
	cmd.Flags().StringVar(&tenantLabel, "tenant-name", "", "Optional friendly tenant name to persist in config")

	return cmd
}

func newTenantAppsGetCommand(env *Environment) *cobra.Command {
	var auth authFlags
	cmd := &cobra.Command{
		Use:   "get <app_id>",
		Short: "Fetch a single application",
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
			app, err := tenantClient.GetApplication(cmd.Context(), strings.TrimSpace(args[0]))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "ID: %s\nNAME: %s\nDESCRIPTION: %s\nCREATED: %s\nUPDATED: %s\n",
				app.ID, app.Name, app.Description, formatTime(app.CreatedAt), formatTime(app.UpdatedAt))
			return nil
		},
	}
	auth.bind(cmd)
	return cmd
}
