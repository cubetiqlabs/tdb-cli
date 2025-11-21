package cli

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/AlecAivazis/survey/v2"
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
	cfgCmd.AddCommand(newConfigUseCommand(env))
	cfgCmd.AddCommand(newConfigSwitchCommand(env))
	cfgCmd.AddCommand(newConfigListCommand(env))

	root.AddCommand(cfgCmd)
}

func newConfigShowCommand(env *Environment) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Print the current CLI config as YAML",
		Long:  `Display the current TinyDB CLI configuration including endpoint, stored API keys, and tenant settings. Admin secrets are masked for security.`,
		Example: `  # Show current configuration
  tdb config show

  # Redirect to file
  tdb config show > config-backup.yaml`,
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
		Long: `Store an API key in the local CLI configuration for convenient reuse.

Stored keys can be referenced by alias instead of passing the full key value with each command. Keys are stored securely in the local config file.

You can optionally mark a key as default for a tenant and associate it with a specific application scope.`,
		Example: `  # Store an API key
  tdb config store-key tenant_123 my-key \
    --key "tdb_abc123..." \
    --description "Production API key"

  # Store and set as default
  tdb config store-key tenant_123 default-key \
    --key "tdb_xyz789..." \
    --set-default \
    --tenant-name "Production"

  # Store key from stdin
  echo "tdb_secret..." | tdb config store-key tenant_456 stdin-key --stdin

  # Store app-scoped key
  tdb config store-key tenant_789 app-key \
    --key "tdb_app..." \
    --app-id app_123 \
    --description "App-specific key"

  # Usage after storing:
  tdb tenant collections list --key my-key`,
		Args: cobra.ExactArgs(2),
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

func newConfigUseCommand(env *Environment) *cobra.Command {
	var shouldSetDefault bool
	cmd := &cobra.Command{
		Use:   "use <tenant_id> [key_alias]",
		Short: "Switch active profile (tenant and optionally key)",
		Long: `Switch the active tenant profile and optionally select a specific API key.

This command sets the default tenant and optionally the default key for that tenant, making it quick to work with different profiles without repeating --tenant flags.

If only tenant_id is provided, the existing default key for that tenant is used.
If key_alias is provided, it becomes the default key for the tenant (unless --no-set-default is used).`,
		Example: `  # Switch to a tenant (using existing default key)
  tdb config use tenant_123

  # Switch to a tenant and set a specific key as default
  tdb config use tenant_123 my-prod-key

  # Switch tenant and key without changing the tenant's default key
  tdb config use tenant_456 temp-key --no-set-default

  # List profiles first, then switch
  tdb config list
  tdb config use tenant_789`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}

			tenantID := strings.TrimSpace(args[0])
			if tenantID == "" {
				return errors.New("tenant_id cannot be empty")
			}

			// Check tenant exists in config
			tc, exists := envCtx.Config.Tenants[tenantID]
			if !exists {
				return fmt.Errorf("tenant %s not found in config; store a key first with `tdb config store-key`", tenantID)
			}

			// Handle optional key alias
			if len(args) == 2 {
				keyAlias := strings.TrimSpace(args[1])
				if keyAlias == "" {
					return errors.New("key_alias cannot be empty when provided")
				}

				// Verify key exists
				if tc.Keys == nil || tc.Keys[keyAlias].Key == "" {
					return fmt.Errorf("key %s not found for tenant %s", keyAlias, tenantID)
				}

				// Set as default key if flag is true (default behavior)
				if shouldSetDefault {
					if err := setDefaultKey(envCtx, tenantID, keyAlias); err != nil {
						return err
					}
					fmt.Fprintf(cmd.OutOrStdout(), "Set key %s as default for tenant %s\n", keyAlias, tenantID)
				}
			} else {
				// No key specified, verify tenant has a default key
				if tc.DefaultKey == "" {
					return fmt.Errorf("tenant %s has no default key; specify a key alias or set one with --default", tenantID)
				}
			}

			// Set as default tenant
			if err := setDefaultTenant(envCtx, tenantID); err != nil {
				return err
			}

			tenantName := tc.Name
			if tenantName == "" {
				tenantName = tenantID
			}
			keyInfo := tc.DefaultKey
			if keyInfo == "" {
				keyInfo = "(no default key)"
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Switched to profile: %s (tenant: %s, key: %s)\n", tenantName, tenantID, keyInfo)
			return nil
		},
	}

	cmd.Flags().BoolVar(&shouldSetDefault, "set-default", true, "Set the specified key as default for the tenant")
	return cmd
}

func newConfigSwitchCommand(env *Environment) *cobra.Command {
	return &cobra.Command{
		Use:   "switch",
		Short: "Interactively switch profiles using arrow keys",
		Long: `Launch an interactive menu to switch between configured tenant profiles.

Use arrow keys (↑/↓) to navigate through available profiles and press Enter to select.
Each profile shows the tenant ID, friendly name (if set), and default key alias.

This is a convenient alternative to 'tdb config use' when you want to browse and select from all available profiles.`,
		Example: `  # Launch interactive profile switcher
  tdb config switch

  # Alternative: use the non-interactive version
  tdb config use tenant_123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}

			if len(envCtx.Config.Tenants) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No profiles configured. Add one with `tdb config store-key` or `tdb config set api-key`")
				return nil
			}

			// Build profile options
			type profileOption struct {
				tenantID   string
				name       string
				defaultKey string
				keyCount   int
				display    string
			}

			defaultTenant := strings.TrimSpace(envCtx.Config.DefaultTenant)
			options := make([]profileOption, 0, len(envCtx.Config.Tenants))

			for tenantID, tc := range envCtx.Config.Tenants {
				name := tc.Name
				if name == "" {
					name = tenantID
				}

				defaultKey := tc.DefaultKey
				if defaultKey == "" {
					defaultKey = "(no default key)"
				}

				active := "  "
				if tenantID == defaultTenant {
					active = "→ "
				}

				// Format: "→ TenantName (tenant_id) - Key: default_key [3 keys]"
				display := fmt.Sprintf("%s%s (%s) - Key: %s [%d keys]",
					active, name, tenantID, defaultKey, len(tc.Keys))

				options = append(options, profileOption{
					tenantID:   tenantID,
					name:       name,
					defaultKey: tc.DefaultKey,
					keyCount:   len(tc.Keys),
					display:    display,
				})
			}

			// Sort options: active first, then alphabetically by name
			sort.Slice(options, func(i, j int) bool {
				iIsActive := options[i].tenantID == defaultTenant
				jIsActive := options[j].tenantID == defaultTenant
				if iIsActive != jIsActive {
					return iIsActive
				}
				return options[i].name < options[j].name
			})

			// Build display strings
			displayStrings := make([]string, len(options))
			for i, opt := range options {
				displayStrings[i] = opt.display
			}

			// Find default selection index (current active profile)
			defaultIdx := 0
			for i, opt := range options {
				if opt.tenantID == defaultTenant {
					defaultIdx = i
					break
				}
			}

			// Create interactive prompt
			prompt := &survey.Select{
				Message: "Select a profile to switch to:",
				Options: displayStrings,
				Default: defaultIdx,
			}

			var selectedIdx int
			if err := survey.AskOne(prompt, &selectedIdx); err != nil {
				return fmt.Errorf("profile selection cancelled or failed: %w", err)
			}

			selected := options[selectedIdx]

			// Check if already active
			if selected.tenantID == defaultTenant {
				fmt.Fprintf(cmd.OutOrStdout(), "Already using profile: %s\n", selected.name)
				return nil
			}

			// For multi-key tenants, optionally let user select a key
			if selected.keyCount > 1 {
				tc := envCtx.Config.Tenants[selected.tenantID]
				keyOptions := make([]string, 0, len(tc.Keys))
				keyAliases := make([]string, 0, len(tc.Keys))

				for alias, entry := range tc.Keys {
					isDefault := ""
					if alias == tc.DefaultKey {
						isDefault = " (current default)"
					}
					desc := entry.Description
					if desc == "" {
						desc = "No description"
					}
					display := fmt.Sprintf("%s - %s%s", alias, desc, isDefault)
					keyOptions = append(keyOptions, display)
					keyAliases = append(keyAliases, alias)
				}

				// Sort keys: default first, then alphabetically
				sort.Slice(keyOptions, func(i, j int) bool {
					iIsDefault := keyAliases[i] == tc.DefaultKey
					jIsDefault := keyAliases[j] == tc.DefaultKey
					if iIsDefault != jIsDefault {
						return iIsDefault
					}
					return keyAliases[i] < keyAliases[j]
				})

				keyPrompt := &survey.Select{
					Message: fmt.Sprintf("Select a key for %s:", selected.name),
					Options: keyOptions,
					Default: 0,
				}

				var selectedKeyIdx int
				if err := survey.AskOne(keyPrompt, &selectedKeyIdx); err != nil {
					// User cancelled key selection, use default key
					if selected.defaultKey == "" {
						return errors.New("key selection cancelled and no default key configured")
					}
					fmt.Fprintf(cmd.OutOrStdout(), "Using default key: %s\n", selected.defaultKey)
				} else {
					// Set selected key as default
					selectedKeyAlias := keyAliases[selectedKeyIdx]
					if err := setDefaultKey(envCtx, selected.tenantID, selectedKeyAlias); err != nil {
						return err
					}
					fmt.Fprintf(cmd.OutOrStdout(), "Set key %s as default\n", selectedKeyAlias)
				}
			}

			// Switch to selected profile
			if err := setDefaultTenant(envCtx, selected.tenantID); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "✓ Switched to profile: %s\n", selected.name)
			return nil
		},
	}
}

func newConfigListCommand(env *Environment) *cobra.Command {
	var showKeys bool
	var raw bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configured profiles (tenants)",
		Long: `Display all configured tenant profiles with their associated keys and metadata.

This command shows all tenants stored in your configuration, making it easy to see available profiles before switching with 'tdb config use'.

The active (default) tenant is indicated with a marker (*).`,
		Aliases: []string{"ls", "profiles"},
		Example: `  # List all profiles
  tdb config list

  # Show detailed key information
  tdb config list --show-keys

  # Get raw JSON output
  tdb config list --raw`,
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}

			if len(envCtx.Config.Tenants) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No profiles configured. Add one with `tdb config store-key` or `tdb config set api-key`")
				return nil
			}

			if raw {
				return printJSON(cmd, envCtx.Config.Tenants)
			}

			defaultTenant := strings.TrimSpace(envCtx.Config.DefaultTenant)

			if !showKeys {
				// Compact view: one row per tenant
				rows := make([][]string, 0, len(envCtx.Config.Tenants))
				for tenantID, tc := range envCtx.Config.Tenants {
					active := " "
					if tenantID == defaultTenant {
						active = "*"
					}

					name := tc.Name
					if name == "" {
						name = "-"
					}

					defaultKey := tc.DefaultKey
					if defaultKey == "" {
						defaultKey = "-"
					}

					keyCount := fmt.Sprintf("%d", len(tc.Keys))

					rows = append(rows, []string{
						active,
						tenantID,
						name,
						defaultKey,
						keyCount,
					})
				}

				renderTable(cmd, []string{"", "TENANT ID", "NAME", "DEFAULT KEY", "KEYS"}, rows)
				fmt.Fprintf(cmd.OutOrStdout(), "\nUse 'tdb config use <tenant_id>' to switch profiles\n")
			} else {
				// Detailed view: show all keys per tenant
				rows := make([][]string, 0)
				for tenantID, tc := range envCtx.Config.Tenants {
					active := " "
					if tenantID == defaultTenant {
						active = "*"
					}

					name := tc.Name
					if name == "" {
						name = "-"
					}

					if len(tc.Keys) == 0 {
						rows = append(rows, []string{
							active,
							tenantID,
							name,
							"-",
							"-",
							"-",
						})
					} else {
						for keyAlias, keyEntry := range tc.Keys {
							isDefault := " "
							if keyAlias == tc.DefaultKey {
								isDefault = "✓"
							}

							desc := keyEntry.Description
							if desc == "" {
								desc = "-"
							}

							appID := keyEntry.AppID
							if appID == "" {
								appID = "-"
							}

							rows = append(rows, []string{
								active,
								tenantID,
								name,
								keyAlias,
								isDefault,
								desc,
								appID,
							})

							// Only show active marker on first key row per tenant
							active = ""
							tenantID = ""
							name = ""
						}
					}
				}

				renderTable(cmd, []string{"", "TENANT ID", "NAME", "KEY ALIAS", "DEFAULT", "DESCRIPTION", "APP ID"}, rows)
				fmt.Fprintf(cmd.OutOrStdout(), "\nUse 'tdb config use <tenant_id> [key_alias]' to switch profiles\n")
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&showKeys, "show-keys", false, "Show detailed information about stored keys")
	cmd.Flags().BoolVarP(&raw, "raw", "r", false, "Output raw JSON")

	return cmd
}
