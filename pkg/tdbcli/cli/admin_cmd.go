package cli

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	clientpkg "cubetiqlabs/tinydb/pkg/tdbcli/client"
	configpkg "cubetiqlabs/tinydb/pkg/tdbcli/config"
)

func registerAdminCommands(root *cobra.Command, env *Environment) {
	adminCmd := &cobra.Command{
		Use:   "admin",
		Short: "Administer tenants and API keys",
	}

	adminTenantsCmd := &cobra.Command{
		Use:   "tenants",
		Short: "Manage tenants",
	}
	adminTenantsCmd.AddCommand(newAdminTenantListCommand(env))
	adminTenantsCmd.AddCommand(newAdminTenantCreateCommand(env))

	adminKeysCmd := &cobra.Command{
		Use:   "keys",
		Short: "Manage tenant API keys",
	}
	adminKeysCmd.AddCommand(newAdminKeyListCommand(env))
	adminKeysCmd.AddCommand(newAdminKeyCreateCommand(env))
	adminKeysCmd.AddCommand(newAdminKeyRevokeCommand(env))

	adminCmd.AddCommand(adminTenantsCmd)
	adminCmd.AddCommand(adminKeysCmd)

	root.AddCommand(adminCmd)
}

func newAdminTenantListCommand(env *Environment) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all tenants",
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			client, err := adminClientFromEnv(envCtx)
			if err != nil {
				return err
			}
			tenants, err := client.ListTenants(cmd.Context())
			if err != nil {
				return err
			}
			rows := make([][]string, 0, len(tenants))
			for _, t := range tenants {
				rows = append(rows, []string{t.ID, t.Name, t.Description, formatTime(t.CreatedAt)})
			}
			if len(rows) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No tenants found")
				return nil
			}
			renderTable(cmd, []string{"ID", "NAME", "DESCRIPTION", "CREATED"}, rows)
			return nil
		},
	}
}

func newAdminTenantCreateCommand(env *Environment) *cobra.Command {
	var name string
	var description string
	var withKey bool
	var saveAlias string
	var setDefault bool
	var tenantLabel string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" {
				return errors.New("--name is required")
			}
			client, err := adminClientFromEnv(envCtx)
			if err != nil {
				return err
			}
			req := clientpkg.CreateTenantRequest{Name: strings.TrimSpace(name), Description: strings.TrimSpace(description), WithAPIKey: withKey}
			tenant, generatedKey, err := client.CreateTenant(cmd.Context(), req)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created tenant %s (%s)\n", tenant.Name, tenant.ID)
			if generatedKey != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "Generated key: %s (prefix %s)\n", generatedKey.APIKey, generatedKey.Prefix)
				if strings.TrimSpace(saveAlias) != "" {
					entry := configpkg.APIKeyEntry{Key: generatedKey.APIKey, Prefix: generatedKey.Prefix}
					if err := storeAPIKey(envCtx, tenant.ID, saveAlias, entry, setDefault, strings.TrimSpace(tenantLabel)); err != nil {
						return fmt.Errorf("tenant created but failed to store key: %w", err)
					}
					fmt.Fprintf(cmd.OutOrStdout(), "Stored generated key as %s\n", saveAlias)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Tenant name")
	cmd.Flags().StringVar(&description, "description", "", "Tenant description")
	cmd.Flags().BoolVar(&withKey, "with-key", false, "Generate an API key for the tenant")
	cmd.Flags().StringVar(&saveAlias, "save-key-as", "", "Alias to store generated key in local config")
	cmd.Flags().BoolVar(&setDefault, "set-default", false, "Mark stored key as default")
	cmd.Flags().StringVar(&tenantLabel, "tenant-name", "", "Optional friendly name to store with tenant config")

	return cmd
}

func newAdminKeyListCommand(env *Environment) *cobra.Command {
	var tenantID string
	var appID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List API keys for a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantIDTrim := strings.TrimSpace(tenantID)
			if tenantIDTrim == "" {
				return errors.New("--tenant is required")
			}
			client, err := adminClientFromEnv(envCtx)
			if err != nil {
				return err
			}
			var filter *string
			if strings.TrimSpace(appID) != "" {
				v := strings.TrimSpace(appID)
				filter = &v
			}
			keys, err := client.ListKeys(cmd.Context(), tenantIDTrim, filter)
			if err != nil {
				return err
			}
			if len(keys) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No keys found")
				return nil
			}
			rows := make([][]string, 0, len(keys))
			for _, k := range keys {
				scope := k.Scope
				if scope == "application" && k.AppID != nil {
					scope = scope + " (" + *k.AppID + ")"
				}
				revoked := ""
				if k.RevokedAt != nil {
					revoked = formatTime(*k.RevokedAt)
				}
				rows = append(rows, []string{k.Prefix, scope, optional(k.Description), formatTime(k.CreatedAt), revoked})
			}
			renderTable(cmd, []string{"PREFIX", "SCOPE", "DESCRIPTION", "CREATED", "REVOKED"}, rows)
			return nil
		},
	}

	cmd.Flags().StringVar(&tenantID, "tenant", "", "Tenant ID (required)")
	cmd.Flags().StringVar(&appID, "app-id", "", "Filter keys by application ID")

	return cmd
}

func newAdminKeyCreateCommand(env *Environment) *cobra.Command {
	var tenantID string
	var appID string
	var description string
	var saveAlias string
	var setDefault bool
	var tenantLabel string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Generate a new API key",
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantIDTrim := strings.TrimSpace(tenantID)
			if tenantIDTrim == "" {
				return errors.New("--tenant is required")
			}
			client, err := adminClientFromEnv(envCtx)
			if err != nil {
				return err
			}
			req := clientpkg.CreateAPIKeyRequest{}
			if strings.TrimSpace(appID) != "" {
				v := strings.TrimSpace(appID)
				req.AppID = &v
			}
			if strings.TrimSpace(description) != "" {
				v := strings.TrimSpace(description)
				req.Description = &v
			}
			generated, err := client.GenerateKey(cmd.Context(), tenantIDTrim, req)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Generated key: %s (prefix %s)\n", generated.APIKey, generated.Prefix)
			if strings.TrimSpace(saveAlias) != "" {
				entry := configpkg.APIKeyEntry{Key: generated.APIKey, Prefix: generated.Prefix}
				if generated.Description != nil {
					entry.Description = *generated.Description
				}
				if generated.AppID != nil {
					entry.AppID = *generated.AppID
				}
				if err := storeAPIKey(envCtx, tenantIDTrim, saveAlias, entry, setDefault, strings.TrimSpace(tenantLabel)); err != nil {
					return fmt.Errorf("key generated but failed to store: %w", err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Stored generated key as %s\n", saveAlias)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&tenantID, "tenant", "", "Tenant ID (required)")
	cmd.Flags().StringVar(&appID, "app-id", "", "Application ID to scope the key")
	cmd.Flags().StringVar(&description, "description", "", "Key description")
	cmd.Flags().StringVar(&saveAlias, "save-key-as", "", "Alias to store the generated key in local config")
	cmd.Flags().BoolVar(&setDefault, "set-default", false, "Mark stored key as default")
	cmd.Flags().StringVar(&tenantLabel, "tenant-name", "", "Optional friendly name for the tenant")

	return cmd
}

func newAdminKeyRevokeCommand(env *Environment) *cobra.Command {
	return &cobra.Command{
		Use:   "revoke <prefix>",
		Short: "Revoke an API key by prefix",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			client, err := adminClientFromEnv(envCtx)
			if err != nil {
				return err
			}
			if err := client.RevokeKey(cmd.Context(), strings.TrimSpace(args[0])); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Revoked key with prefix %s\n", args[0])
			return nil
		},
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func optional(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}
