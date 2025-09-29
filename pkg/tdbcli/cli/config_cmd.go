package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	clientpkg "github.com/cubetiqlabs/tdb-cli/pkg/tdbcli/client"
	configpkg "github.com/cubetiqlabs/tdb-cli/pkg/tdbcli/config"
)

func registerConfigCommands(root *cobra.Command, env *Environment) {
	cfgCmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect or update local TinyDB CLI configuration",
	}

	cfgCmd.AddCommand(newConfigShowCommand(env))
	cfgCmd.AddCommand(newConfigSetCommand(env))
	cfgCmd.AddCommand(newConfigStoreKeyCommand(env))
	cfgCmd.AddCommand(newConfigDeleteKeyCommand(env))

	root.AddCommand(cfgCmd)
}

func newConfigShowCommand(env *Environment) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Print the current CLI config as YAML",
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			display := *env.Config
			display.AdminSecret = env.Config.MaskedAdminSecret()
			data, err := yaml.Marshal(display)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		},
	}
}

func newConfigStoreKeyCommand(env *Environment) *cobra.Command {
	var storeKey string
	var fromStdin bool
	var prefix string
	var appID string
	var description string
	var setDefault bool
	var tenantName string

	cmd := &cobra.Command{
		Use:   "store-key <tenant_id> <alias>",
		Short: "Persist an API key in the local config",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantID := args[0]
			alias := args[1]

			keyValue := strings.TrimSpace(storeKey)
			if fromStdin {
				data, err := io.ReadAll(cmd.InOrStdin())
				if err != nil {
					return err
				}
				keyValue = strings.TrimSpace(string(data))
			}
			if keyValue == "" {
				return errors.New("api key value is required (use --key or --stdin)")
			}

			entry := configpkg.APIKeyEntry{
				Key:         keyValue,
				Prefix:      strings.TrimSpace(prefix),
				AppID:       strings.TrimSpace(appID),
				Description: strings.TrimSpace(description),
			}

			if err := storeAPIKey(env, tenantID, alias, entry, setDefault, strings.TrimSpace(tenantName)); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Stored key %s for tenant %s\n", alias, tenantID)
			return nil
		},
	}

	cmd.Flags().StringVar(&storeKey, "key", "", "API key value to store")
	cmd.Flags().BoolVar(&fromStdin, "stdin", false, "Read API key value from stdin")
	cmd.Flags().StringVar(&prefix, "prefix", "", "Optional key prefix for reference")
	cmd.Flags().StringVar(&appID, "app-id", "", "Optional application ID associated with the key")
	cmd.Flags().StringVar(&description, "description", "", "Optional description for this key")
	cmd.Flags().BoolVar(&setDefault, "default", false, "Mark this key as the tenant default")
	cmd.Flags().StringVar(&tenantName, "tenant-name", "", "Optional friendly name for the tenant")

	return cmd
}

func newConfigDeleteKeyCommand(env *Environment) *cobra.Command {
	return &cobra.Command{
		Use:   "delete-key <tenant_id> <alias>",
		Short: "Remove a stored API key",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			removed, err := deleteAPIKey(env, args[0], args[1])
			if err != nil {
				return err
			}
			if removed {
				fmt.Fprintf(cmd.OutOrStdout(), "Removed key %s for tenant %s\n", args[1], args[0])
			}
			return nil
		},
	}
}

func newConfigSetCommand(env *Environment) *cobra.Command {
	return &cobra.Command{
		Use:   "set <field> [values...]",
		Short: "Update core CLI settings (endpoint, admin-secret, api-key, default-key, tenant-name, default-tenant)",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			field := strings.ToLower(strings.TrimSpace(args[0]))
			switch field {
			case "endpoint", "api-endpoint":
				if len(args) != 2 {
					return errors.New("usage: tdb config set endpoint <url>")
				}
				endpoint := strings.TrimSpace(args[1])
				if endpoint == "" {
					return errors.New("endpoint cannot be empty")
				}
				envCtx.Config.Endpoint = endpoint
				if err := envCtx.Save(); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Endpoint set to %s\n", endpoint)
			case "admin-secret", "admin_secret":
				if len(args) != 2 {
					return errors.New("usage: tdb config set admin-secret <secret>")
				}
				secret := strings.TrimSpace(args[1])
				if secret == "" {
					return errors.New("admin secret cannot be empty")
				}
				envCtx.Config.AdminSecret = secret
				if err := envCtx.Save(); err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), "Admin secret updated")
			case "default-key", "default_key":
				if len(args) != 3 {
					return errors.New("usage: tdb config set default-key <tenant_id> <alias>")
				}
				if err := setDefaultKey(envCtx, args[1], args[2]); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Set key %s as default for tenant %s\n", args[2], args[1])
			case "api-key", "api_key":
				if len(args) != 2 {
					return errors.New("usage: tdb config set api-key <api_key>")
				}
				apiKey := strings.TrimSpace(args[1])
				if apiKey == "" {
					return errors.New("api key cannot be empty")
				}
				endpoint, err := ensureEndpoint(envCtx)
				if err != nil {
					return err
				}
				tenantClient, err := clientpkg.NewTenantClient(endpoint, apiKey)
				if err != nil {
					return fmt.Errorf("create tenant client: %w", err)
				}
				status, err := tenantClient.AuthStatus(cmd.Context(), "")
				if err != nil {
					return fmt.Errorf("api key verification failed: %w", err)
				}
				tenantID := strings.TrimSpace(status.TenantID)
				if tenantID == "" {
					return errors.New("api key verification succeeded but tenant id is missing")
				}
				alias := strings.TrimSpace(status.KeyPrefix)
				if alias == "" {
					alias = strings.TrimSpace(status.AppID)
				}
				if alias == "" {
					alias = "default"
				}
				entry := configpkg.APIKeyEntry{
					Key:    apiKey,
					Prefix: strings.TrimSpace(status.KeyPrefix),
					AppID:  strings.TrimSpace(status.AppID),
				}
				if desc := strings.TrimSpace(status.AppName); desc != "" {
					entry.Description = desc
				} else if scope := strings.TrimSpace(status.Scope); scope != "" {
					entry.Description = scope
				}
				tenantName := strings.TrimSpace(status.TenantName)
				if err := storeAPIKey(envCtx, tenantID, alias, entry, true, tenantName); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Verified key for tenant %s and stored as alias %s (default)\n", tenantID, alias)
			case "default-tenant", "default_tenant":
				if len(args) != 2 {
					return errors.New("usage: tdb config set default-tenant <tenant_id>")
				}
				if err := setDefaultTenant(envCtx, args[1]); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Default tenant set to %s\n", args[1])
			case "tenant-name", "tenant_name":
				if len(args) < 3 {
					return errors.New("usage: tdb config set tenant-name <tenant_id> <name>")
				}
				tenantID := strings.TrimSpace(args[1])
				name := strings.TrimSpace(strings.Join(args[2:], " "))
				if tenantID == "" || name == "" {
					return errors.New("tenant id and name are required")
				}
				cfg := envCtx.Config
				tc := cfg.EnsureTenant(tenantID)
				tc.Name = name
				cfg.UpdateTenant(tenantID, tc)
				if err := envCtx.Save(); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Tenant %s labeled as %s\n", tenantID, name)
			default:
				return fmt.Errorf("unknown config field %q; supported values: endpoint, admin-secret, api-key, default-key, tenant-name, default-tenant", field)
			}
			return nil
		},
	}
}
