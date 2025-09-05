package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// legalCmd represents the legal command
var legalCmd = &cobra.Command{
	Use:   "legal",
	Short: "显示法律协议和条款信息",
	Long: `显示 goBili 的法律协议和条款信息，包括：
- 使用条款
- 隐私政策
- 开源协议
- 免责声明`,
	Run: runLegal,
}

func init() {
	rootCmd.AddCommand(legalCmd)
}

func runLegal(cmd *cobra.Command, args []string) {
	fmt.Println("=== goBili 法律协议和条款 ===")
	fmt.Println()

	fmt.Println("📋 重要声明：")
	fmt.Println("goBili 是一个开源项目，仅供学习和研究使用。")
	fmt.Println("使用本软件时，请遵守相关法律法规和平台服务条款。")
	fmt.Println()

	fmt.Println("📄 协议文件：")
	fmt.Println("• LICENSE      - MIT 开源协议")
	fmt.Println("• TERMS.md     - 使用条款")
	fmt.Println("• PRIVACY.md   - 隐私政策")
	fmt.Println("• CONTRIBUTING.md - 贡献指南")
	fmt.Println()

	fmt.Println("⚠️  使用须知：")
	fmt.Println("1. 仅允许下载您有合法权限访问的内容")
	fmt.Println("2. 下载的内容仅限个人使用，不得传播或分享")
	fmt.Println("3. 不得将下载的内容用于商业目的")
	fmt.Println("4. 尊重内容创作者的版权")
	fmt.Println("5. 遵守 Bilibili 平台的服务条款")
	fmt.Println()

	fmt.Println("🔒 隐私保护：")
	fmt.Println("• 本软件仅在本地处理用户数据")
	fmt.Println("• 不会收集或上传用户的个人信息")
	fmt.Println("• Cookie 等认证信息仅存储在本地")
	fmt.Println()

	fmt.Println("⚖️  免责声明：")
	fmt.Println("• 开发者不对用户使用本软件产生的任何后果负责")
	fmt.Println("• 用户因违反法律法规或平台条款产生的后果由用户自行承担")
	fmt.Println("• 开发者不承担因软件缺陷或故障造成的任何损失")
	fmt.Println()

	fmt.Println("📖 完整协议：")
	fmt.Println("请查看项目根目录下的协议文件获取完整信息。")
	fmt.Println()

	// 检查协议文件是否存在
	protocolFiles := []string{"LICENSE", "TERMS.md", "PRIVACY.md", "CONTRIBUTING.md"}
	fmt.Println("📁 协议文件状态：")

	for _, file := range protocolFiles {
		if _, err := os.Stat(file); err == nil {
			fmt.Printf("✅ %s - 存在\n", file)
		} else {
			fmt.Printf("❌ %s - 不存在\n", file)
		}
	}
	fmt.Println()

	fmt.Println("📧 联系方式：")
	fmt.Println("• Email: my@dengmengmian.com")
	fmt.Println("• GitHub Issues: https://github.com/dengmengmian/goBili")
	fmt.Println()

	fmt.Println("💡 提示：")
	fmt.Println("使用 'goBili version' 查看版本信息")
	fmt.Println("使用 'goBili help' 查看所有可用命令")
	fmt.Println()

	fmt.Println("继续使用本软件即表示您同意相关协议和条款。")
}
