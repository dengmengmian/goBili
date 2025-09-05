package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"goBili/auth"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// logoutCmd represents the logout command
var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear current login session and remove saved cookies",
	Long: `Logout from Bilibili by clearing all saved authentication cookies.
This will remove the current login session and require re-authentication for future downloads.`,
	RunE: runLogout,
}

func init() {
	rootCmd.AddCommand(logoutCmd)

	// Add flag for force logout without confirmation
	logoutCmd.Flags().BoolP("force", "f", false, "force logout without confirmation")
}

func runLogout(cmd *cobra.Command, args []string) error {
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

	// Load existing cookies to check if logged in
	if err := authManager.LoadCookies(); err != nil {
		logger.Debugf("Failed to load cookies: %v", err)
	}

	// Check if currently logged in
	if !authManager.IsAuthenticated() {
		fmt.Println("No active login session found.")
		return nil
	}

	// Get user info before logout
	userInfo, err := authManager.GetUserInfo()
	if err != nil {
		logger.Warnf("Failed to get user info: %v", err)
		fmt.Println("Currently logged in (user info unavailable)")
	} else {
		fmt.Printf("Currently logged in as: %s (UID: %d)\n", userInfo.Name, userInfo.Mid)
	}

	// Check for force flag
	force, _ := cmd.Flags().GetBool("force")

	if !force {
		// Ask for confirmation
		fmt.Print("Are you sure you want to logout? (y/N): ")
		var input string
		fmt.Scanln(&input)

		if input != "y" && input != "Y" && input != "yes" && input != "Yes" {
			fmt.Println("Logout cancelled.")
			return nil
		}
	}

	// Remove cookie file
	cookieFile := filepath.Join(configDir, "cookies.json")
	if _, err := os.Stat(cookieFile); err == nil {
		if err := os.Remove(cookieFile); err != nil {
			return fmt.Errorf("failed to remove cookie file: %v", err)
		}
		fmt.Println("✓ Cookie file removed")
	} else {
		fmt.Println("✓ No cookie file found")
	}

	// Clear in-memory cookies
	authManager.ClearCookies()

	fmt.Println("✓ Login session cleared")
	fmt.Println("You will need to login again to download videos.")

	return nil
}
