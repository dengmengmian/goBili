package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// completionCmd generates shell completion scripts for bash, zsh, fish, and powershell.
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Generate the autocompletion script for goBili for the specified shell.

Usage:
  source <(goBili completion bash)   # bash
  source <(goBili completion zsh)    # zsh
  goBili completion fish | source    # fish
  goBili completion powershell | Out-String | Invoke-Expression  # powershell`,
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
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
			return fmt.Errorf("unsupported shell: %s", args[0])
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
