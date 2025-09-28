package cli

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"

	clientpkg "github.com/cubetiqlabs/tdb-cli/pkg/tdbcli/client"
	configpkg "github.com/cubetiqlabs/tdb-cli/pkg/tdbcli/config"
	versionpkg "github.com/cubetiqlabs/tdb-cli/pkg/tdbcli/version"
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
	var hideRevoked bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List API keys for a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantIDTrim, err := resolveTenantID(envCtx, tenantID)
			if err != nil {
				return err
			}
			if !cmd.Flags().Changed("tenant") {
				fmt.Fprintf(cmd.OutOrStdout(), "Using default tenant %s\n", tenantIDTrim)
			}
			client, err := adminClientFromEnv(envCtx)
			if err != nil {
				return err
			}
			filter := normalizeOptionalString(appID)
			keys, err := client.ListKeys(cmd.Context(), tenantIDTrim, filter)
			if err != nil {
				return err
			}
			rows, message := buildKeyRows(keys, hideRevoked)
			if len(rows) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), message)
				return nil
			}
			renderTable(cmd, []string{"PREFIX", "SCOPE", "DESCRIPTION", "HAS APP", "STATUS", "CREATED", "LAST USED", "REVOKED"}, rows)
			return nil
		},
	}

	cmd.Flags().StringVar(&tenantID, "tenant", "", "Tenant ID (defaults to your configured default tenant when omitted)")
	cmd.Flags().StringVar(&appID, "app-id", "", "Filter keys by application ID")
	cmd.Flags().BoolVar(&hideRevoked, "hide-revoked", false, "Hide revoked keys from the output")

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
			tenantIDTrim, err := resolveTenantID(envCtx, tenantID)
			if err != nil {
				return err
			}
			if !cmd.Flags().Changed("tenant") {
				fmt.Fprintf(cmd.OutOrStdout(), "Using default tenant %s\n", tenantIDTrim)
			}
			client, err := adminClientFromEnv(envCtx)
			if err != nil {
				return err
			}
			req, desc := buildCreateKeyRequest(appID, description)
			generated, err := client.GenerateKey(cmd.Context(), tenantIDTrim, req)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Generated key: %s (prefix %s)\n", generated.APIKey, generated.Prefix)
			if alias := strings.TrimSpace(saveAlias); alias != "" {
				if err := persistGeneratedKey(envCtx, tenantIDTrim, alias, generated, desc, setDefault, tenantLabel); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Stored generated key as %s\n", alias)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&tenantID, "tenant", "", "Tenant ID (defaults to your configured default tenant when omitted)")
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
		return "-"
	}
	return t.Local().Format("2006 Jan 02 03:04 PM")
}

func normalizeOptionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func buildKeyRows(keys []clientpkg.APIKey, hideRevoked bool) ([][]string, string) {
	if len(keys) == 0 {
		return nil, "No keys found"
	}
	rows := make([][]string, 0, len(keys))
	for _, key := range keys {
		if hideRevoked && key.RevokedAt != nil {
			continue
		}
		rows = append(rows, formatKeyRow(key))
	}
	if len(rows) == 0 {
		return nil, "No keys found (all revoked keys hidden)"
	}
	return rows, ""
}

func formatKeyRow(k clientpkg.APIKey) []string {
	scope := keyScope(k)
	status, revoked := keyStatusAndRevoked(k)
	created := formatCreatedWithAge(k.CreatedAt)
	lastUsed := formatRelativeTimePtr(k.LastUsedAt, "never")
	return []string{k.Prefix, scope, optional(k.Description), hasAppIndicator(k), status, created, lastUsed, revoked}
}

func keyScope(k clientpkg.APIKey) string {
	if k.Scope != "application" || k.AppID == nil {
		return k.Scope
	}
	app := strings.TrimSpace(*k.AppID)
	if app == "" {
		return k.Scope
	}
	return fmt.Sprintf("%s (%s)", k.Scope, app)
}

func hasAppIndicator(k clientpkg.APIKey) string {
	if k.AppID == nil {
		return "❌"
	}
	if strings.TrimSpace(*k.AppID) == "" {
		return "❌"
	}
	return "✅"
}

func keyStatusAndRevoked(k clientpkg.APIKey) (string, string) {
	if k.RevokedAt == nil {
		return "Active", "-"
	}
	return "Revoked", formatRelativeTimePtr(k.RevokedAt, "-")
}

func formatCreatedWithAge(t time.Time) string {
	created := formatTime(t)
	age := formatRelativeTime(t, "-")
	if age == "-" {
		return created
	}
	return fmt.Sprintf("%s (%s)", created, age)
}

func buildCreateKeyRequest(appID, description string) (clientpkg.CreateAPIKeyRequest, string) {
	req := clientpkg.CreateAPIKeyRequest{}
	if trimmed := strings.TrimSpace(appID); trimmed != "" {
		req.AppID = &trimmed
	}
	desc := strings.TrimSpace(description)
	if desc == "" {
		desc = versionpkg.DefaultAPIKeyDescription()
	}
	req.Description = &desc
	return req, desc
}

func persistGeneratedKey(envCtx *Environment, tenantID, alias string, generated *clientpkg.GeneratedKey, fallbackDesc string, setDefault bool, tenantLabel string) error {
	entry := configpkg.APIKeyEntry{Key: generated.APIKey, Prefix: generated.Prefix, Description: fallbackDesc}
	if generated.Description != nil {
		if trimmed := strings.TrimSpace(*generated.Description); trimmed != "" {
			entry.Description = trimmed
		}
	}
	if generated.AppID != nil {
		entry.AppID = *generated.AppID
	}
	if err := storeAPIKey(envCtx, tenantID, alias, entry, setDefault, strings.TrimSpace(tenantLabel)); err != nil {
		return fmt.Errorf("key generated but failed to store: %w", err)
	}
	return nil
}

func formatRelativeTime(t time.Time, zeroFallback string) string {
	if t.IsZero() {
		return zeroFallback
	}
	return humanize.Time(t)
}

func formatRelativeTimePtr(t *time.Time, zeroFallback string) string {
	if t == nil {
		return zeroFallback
	}
	return formatRelativeTime(*t, zeroFallback)
}

func optional(ptr *string) string {
	if ptr == nil {
		return "-"
	}
	trimmed := strings.TrimSpace(*ptr)
	if trimmed == "" {
		return "-"
	}
	return trimmed
}
