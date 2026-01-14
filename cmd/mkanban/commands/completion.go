package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for mkanban.

To load completions:

Bash:
  $ source <(mkanban completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ mkanban completion bash > /etc/bash_completion.d/mkanban
  # macOS:
  $ mkanban completion bash > $(brew --prefix)/etc/bash_completion.d/mkanban

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ mkanban completion zsh > "${fpath[1]}/_mkanban"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ mkanban completion fish | source

  # To load completions for each session, execute once:
  $ mkanban completion fish > ~/.config/fish/completions/mkanban.fish

PowerShell:
  PS> mkanban completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> mkanban completion powershell > mkanban.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resolvedArgs, err := resolveArgs(args, 1)
		if err != nil {
			return err
		}
		switch resolvedArgs[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		default:
			return fmt.Errorf("invalid shell: %s", resolvedArgs[0])
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
