package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"goBili/auth"
	"goBili/downloader"
	"goBili/parser"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// downloadCmd represents the download command
var downloadCmd = &cobra.Command{
	Use:   "download [URL]",
	Short: "Download a Bilibili video or playlist",
	Long: `Download a single video or entire playlist from Bilibili.
Supports both single video URLs and playlist URLs.

Examples:
  goBili download "https://www.bilibili.com/video/BV1qt4y1X7TW"
  goBili download "https://www.bilibili.com/bangumi/play/ss33073"`,
	Args: cobra.ExactArgs(1),
	RunE: runDownload,
}

func init() {
	rootCmd.AddCommand(downloadCmd)

	// Local flags for download command
	downloadCmd.Flags().StringP("quality", "q", "best", "video quality (best, 1080p, 720p, 480p, 360p)")
	downloadCmd.Flags().StringP("format", "f", "mp4", "output format (mp4, flv)")
	downloadCmd.Flags().BoolP("audio-only", "a", false, "download audio only")
	downloadCmd.Flags().Bool("video-only", false, "download video only")
	downloadCmd.Flags().StringP("pages", "p", "all", "specific pages to download (e.g., 1,2,3 or 1-5 or all)")
}

func runDownload(cmd *cobra.Command, args []string) error {
	url := args[0]

	// Get configuration
	outputDir := viper.GetString("output")
	threads := viper.GetInt("threads")
	verbose := viper.GetBool("verbose")

	quality, _ := cmd.Flags().GetString("quality")
	format, _ := cmd.Flags().GetString("format")
	audioOnly, _ := cmd.Flags().GetBool("audio-only")
	videoOnly, _ := cmd.Flags().GetBool("video-only")
	pages, _ := cmd.Flags().GetString("pages")

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Initialize logger
	logger := logrus.New()
	if verbose {
		logger.SetLevel(logrus.DebugLevel)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}

	// Initialize auth manager
	configDir := getConfigDir()
	authManager := auth.NewAuthManager(configDir, logger)

	// Load existing cookies
	if err := authManager.LoadCookies(); err != nil {
		logger.Warnf("Failed to load cookies: %v", err)
	}

	// Check authentication
	if !authManager.IsAuthenticated() {
		fmt.Println("Not authenticated. Please login first using: goBili login")
		return fmt.Errorf("authentication required")
	}

	// Initialize parser with auth manager
	p := parser.NewBilibiliParser(authManager, logger)

	// Parse URL to determine if it's a single video or playlist
	videoInfo, err := p.ParseURL(url)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %v", err)
	}

	// Initialize downloader
	dl := downloader.NewDownloader(downloader.Config{
		OutputDir:   outputDir,
		Threads:     threads,
		Verbose:     verbose,
		Quality:     quality,
		Format:      format,
		AudioOnly:   audioOnly,
		VideoOnly:   videoOnly,
		AuthManager: authManager,
	})

	// Handle different types of content
	switch videoInfo.Type {
	case "video":
		return downloadSingleVideo(p, dl, videoInfo, pages)
	case "playlist":
		return downloadPlaylist(p, dl, videoInfo, pages)
	default:
		return fmt.Errorf("unsupported content type: %s", videoInfo.Type)
	}
}

func downloadSingleVideo(p *parser.BilibiliParser, dl *downloader.Downloader, videoInfo *parser.VideoInfo, pages string) error {
	fmt.Printf("Downloading video: %s\n", videoInfo.Title)

	// Check if this is actually a multi-part video that was misclassified
	if len(videoInfo.Pages) > 1 {
		fmt.Printf("Detected multi-part video with %d parts\n", len(videoInfo.Pages))
		return downloadPlaylist(p, dl, videoInfo, pages)
	}

	// Get video streams using parser
	streams, err := p.GetVideoStreams(videoInfo)
	if err != nil {
		return fmt.Errorf("failed to get video streams: %v", err)
	}

	// Download the video
	return dl.DownloadVideo(videoInfo, streams)
}

func downloadPlaylist(p *parser.BilibiliParser, dl *downloader.Downloader, videoInfo *parser.VideoInfo, pages string) error {
	fmt.Printf("Downloading playlist: %s (%d episodes)\n", videoInfo.Title, len(videoInfo.Episodes))

	// Parse pages parameter
	var episodesToDownload []*parser.EpisodeInfo
	if pages == "all" {
		episodesToDownload = videoInfo.Episodes
	} else {
		// Parse specific pages (e.g., "1,2,3" or "1-5")
		indices, err := parsePageRange(pages, len(videoInfo.Episodes))
		if err != nil {
			return fmt.Errorf("invalid pages parameter: %v", err)
		}

		for _, idx := range indices {
			if idx > 0 && idx <= len(videoInfo.Episodes) {
				episodesToDownload = append(episodesToDownload, videoInfo.Episodes[idx-1])
			}
		}
	}

	// Download each episode
	for i, episode := range episodesToDownload {
		fmt.Printf("\n[%d/%d] Downloading: %s\n", i+1, len(episodesToDownload), episode.Title)

		// Create episode info with original video info and pages
		episodeVideoInfo := &parser.VideoInfo{
			BVID:  episode.BVID,
			Title: episode.Title,
			Type:  "video",
			Pages: videoInfo.Pages, // Include the original pages info
		}

		// Get video streams using parser for the specific page
		streams, err := p.GetVideoStreamsForPage(episodeVideoInfo, episode.Index)
		if err != nil {
			fmt.Printf("Failed to get streams for episode %s: %v\n", episode.Title, err)
			continue
		}

		// Download the episode
		if err := dl.DownloadVideo(episodeVideoInfo, streams); err != nil {
			fmt.Printf("Failed to download episode %s: %v\n", episode.Title, err)
			continue
		}
	}

	fmt.Printf("\nPlaylist download completed!\n")
	return nil
}

func parsePageRange(pages string, maxPages int) ([]int, error) {
	var indices []int

	// Handle comma-separated values
	parts := strings.Split(pages, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)

		// Handle range (e.g., "1-5")
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid range format: %s", part)
			}

			start, err := parseInt(rangeParts[0])
			if err != nil {
				return nil, fmt.Errorf("invalid start page: %s", rangeParts[0])
			}

			end, err := parseInt(rangeParts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid end page: %s", rangeParts[1])
			}

			if start > end {
				return nil, fmt.Errorf("start page (%d) cannot be greater than end page (%d)", start, end)
			}

			for i := start; i <= end; i++ {
				indices = append(indices, i)
			}
		} else {
			// Handle single page
			page, err := parseInt(part)
			if err != nil {
				return nil, fmt.Errorf("invalid page number: %s", part)
			}
			indices = append(indices, page)
		}
	}

	return indices, nil
}

func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

// getConfigDir returns the configuration directory
func getConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		return "."
	}
	return filepath.Join(home, ".goBili")
}
