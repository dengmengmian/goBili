package cmd

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// Build-time variables, injected via -ldflags.
// See Makefile for release builds.
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print goBili version information",
	Long:  "Print the version, build time, and git commit of goBili.",
	Run:   runVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func runVersion(_ *cobra.Command, _ []string) {
	fmt.Printf("goBili %s\n", Version)
	fmt.Printf("  build time: %s\n", BuildTime)
	fmt.Printf("  git commit: %s\n", GitCommit)

	// Fallback: if not built via Makefile (e.g. go install), use runtime build info.
	if Version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok {
			if info.Main.Version != "" && info.Main.Version != "(devel)" {
				fmt.Printf("  go module:  %s\n", info.Main.Version)
			}
			for _, s := range info.Settings {
				switch s.Key {
				case "vcs.revision":
					fmt.Printf("  vcs.revision: %s\n", s.Value)
				case "vcs.time":
					fmt.Printf("  vcs.time:     %s\n", s.Value)
				}
			}
		}
	}
}
