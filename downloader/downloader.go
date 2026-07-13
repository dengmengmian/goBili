// Package downloader downloads Bilibili video and audio streams.
// It handles DASH stream merging (with or without ffmpeg), progress
// reporting, safe filename generation, and configurable quality selection.
package downloader

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/dengmengmian/goBili/parser"

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

	// Transport with sensible timeouts to prevent hanging connections.
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   15 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConns:          10,
		IdleConnTimeout:       90 * time.Second,
	}

	return &Downloader{
		config: config,
		logger: logger,
		client: &http.Client{
			Transport: transport,
			Timeout:   0, // No global timeout; per-operation deadlines are handled via context.
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
	ctx := context.Background()
	return d.DownloadVideoContext(ctx, videoInfo, streams)
}

// DownloadVideoContext downloads a video with context support for cancellation.
func (d *Downloader) DownloadVideoContext(ctx context.Context, videoInfo *parser.VideoInfo, streams []*parser.StreamInfo) error {
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

	// Check context before starting downloads.
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Download based on configuration
	if d.config.AudioOnly {
		return d.downloadAudio(ctx, stream, outputPath)
	} else if d.config.VideoOnly {
		return d.downloadVideoOnly(ctx, stream, outputPath)
	} else {
		return d.downloadVideoAndAudio(ctx, stream, outputPath)
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
	title := sanitizeFilename(videoInfo.Title)

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

// sanitizeFilename cleans a string to be a safe filename component.
// It removes path separators, control characters, and other unsafe runes,
// truncates to a reasonable length, and ensures the result is not empty
// and does not resolve to a parent directory.
func sanitizeFilename(name string) string {
	// Replace known dangerous characters with underscores.
	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_", "?", "_",
		"\"", "_", "<", "_", ">", "_", "|", "_",
	)
	clean := replacer.Replace(name)

	// Remove any remaining control or non-printable characters.
	clean = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || !unicode.IsPrint(r) {
			return '_'
		}
		return r
	}, clean)

	// Strip leading/trailing spaces and dots (problematic on Windows).
	clean = strings.Trim(clean, " .")

	// Ensure the result is not empty.
	if clean == "" {
		clean = "video"
	}

	// Truncate to a reasonable maximum length.
	const maxLen = 200
	cleanRunes := []rune(clean)
	if len(cleanRunes) > maxLen {
		clean = string(cleanRunes[:maxLen])
	}

	// Prevent path traversal: use only the base name.
	clean = filepath.Base(clean)
	if clean == "." || clean == ".." {
		clean = "video"
	}

	return clean
}

// downloadAudio downloads only the audio stream
func (d *Downloader) downloadAudio(ctx context.Context, stream *parser.StreamInfo, outputPath string) error {
	d.logger.Info("Downloading audio...")

	// Change extension to audio format
	outputPath = strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".m4a"

	return d.downloadFile(ctx, stream.AudioURL, outputPath)
}

// downloadVideoOnly downloads only the video stream
func (d *Downloader) downloadVideoOnly(ctx context.Context, stream *parser.StreamInfo, outputPath string) error {
	d.logger.Info("Downloading video...")

	// Change extension to video format
	outputPath = strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".mp4"

	return d.downloadFile(ctx, stream.VideoURL, outputPath)
}

// downloadVideoAndAudio downloads both video and audio streams
func (d *Downloader) downloadVideoAndAudio(ctx context.Context, stream *parser.StreamInfo, outputPath string) error {
	d.logger.Info("Downloading video and audio...")

	// For simplicity, we'll download them separately and then merge
	// In a real implementation, you would use ffmpeg to merge them

	videoPath := strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + "_video.mp4"
	audioPath := strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + "_audio.m4a"

	// Download video and audio concurrently with context.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	var videoErr, audioErr error

	wg.Add(2)

	go func() {
		defer wg.Done()
		videoErr = d.downloadFile(ctx, stream.VideoURL, videoPath)
		if videoErr != nil {
			cancel() // Cancel audio download if video fails.
		}
	}()

	go func() {
		defer wg.Done()
		audioErr = d.downloadFile(ctx, stream.AudioURL, audioPath)
		if audioErr != nil {
			cancel() // Cancel video download if audio fails.
		}
	}()

	wg.Wait()

	if videoErr != nil {
		os.Remove(audioPath) // Clean up partial audio.
		return fmt.Errorf("failed to download video: %v", videoErr)
	}
	if audioErr != nil {
		os.Remove(videoPath) // Clean up partial video.
		return fmt.Errorf("failed to download audio: %v", audioErr)
	}

	// For now, just copy the video file as the final output
	// In a real implementation, you would merge video and audio using ffmpeg
	return d.mergeVideoAndAudio(videoPath, audioPath, outputPath)
}

// downloadFile downloads a file from URL to local path
func (d *Downloader) downloadFile(ctx context.Context, url, outputPath string) error {
	d.logger.Debugf("Downloading %s to %s", url, outputPath)

	// Use chunked download when threads > 1 and server supports Range.
	if d.config.Threads > 1 {
		supportsRange, contentLength, err := d.checkRangeSupport(ctx, url)
		if err == nil && supportsRange && contentLength > 0 {
			d.logger.Infof("Using chunked download with %d threads (%.2f MB)",
				d.config.Threads, float64(contentLength)/(1024*1024))
			return d.downloadFileChunked(ctx, url, outputPath, contentLength)
		}
		d.logger.Debug("Range not supported, falling back to single-threaded download")
	}

	return d.downloadFileSingle(ctx, url, outputPath)
}

// downloadFileSingle downloads a file with a single HTTP request and retry support.
func (d *Downloader) downloadFileSingle(ctx context.Context, url, outputPath string) error {
	// Create the output file.
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer file.Close()

	cfg := defaultRetryConfig()

	return retry(ctx, cfg, func() (int, error) {
		// Build the request.
		var req *http.Request
		var reqErr error

		if d.config.AuthManager != nil {
			if authManager, ok := d.config.AuthManager.(interface {
				CreateAuthenticatedRequest(method, url string, body io.Reader) (*http.Request, error)
			}); ok {
				req, reqErr = authManager.CreateAuthenticatedRequest("GET", url, nil)
			}
		}
		if req == nil && reqErr == nil {
			req, reqErr = http.NewRequestWithContext(ctx, "GET", url, nil)
		}
		if reqErr != nil {
			return 0, fmt.Errorf("failed to create request: %v", reqErr)
		}

		req = req.WithContext(ctx)

		resp, err := d.client.Do(req)
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return resp.StatusCode, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
		}

		totalSize := resp.ContentLength
		if totalSize > 0 {
			d.logger.Infof("File size: %.2f MB", float64(totalSize)/(1024*1024))
		}

		if err := file.Truncate(0); err != nil {
			return 0, fmt.Errorf("failed to truncate file: %v", err)
		}
		if _, err := file.Seek(0, 0); err != nil {
			return 0, fmt.Errorf("failed to seek file: %v", err)
		}

		progressReader := &ProgressReader{
			Reader:   resp.Body,
			Total:    totalSize,
			Progress: nil, // No progress channel for simple downloads
		}

		if _, err := io.Copy(file, progressReader); err != nil {
			return 0, fmt.Errorf("failed to write file: %v", err)
		}

		d.logger.Infof("Successfully downloaded: %s", outputPath)
		return resp.StatusCode, nil
	})
}

// checkRangeSupport checks if the server supports HTTP Range requests.
// It returns (supportsRange, contentLength, error).
func (d *Downloader) checkRangeSupport(ctx context.Context, url string) (bool, int64, error) {
	req, reqErr := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if reqErr != nil {
		return false, 0, reqErr
	}

	// Try to use auth if available.
	if d.config.AuthManager != nil {
		if authManager, ok := d.config.AuthManager.(interface {
			CreateAuthenticatedRequest(method, url string, body io.Reader) (*http.Request, error)
		}); ok {
			if authReq, err := authManager.CreateAuthenticatedRequest("HEAD", url, nil); err == nil {
				authReq = authReq.WithContext(ctx)
				req = authReq
			}
		}
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return false, 0, err
	}
	defer resp.Body.Close()

	acceptRanges := resp.Header.Get("Accept-Ranges")
	supportsRange := strings.EqualFold(acceptRanges, "bytes")

	return supportsRange, resp.ContentLength, nil
}

// downloadFileChunked downloads a file using concurrent chunked Range requests.
func (d *Downloader) downloadFileChunked(ctx context.Context, url string, outputPath string, contentLength int64) error {
	numThreads := d.config.Threads
	if numThreads < 1 {
		numThreads = 1
	}
	if numThreads > 16 {
		numThreads = 16 // Cap to avoid excessive connections.
	}

	chunkSize := contentLength / int64(numThreads)
	if chunkSize < 1024*1024 {
		// File too small for chunking; fall back to single-threaded.
		return d.downloadFileSingle(ctx, url, outputPath)
	}

	// Create the output file and pre-allocate it.
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer file.Close()

	if err := file.Truncate(contentLength); err != nil {
		return fmt.Errorf("failed to pre-allocate file: %v", err)
	}

	// Download chunks concurrently.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	errs := make(chan error, numThreads)

	for i := 0; i < numThreads; i++ {
		start := int64(i) * chunkSize
		end := start + chunkSize - 1
		if i == numThreads-1 {
			end = contentLength - 1 // Last chunk gets the remainder.
		}

		wg.Add(1)
		go func(chunkStart, chunkEnd int64) {
			defer wg.Done()
			if err := d.downloadChunk(ctx, url, file, chunkStart, chunkEnd); err != nil {
				errs <- fmt.Errorf("chunk %d-%d: %w", chunkStart, chunkEnd, err)
				cancel()
			}
		}(start, end)
	}

	wg.Wait()
	close(errs)

	// Collect the first error, if any.
	for err := range errs {
		d.logger.Errorf("Chunk download failed: %v", err)
		// Return the first error; partial file will be cleaned up by caller.
		return err
	}

	d.logger.Infof("Successfully downloaded: %s", outputPath)
	return nil
}

// downloadChunk downloads a single byte range to the file at the given offset.
func (d *Downloader) downloadChunk(ctx context.Context, url string, file *os.File, start, end int64) error {
	cfg := defaultRetryConfig()

	return retry(ctx, cfg, func() (int, error) {
		var req *http.Request
		var reqErr error

		if d.config.AuthManager != nil {
			if authManager, ok := d.config.AuthManager.(interface {
				CreateAuthenticatedRequest(method, url string, body io.Reader) (*http.Request, error)
			}); ok {
				req, reqErr = authManager.CreateAuthenticatedRequest("GET", url, nil)
			}
		}
		if req == nil && reqErr == nil {
			req, reqErr = http.NewRequestWithContext(ctx, "GET", url, nil)
		}
		if reqErr != nil {
			return 0, fmt.Errorf("failed to create request: %v", reqErr)
		}

		req = req.WithContext(ctx)
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

		resp, err := d.client.Do(req)
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
			return resp.StatusCode, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
		}

		// Read the chunk into memory.
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return 0, fmt.Errorf("failed to read chunk: %v", err)
		}

		// Write chunk at the correct offset.
		if _, err := file.WriteAt(data, start); err != nil {
			return 0, fmt.Errorf("failed to write chunk at offset %d: %v", start, err)
		}

		return resp.StatusCode, nil
	})
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
	if err := os.Remove(videoPath); err != nil {
		d.logger.Warnf("failed to remove temporary video file %s: %v", videoPath, err)
	}
	if err := os.Remove(audioPath); err != nil {
		d.logger.Warnf("failed to remove temporary audio file %s: %v", audioPath, err)
	}

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
func (d *Downloader) DownloadWithProgress(ctx context.Context, url, outputPath string, progressChan chan<- DownloadProgress) error {
	// Create the output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer file.Close()

	cfg := defaultRetryConfig()

	return retry(ctx, cfg, func() (int, error) {
		// Reset file for retry.
		if err := file.Truncate(0); err != nil {
			return 0, fmt.Errorf("failed to truncate file: %v", err)
		}
		if _, err := file.Seek(0, 0); err != nil {
			return 0, fmt.Errorf("failed to seek file: %v", err)
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return 0, fmt.Errorf("failed to create request: %v", err)
		}

		resp, err := d.client.Do(req)
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return resp.StatusCode, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
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
		if _, err := io.Copy(file, progressReader); err != nil {
			return 0, fmt.Errorf("failed to write file: %v", err)
		}

		return resp.StatusCode, nil
	})
}

// ProgressReader wraps an io.Reader to report progress
type ProgressReader struct {
	Reader    io.Reader
	Total     int64
	Progress  chan<- DownloadProgress
	ReadBytes int64
	startTime time.Time
	lastTime  time.Time
	lastBytes int64
}

func (pr *ProgressReader) Read(p []byte) (n int, err error) {
	n, err = pr.Reader.Read(p)
	pr.ReadBytes += int64(n)

	now := time.Now()
	if pr.startTime.IsZero() {
		pr.startTime = now
		pr.lastTime = now
	}

	// Emit progress periodically: every 500ms or on completion/error.
	sinceLast := now.Sub(pr.lastTime)
	if sinceLast >= 500*time.Millisecond || err != nil {
		progress := DownloadProgress{
			TotalSize:  pr.Total,
			Downloaded: pr.ReadBytes,
		}

		if pr.Total > 0 {
			progress.Percentage = float64(pr.ReadBytes) / float64(pr.Total) * 100
		}

		// Windowed speed (bytes since last emission / time since last emission).
		bytesDelta := pr.ReadBytes - pr.lastBytes
		timeDelta := now.Sub(pr.lastTime)
		if timeDelta > 0 {
			progress.Speed = int64(float64(bytesDelta) / timeDelta.Seconds())
		}

		// ETA based on windowed speed.
		if progress.Speed > 0 && pr.Total > 0 {
			remaining := pr.Total - pr.ReadBytes
			progress.ETA = time.Duration(remaining/progress.Speed) * time.Second
		}

		pr.lastTime = now
		pr.lastBytes = pr.ReadBytes

		// Print progress to stdout for basic progress display.
		// (A proper progress bar library replaces this in a follow-up.)
		if pr.Total > 0 {
			fmt.Printf("\rDownloading: %.1f%% (%.2f/%.2f MB) %s/s",
				progress.Percentage,
				float64(pr.ReadBytes)/(1024*1024),
				float64(pr.Total)/(1024*1024),
				formatSpeed(progress.Speed))
		} else {
			fmt.Printf("\rDownloading: %.2f MB %s/s",
				float64(pr.ReadBytes)/(1024*1024),
				formatSpeed(progress.Speed))
		}

		if pr.Progress != nil {
			select {
			case pr.Progress <- progress:
			default:
			}
		}
	}

	return n, err
}

// formatSpeed returns a human-readable speed string.
func formatSpeed(bytesPerSec int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case bytesPerSec >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytesPerSec)/float64(GB))
	case bytesPerSec >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytesPerSec)/float64(MB))
	case bytesPerSec >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytesPerSec)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytesPerSec)
	}
}
