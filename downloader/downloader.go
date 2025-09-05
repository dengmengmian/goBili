package downloader

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"goBili/parser"

	"github.com/sirupsen/logrus"
)

// Config holds downloader configuration
type Config struct {
	OutputDir   string
	Threads     int
	Verbose     bool
	Quality     string
	Format      string
	AudioOnly   bool
	VideoOnly   bool
	AuthManager interface{} // Will be cast to *auth.AuthManager when needed
}

// Downloader handles video downloading
type Downloader struct {
	config Config
	logger *logrus.Logger
	client *http.Client
}

// DownloadProgress represents download progress information
type DownloadProgress struct {
	TotalSize  int64
	Downloaded int64
	Percentage float64
	Speed      int64
	ETA        time.Duration
}

// NewDownloader creates a new downloader instance
func NewDownloader(config Config) *Downloader {
	logger := logrus.New()
	if config.Verbose {
		logger.SetLevel(logrus.DebugLevel)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}

	return &Downloader{
		config: config,
		logger: logger,
		client: &http.Client{
			Timeout: 0, // No timeout for downloads
		},
	}
}

// GetVideoStreams fetches available video streams for a video
func (d *Downloader) GetVideoStreams(videoInfo *parser.VideoInfo) ([]*parser.StreamInfo, error) {
	// This method is now handled by the parser
	// We'll create a parser instance to get the streams
	// In a real implementation, you might want to pass the parser as a dependency

	// For now, return an error to indicate this should be handled by the parser
	return nil, fmt.Errorf("GetVideoStreams should be called on the parser, not the downloader")
}

// DownloadVideo downloads a video using the specified streams
func (d *Downloader) DownloadVideo(videoInfo *parser.VideoInfo, streams []*parser.StreamInfo) error {
	// Select the appropriate stream based on quality preference
	stream := d.selectStream(streams)
	if stream == nil {
		return fmt.Errorf("no suitable stream found")
	}

	d.logger.Infof("Selected stream: %s (%s)", stream.Resolution, stream.Format)

	// Generate output filename
	filename := d.generateFilename(videoInfo, stream)
	outputPath := filepath.Join(d.config.OutputDir, filename)

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Download based on configuration
	if d.config.AudioOnly {
		return d.downloadAudio(stream, outputPath)
	} else if d.config.VideoOnly {
		return d.downloadVideoOnly(stream, outputPath)
	} else {
		return d.downloadVideoAndAudio(stream, outputPath)
	}
}

// selectStream selects the appropriate stream based on quality preference
func (d *Downloader) selectStream(streams []*parser.StreamInfo) *parser.StreamInfo {
	if len(streams) == 0 {
		return nil
	}

	// Quality mapping
	qualityMap := map[string]int{
		"best":  80,
		"1080p": 80,
		"720p":  64,
		"480p":  32,
		"360p":  16,
	}

	targetQuality, exists := qualityMap[d.config.Quality]
	if !exists {
		// Default to best quality
		targetQuality = 80
	}

	// Find exact quality match
	for _, stream := range streams {
		if stream.Quality == targetQuality {
			return stream
		}
	}

	// If exact quality not found, return the best available
	best := streams[0]
	for _, stream := range streams[1:] {
		if stream.Quality > best.Quality {
			best = stream
		}
	}

	return best
}

// generateFilename generates a filename for the downloaded video
func (d *Downloader) generateFilename(videoInfo *parser.VideoInfo, stream *parser.StreamInfo) string {
	// Clean the title for use as filename
	title := strings.ReplaceAll(videoInfo.Title, "/", "_")
	title = strings.ReplaceAll(title, "\\", "_")
	title = strings.ReplaceAll(title, ":", "_")
	title = strings.ReplaceAll(title, "*", "_")
	title = strings.ReplaceAll(title, "?", "_")
	title = strings.ReplaceAll(title, "\"", "_")
	title = strings.ReplaceAll(title, "<", "_")
	title = strings.ReplaceAll(title, ">", "_")
	title = strings.ReplaceAll(title, "|", "_")

	// Truncate if too long
	if len(title) > 100 {
		title = title[:100]
	}

	// Add quality suffix
	qualitySuffix := ""
	switch stream.Quality {
	case 80:
		qualitySuffix = "_1080p"
	case 64:
		qualitySuffix = "_720p"
	case 32:
		qualitySuffix = "_480p"
	case 16:
		qualitySuffix = "_360p"
	}

	return fmt.Sprintf("%s%s.%s", title, qualitySuffix, d.config.Format)
}

// downloadAudio downloads only the audio stream
func (d *Downloader) downloadAudio(stream *parser.StreamInfo, outputPath string) error {
	d.logger.Info("Downloading audio...")

	// Change extension to audio format
	outputPath = strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".m4a"

	return d.downloadFile(stream.AudioURL, outputPath)
}

// downloadVideoOnly downloads only the video stream
func (d *Downloader) downloadVideoOnly(stream *parser.StreamInfo, outputPath string) error {
	d.logger.Info("Downloading video...")

	// Change extension to video format
	outputPath = strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".mp4"

	return d.downloadFile(stream.VideoURL, outputPath)
}

// downloadVideoAndAudio downloads both video and audio streams
func (d *Downloader) downloadVideoAndAudio(stream *parser.StreamInfo, outputPath string) error {
	d.logger.Info("Downloading video and audio...")

	// For simplicity, we'll download them separately and then merge
	// In a real implementation, you would use ffmpeg to merge them

	videoPath := strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + "_video.mp4"
	audioPath := strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + "_audio.m4a"

	// Download video and audio concurrently
	var wg sync.WaitGroup
	var videoErr, audioErr error

	wg.Add(2)

	go func() {
		defer wg.Done()
		videoErr = d.downloadFile(stream.VideoURL, videoPath)
	}()

	go func() {
		defer wg.Done()
		audioErr = d.downloadFile(stream.AudioURL, audioPath)
	}()

	wg.Wait()

	if videoErr != nil {
		return fmt.Errorf("failed to download video: %v", videoErr)
	}
	if audioErr != nil {
		return fmt.Errorf("failed to download audio: %v", audioErr)
	}

	// For now, just copy the video file as the final output
	// In a real implementation, you would merge video and audio using ffmpeg
	return d.mergeVideoAndAudio(videoPath, audioPath, outputPath)
}

// downloadFile downloads a file from URL to local path
func (d *Downloader) downloadFile(url, outputPath string) error {
	d.logger.Debugf("Downloading %s to %s", url, outputPath)

	// Create the output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer file.Close()

	// Make authenticated HTTP request if auth manager is available
	var resp *http.Response
	if d.config.AuthManager != nil {
		// Cast to auth manager and create authenticated request
		if authManager, ok := d.config.AuthManager.(interface {
			CreateAuthenticatedRequest(method, url string, body io.Reader) (*http.Request, error)
		}); ok {
			req, err := authManager.CreateAuthenticatedRequest("GET", url, nil)
			if err != nil {
				return fmt.Errorf("failed to create authenticated request: %v", err)
			}
			resp, err = d.client.Do(req)
		} else {
			// Fallback to regular request
			resp, err = d.client.Get(url)
		}
	} else {
		// Regular HTTP request
		resp, err = d.client.Get(url)
	}

	if err != nil {
		return fmt.Errorf("failed to make HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode)
	}

	// Get content length for progress tracking
	totalSize := resp.ContentLength
	if totalSize > 0 {
		d.logger.Infof("File size: %.2f MB", float64(totalSize)/(1024*1024))
	}

	// Create a progress reader for better user experience
	progressReader := &ProgressReader{
		Reader:   resp.Body,
		Total:    totalSize,
		Progress: nil, // No progress channel for simple downloads
	}

	// Copy the response body to the file with progress tracking
	_, err = io.Copy(file, progressReader)
	if err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	d.logger.Infof("Successfully downloaded: %s", outputPath)
	return nil
}

// mergeVideoAndAudio merges video and audio files using ffmpeg
func (d *Downloader) mergeVideoAndAudio(videoPath, audioPath, outputPath string) error {
	d.logger.Info("Merging video and audio...")

	// Check if ffmpeg is available
	if !d.isFFmpegAvailable() {
		d.logger.Warn("ffmpeg not found, copying video file only (no audio)")
		// Fallback: just copy the video file
		return d.copyFile(videoPath, outputPath)
	}

	// Use ffmpeg to merge video and audio
	cmd := exec.Command("ffmpeg",
		"-i", videoPath, // Input video
		"-i", audioPath, // Input audio
		"-c:v", "copy", // Copy video stream without re-encoding
		"-c:a", "aac", // Encode audio to AAC
		"-map", "0:v:0", // Map video from first input
		"-map", "1:a:0", // Map audio from second input
		"-y",       // Overwrite output file
		outputPath, // Output file
	)

	// Set up command output
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	d.logger.Debugf("Running ffmpeg command: %s", strings.Join(cmd.Args, " "))

	// Execute ffmpeg command
	err := cmd.Run()
	if err != nil {
		d.logger.Errorf("ffmpeg failed: %v", err)
		// Fallback: just copy the video file
		d.logger.Warn("Falling back to video-only output")
		return d.copyFile(videoPath, outputPath)
	}

	// Clean up temporary files
	os.Remove(videoPath)
	os.Remove(audioPath)

	d.logger.Infof("Successfully merged: %s", outputPath)
	return nil
}

// isFFmpegAvailable checks if ffmpeg is available in the system
func (d *Downloader) isFFmpegAvailable() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

// copyFile copies a file from src to dst
func (d *Downloader) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %v", err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %v", err)
	}

	return nil
}

// DownloadWithProgress downloads a file with progress reporting
func (d *Downloader) DownloadWithProgress(url, outputPath string, progressChan chan<- DownloadProgress) error {
	// Create the output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer file.Close()

	// Make HTTP request
	resp, err := d.client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to make HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode)
	}

	// Get content length
	totalSize := resp.ContentLength
	if totalSize == -1 {
		totalSize = 0 // Unknown size
	}

	// Create a progress reader
	progressReader := &ProgressReader{
		Reader:   resp.Body,
		Total:    totalSize,
		Progress: progressChan,
	}

	// Copy with progress
	_, err = io.Copy(file, progressReader)
	if err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	return nil
}

// ProgressReader wraps an io.Reader to report progress
type ProgressReader struct {
	Reader    io.Reader
	Total     int64
	Progress  chan<- DownloadProgress
	ReadBytes int64
	LastTime  time.Time
}

func (pr *ProgressReader) Read(p []byte) (n int, err error) {
	n, err = pr.Reader.Read(p)
	pr.ReadBytes += int64(n)

	// Show progress every 1MB or when complete
	if pr.Total > 0 && (pr.ReadBytes%1024*1024 == 0 || err != nil) {
		percentage := float64(pr.ReadBytes) / float64(pr.Total) * 100
		fmt.Printf("\rDownloading: %.1f%% (%.2f/%.2f MB)",
			percentage,
			float64(pr.ReadBytes)/(1024*1024),
			float64(pr.Total)/(1024*1024))
	}

	if pr.Progress != nil {
		now := time.Now()
		if pr.LastTime.IsZero() {
			pr.LastTime = now
		}

		progress := DownloadProgress{
			TotalSize:  pr.Total,
			Downloaded: pr.ReadBytes,
		}

		if pr.Total > 0 {
			progress.Percentage = float64(pr.ReadBytes) / float64(pr.Total) * 100
		}

		// Calculate speed
		elapsed := now.Sub(pr.LastTime)
		if elapsed > 0 {
			progress.Speed = int64(float64(pr.ReadBytes) / elapsed.Seconds())

			// Calculate ETA
			if progress.Speed > 0 && pr.Total > 0 {
				remaining := pr.Total - pr.ReadBytes
				progress.ETA = time.Duration(remaining/progress.Speed) * time.Second
			}
		}

		select {
		case pr.Progress <- progress:
		default:
		}
	}

	return n, err
}
