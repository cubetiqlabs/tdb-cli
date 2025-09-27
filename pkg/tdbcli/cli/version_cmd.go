package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	versionpkg "cubetiqlabs/tinydb/pkg/tdbcli/version"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the TinyDB CLI version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), versionpkg.Display())
		},
	}
}
