package cli

import (
	"fmt"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
)

func newTenantAuthCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var raw bool

	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Verify the configured API key by calling /api/me",
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantClient, keyEntry, tenantID, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}
			status, err := tenantClient.AuthStatus(cmd.Context(), auth.appID)
			if err != nil {
				return err
			}
			if raw {
				return printJSON(cmd, status)
			}
			out := cmd.OutOrStdout()
			tenName := strings.TrimSpace(status.TenantName)
			if tenName == "" {
				tenName = tenantID
			}
			fmt.Fprintf(out, "Tenant: %s (%s)\n", tenName, status.TenantID)
			appID := strings.TrimSpace(status.AppID)
			if appID == "" {
				appID = strings.TrimSpace(auth.appID)
			}
			if appID != "" {
				appName := strings.TrimSpace(status.AppName)
				if appName == "" {
					appName = appID
				}
				fmt.Fprintf(out, "Application: %s (%s)\n", appName, appID)
			} else {
				fmt.Fprintln(out, "Application: (not scoped)")
			}
			if prefix := strings.TrimSpace(keyEntry.Prefix); prefix != "" {
				fmt.Fprintf(out, "Key Prefix: %s\n", prefix)
			}
			statusText := strings.TrimSpace(status.Status)
			if statusText == "" {
				statusText = "unknown"
			}

			if scope := strings.TrimSpace(status.Scope); scope != "" {
				fmt.Fprintf(out, "Scope: %s\n", scope)
			}

			if status.CreatedAt != nil {
				fmt.Fprintf(out, "Created At: %s\n", humanize.Time(*status.CreatedAt))
			}

			if status.LastUsed != nil {
				fmt.Fprintf(out, "Last Used: %s\n", humanize.Time(*status.LastUsed))
			}

			return nil
		},
	}

	auth.bindWithApp(cmd)
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON response")
	return cmd
}
