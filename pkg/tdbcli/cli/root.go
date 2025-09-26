package cli

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	configpkg "cubetiqlabs/tinydb/pkg/tdbcli/config"
)

// NewRootCommand constructs the root Cobra command for the TinyDB CLI.
func NewRootCommand() *cobra.Command {
	env := &Environment{}
	var configPath string
	var overrideEndpoint string
	var overrideAdminSecret string

	defaultPath, err := configpkg.DefaultPath()
	if err == nil {
		configPath = defaultPath
	}

	cmd := &cobra.Command{
		Use:           "tdb",
		Short:         "TinyDB administrative and client CLI",
		Long:          "Manage TinyDB tenants, API keys, and applications, and interact with client endpoints.",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			path := strings.TrimSpace(configPath)
			if path == "" {
				var err error
				path, err = configpkg.DefaultPath()
				if err != nil {
					return err
				}
			}

			cfg, err := configpkg.Load(path)
			if err != nil {
				return err
			}

			env.ConfigPath = path
			env.Config = cfg

			if ep := strings.TrimSpace(overrideEndpoint); ep != "" {
				env.Config.Endpoint = ep
			}
			if secret := strings.TrimSpace(overrideAdminSecret); secret != "" {
				env.Config.AdminSecret = secret
			}

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			ctx = withEnvironment(ctx, env)
			cmd.SetContext(ctx)
			if root := cmd.Root(); root != cmd {
				root.SetContext(ctx)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().StringVar(&configPath, "config", configPath, "Path to TinyDB CLI config file")
	cmd.PersistentFlags().StringVar(&overrideEndpoint, "endpoint", "", "Override TinyDB endpoint for this invocation")
	cmd.PersistentFlags().StringVar(&overrideAdminSecret, "admin-secret", "", "Override admin secret for this invocation")

	cmd.CompletionOptions.DisableDefaultCmd = true

	registerConfigCommands(cmd, env)
	registerAdminCommands(cmd, env)
	registerTenantCommands(cmd, env)

	return cmd
}

// Execute runs the TinyDB CLI with the provided context.
func Execute(ctx context.Context) error {
	root := NewRootCommand()
	if ctx != nil {
		return root.ExecuteContext(ctx)
	}
	return root.Execute()
}
