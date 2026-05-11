package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Generate a shell completion script for the specified shell.

To load completions:

Bash (Linux):
  $ sampleslice completion bash > /etc/bash_completion.d/sampleslice

Bash (macOS with Homebrew bash-completion@2):
  $ sampleslice completion bash > /usr/local/etc/bash_completion.d/sampleslice

Zsh:
  $ sampleslice completion zsh > "${fpath[1]}/_sampleslice"

Fish:
  $ sampleslice completion fish > ~/.config/fish/completions/sampleslice.fish

PowerShell:
  PS> sampleslice completion powershell | Out-String | Invoke-Expression
`,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			return cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		default:
			return fmt.Errorf("unsupported shell %q (use bash, zsh, fish, or powershell)", args[0])
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
