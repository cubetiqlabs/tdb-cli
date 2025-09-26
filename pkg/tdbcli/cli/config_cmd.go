package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	configpkg "cubetiqlabs/tinydb/pkg/tdbcli/config"
)

func registerConfigCommands(root *cobra.Command, env *Environment) {
	cfgCmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect or update local TinyDB CLI configuration",
	}

	cfgCmd.AddCommand(newConfigShowCommand(env))
	cfgCmd.AddCommand(newConfigSetEndpointCommand(env))
	cfgCmd.AddCommand(newConfigSetAdminSecretCommand(env))
	cfgCmd.AddCommand(newConfigStoreKeyCommand(env))
	cfgCmd.AddCommand(newConfigDeleteKeyCommand(env))
	cfgCmd.AddCommand(newConfigSetDefaultKeyCommand(env))
	cfgCmd.AddCommand(newConfigSetTenantNameCommand(env))

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

func newConfigSetEndpointCommand(env *Environment) *cobra.Command {
	return &cobra.Command{
		Use:   "set-endpoint <url>",
		Short: "Set the TinyDB API endpoint",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			endpoint := strings.TrimSpace(args[0])
			if endpoint == "" {
				return errors.New("endpoint cannot be empty")
			}
			env.Config.Endpoint = endpoint
			if err := env.Save(); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Endpoint set to %s\n", endpoint)
			return nil
		},
	}
}

func newConfigSetAdminSecretCommand(env *Environment) *cobra.Command {
	return &cobra.Command{
		Use:   "set-admin-secret <secret>",
		Short: "Set the admin secret used for privileged API calls",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			secret := strings.TrimSpace(args[0])
			if secret == "" {
				return errors.New("admin secret cannot be empty")
			}
			env.Config.AdminSecret = secret
			if err := env.Save(); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Admin secret updated")
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

func newConfigSetDefaultKeyCommand(env *Environment) *cobra.Command {
	return &cobra.Command{
		Use:   "set-default-key <tenant_id> <alias>",
		Short: "Mark a stored API key as the default for the tenant",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			if err := setDefaultKey(env, args[0], args[1]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Set key %s as default for tenant %s\n", args[1], args[0])
			return nil
		},
	}
}

func newConfigSetTenantNameCommand(env *Environment) *cobra.Command {
	return &cobra.Command{
		Use:   "set-tenant-name <tenant_id> <name>",
		Short: "Assign a friendly name for a tenant in config",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantID := strings.TrimSpace(args[0])
			name := strings.TrimSpace(args[1])
			if tenantID == "" || name == "" {
				return errors.New("tenant id and name are required")
			}
			cfg := env.Config
			tc := cfg.EnsureTenant(tenantID)
			tc.Name = name
			cfg.UpdateTenant(tenantID, tc)
			if err := env.Save(); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Tenant %s labeled as %s\n", tenantID, name)
			return nil
		},
	}
}
