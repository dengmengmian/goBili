package cmd

import (
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
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
