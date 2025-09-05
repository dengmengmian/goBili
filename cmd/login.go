package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"goBili/auth"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to Bilibili using QR code or cookie file",
	Long: `Login to Bilibili using QR code authentication or cookie file.
This will generate a QR code that you can scan with the Bilibili mobile app to authenticate,
or you can provide a cookie file with authentication information.`,
	RunE: runLogin,
}

func init() {
	rootCmd.AddCommand(loginCmd)

	// Add flag for cookie file
	loginCmd.Flags().StringP("cookie-file", "c", "", "path to cookie file containing authentication information")
	// Add flag for browser login
	loginCmd.Flags().BoolP("browser", "b", false, "open browser to login and automatically capture cookies")
}

func runLogin(cmd *cobra.Command, args []string) error {
	// Get config directory
	configDir := getConfigDir()

	// Initialize logger
	logger := logrus.New()
	if viper.GetBool("verbose") {
		logger.SetLevel(logrus.DebugLevel)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}

	// Initialize auth manager
	authManager := auth.NewAuthManager(configDir, logger)

	// Load existing cookies if any
	if err := authManager.LoadCookies(); err != nil {
		logger.Warnf("Failed to load existing cookies: %v", err)
	}

	// Check if already authenticated
	if authManager.IsAuthenticated() {
		userInfo, err := authManager.GetUserInfo()
		if err != nil {
			logger.Warnf("Failed to get user info: %v", err)
			fmt.Println("You appear to be logged in, but user info could not be retrieved.")
			fmt.Println("You may need to re-login.")
		} else {
			fmt.Printf("Already logged in as: %s (UID: %d)\n", userInfo.Name, userInfo.Mid)
			fmt.Println("Use --force flag to force re-login if needed.")
			return nil
		}
	}

	// Check login method
	cookieFile, _ := cmd.Flags().GetString("cookie-file")
	useBrowser, _ := cmd.Flags().GetBool("browser")

	if useBrowser {
		// Browser login
		fmt.Println("Starting browser login...")
		if err := loginWithBrowser(authManager, logger); err != nil {
			return fmt.Errorf("browser login failed: %v", err)
		}
	} else if cookieFile != "" {
		// Load cookies from file
		fmt.Printf("Loading cookies from file: %s\n", cookieFile)
		if err := loadCookiesFromFile(authManager, cookieFile); err != nil {
			return fmt.Errorf("failed to load cookies from file: %v", err)
		}

		// Save cookies to config directory
		if err := authManager.SaveCookies(); err != nil {
			logger.Warnf("Failed to save cookies: %v", err)
		}
	} else {
		// Perform QR code login
		fmt.Println("Starting QR code login...")
		if err := authManager.LoginWithQRCode(); err != nil {
			return fmt.Errorf("QR code login failed: %v", err)
		}
	}

	// Verify login by getting user info
	userInfo, err := authManager.GetUserInfo()
	if err != nil {
		return fmt.Errorf("login verification failed: %v", err)
	}

	fmt.Printf("Login successful! Welcome, %s (UID: %d)\n", userInfo.Name, userInfo.Mid)
	fmt.Printf("User level: %d\n", userInfo.Level)
	if userInfo.VipStatus > 0 {
		fmt.Println("VIP status: Active")
	}

	return nil
}

// loadCookiesFromFile loads cookies from a text file
func loadCookiesFromFile(authManager *auth.AuthManager, filePath string) error {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("cookie file does not exist: %s", filePath)
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read cookie file: %v", err)
	}

	// Parse cookie content
	lines := strings.Split(string(content), "\n")
	cookieCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue // Skip empty lines and comments
		}

		// Parse cookie line (format: name value domain path expires size httpOnly secure)
		parts := strings.Split(line, "\t")
		if len(parts) >= 2 {
			name := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Only load essential cookies
			essentialCookies := []string{"SESSDATA", "bili_jct", "DedeUserID", "DedeUserID__ckMd5", "sid", "buvid3", "buvid4"}
			for _, essential := range essentialCookies {
				if name == essential {
					authManager.SetCookie(name, value)
					cookieCount++
					break
				}
			}
		}
	}

	if cookieCount == 0 {
		return fmt.Errorf("no valid cookies found in file")
	}

	fmt.Printf("Loaded %d cookies from file\n", cookieCount)
	return nil
}

// loginWithBrowser opens browser and provides instructions for manual cookie extraction
func loginWithBrowser(authManager *auth.AuthManager, logger *logrus.Logger) error {
	fmt.Println("=== 浏览器登录模式 ===")
	fmt.Println("此模式将打开浏览器让您登录B站，然后您需要手动复制Cookie。")
	fmt.Println()

	// Open browser to Bilibili login page
	bilibiliLoginURL := "https://passport.bilibili.com/login"

	fmt.Printf("正在打开浏览器到: %s\n", bilibiliLoginURL)

	if err := openBrowser(bilibiliLoginURL); err != nil {
		logger.Warnf("Failed to open browser: %v", err)
		fmt.Printf("请手动打开浏览器访问: %s\n", bilibiliLoginURL)
	}

	fmt.Println()
	fmt.Println("请在浏览器中完成登录，然后按照以下步骤获取Cookie：")
	fmt.Println()
	fmt.Println("1. 登录成功后，按F12打开开发者工具")
	fmt.Println("2. 切换到 'Application' 或 '存储' 标签页")
	fmt.Println("3. 在左侧找到 'Cookies' -> 'https://www.bilibili.com'")
	fmt.Println("4. 找到以下Cookie并复制其值：")
	fmt.Println("   - SESSDATA")
	fmt.Println("   - bili_jct")
	fmt.Println("   - DedeUserID")
	fmt.Println("   - DedeUserID__ckMd5")
	fmt.Println("   - sid")
	fmt.Println("   - buvid3")
	fmt.Println("   - buvid4")
	fmt.Println()
	fmt.Println("5. 将Cookie保存为文本文件，格式如下：")
	fmt.Println("   SESSDATA	你的SESSDATA值")
	fmt.Println("   bili_jct	你的bili_jct值")
	fmt.Println("   DedeUserID	你的DedeUserID值")
	fmt.Println("   ...")
	fmt.Println()
	fmt.Println("6. 保存文件后，使用以下命令导入Cookie：")
	fmt.Println("   ./goBili login -c 你的cookie文件路径")
	fmt.Println()

	// Wait for user to complete the process
	fmt.Print("按回车键继续，或输入 'q' 退出: ")
	var input string
	fmt.Scanln(&input)

	if input == "q" || input == "Q" {
		return fmt.Errorf("用户取消登录")
	}

	return fmt.Errorf("请按照上述步骤获取Cookie，然后使用 -c 参数导入")
}

// openBrowser opens the specified URL in the default browser
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}
