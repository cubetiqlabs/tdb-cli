package cli

import (
	"os"

	"github.com/spf13/cobra"
)

func newCompletionCommand(root *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh]",
		Short: "Generate shell completion script",
		Long: `To load completions:

Bash:
  $ source <(tdb completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ tdb completion bash > /etc/bash_completion.d/tdb
  # macOS:
  $ tdb completion bash > /usr/local/etc/bash_completion.d/tdb

Zsh:
  $ autoload -U compinit; compinit
  $ source <(tdb completion zsh)
  # To load completions for each session, add to your ~/.zshrc:
  $ tdb completion zsh > "${fpath[1]}/_tdb"
`,
		Args: cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: []string{"bash", "zsh"},
		Hidden: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return cmd.Help()
			}
			switch args[0] {
			case "bash":
				return root.GenBashCompletion(os.Stdout)
			case "zsh":
				return root.GenZshCompletion(os.Stdout)
			default:
				return cmd.Help()
			}
		},
	}
	return cmd
}
