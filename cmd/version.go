package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// 版本信息变量，需要在构建时设置
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示 goBili 版本信息",
	Long: `显示 goBili 的版本信息，包括：
- 版本号
- 构建时间
- Git 提交哈希`,
	Run: runVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func runVersion(cmd *cobra.Command, args []string) {
	fmt.Printf("goBili 版本信息:\n")
	fmt.Printf("  版本: %s\n", Version)
	fmt.Printf("  构建时间: %s\n", BuildTime)
	fmt.Printf("  Git提交: %s\n", GitCommit)
}
