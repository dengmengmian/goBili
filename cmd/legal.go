package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// legalCmd represents the legal command
var legalCmd = &cobra.Command{
	Use:   "legal",
	Short: "Show legal notices and terms of use",
	Long: `Display goBili legal notices, including:
- Terms of use
- Privacy policy
- Open source license
- Disclaimer`,
	Run: runLegal,
}

func init() {
	rootCmd.AddCommand(legalCmd)
}

func runLegal(_ *cobra.Command, _ []string) {
	fmt.Println("=== goBili Legal Notices ===")
	fmt.Println()

	fmt.Println("\U0001F4CB Important Notice:")
	fmt.Println("goBili is an open-source project for educational and research purposes only.")
	fmt.Println("By using this software, you agree to comply with all applicable laws and platform terms.")
	fmt.Println()

	fmt.Println("\U0001F4C4 Legal Documents:")
	fmt.Println("  LICENSE          - MIT Open Source License")
	fmt.Println("  TERMS.md         - Terms of Use")
	fmt.Println("  PRIVACY.md       - Privacy Policy")
	fmt.Println("  CONTRIBUTING.md  - Contribution Guidelines")
	fmt.Println()

	fmt.Println("\u26A0\uFE0F  Usage Restrictions:")
	fmt.Println("1. Only download content you have legal rights to access")
	fmt.Println("2. Downloaded content is for personal use only; do not redistribute")
	fmt.Println("3. Do not use downloaded content for commercial purposes")
	fmt.Println("4. Respect the copyright of content creators")
	fmt.Println("5. Comply with the Bilibili platform terms of service")
	fmt.Println()

	fmt.Println("\U0001F512 Privacy:")
	fmt.Println("  This software processes data locally only.")
	fmt.Println("  No personal information is collected or uploaded.")
	fmt.Println("  Authentication cookies are stored locally on your device.")
	fmt.Println()

	fmt.Println("\u2696\uFE0F  Disclaimer:")
	fmt.Println("  The developers are not responsible for any consequences of using this software.")
	fmt.Println("  Users bear full responsibility for violations of laws or platform terms.")
	fmt.Println("  The developers assume no liability for damages caused by software defects.")
	fmt.Println()

	fmt.Println("\U0001F4D6 Full Documents:")
	fmt.Println("See the project root directory for the complete legal documents.")
	fmt.Println()

	// Check whether the legal documents exist.
	protocolFiles := []string{"LICENSE", "TERMS.md", "PRIVACY.md", "CONTRIBUTING.md"}
	fmt.Println("\U0001F4C1 Document Status:")

	for _, file := range protocolFiles {
		if _, err := os.Stat(file); err == nil {
			fmt.Printf("  \u2705 %s - present\n", file)
		} else {
			fmt.Printf("  \u274C %s - missing\n", file)
		}
	}
	fmt.Println()

	fmt.Println("\U0001F4E7 Contact:")
	fmt.Println("  Email: my@dengmengmian.com")
	fmt.Println("  GitHub Issues: https://github.com/dengmengmian/goBili")
	fmt.Println()

	fmt.Println("\U0001F4A1 Tip:")
	fmt.Println("  Use 'goBili version' to see version information")
	fmt.Println("  Use 'goBili help' to list all available commands")
	fmt.Println()

	fmt.Println("Continued use of this software constitutes acceptance of these terms.")
}
