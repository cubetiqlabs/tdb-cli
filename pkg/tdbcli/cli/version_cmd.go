package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	versionpkg "github.com/cubetiqlabs/tdb-cli/pkg/tdbcli/version"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the TinyDB CLI version",
		Run: func(cmd *cobra.Command, args []string) {
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "TinyDB CLI (%s)\n", versionpkg.Display())
			fmt.Fprintf(out, "  Version: %s\n", versionpkg.Number())
			fmt.Fprintf(out, "  Commit:  %s\n", versionpkg.CommitHash())
			fmt.Fprintf(out, "  Built:   %s\n", versionpkg.BuiltAt())
			fmt.Fprintf(out, "  Issues:  %s\n", versionpkg.IssuesURL)
		},
	}
}
