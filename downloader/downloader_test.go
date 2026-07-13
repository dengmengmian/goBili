package downloader

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/dengmengmian/goBili/parser"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple ascii with spaces preserved", "My Video Title", "My Video Title"},
		{"slash replaced", "video/name", "video_name"},
		{"backslash replaced", "video\\name", "video_name"},
		{"colon replaced", "video: name", "video_ name"},
		{"asterisk replaced", "video*name", "video_name"},
		{"question replaced", "video?name", "video_name"},
		{"quote replaced", `video"name`, "video_name"},
		{"angle brackets replaced", "video<name>", "video_name_"},
		{"pipe replaced", "video|name", "video_name"},
		// Path traversal prevention
		{"dots resolved", "..", "video"},
		{"dot resolved", ".", "video"},
		{"path traversal attempt", "../../etc/passwd", "_.._etc_passwd"},
		{"absolute unix path", "/etc/passwd", "_etc_passwd"},
		// Edge cases
		{"empty string", "", "video"},
		{"only spaces and dots", " . . ", "video"},
		{"leading trailing stripped", "  hello  ", "hello"},
		{"dots at edges", "...hello...", "hello"},
		// Truncation
		{"very long name", strings.Repeat("a", 300), strings.Repeat("a", 200)},
		// Unicode
		{"chinese title", "测试视频", "测试视频"},
		{"mixed unicode", "テスト動画", "テスト動画"},
		// Null byte (should be replaced)
		{"null byte", "video\x00name", "video_name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGenerateFilename(t *testing.T) {
	d := &Downloader{
		config: Config{Format: "mp4"},
	}

	info := &parser.VideoInfo{Title: "My Great Video"}
	stream := &parser.StreamInfo{Quality: 80}

	got := d.generateFilename(info, stream)
	want := "My Great Video_1080p.mp4"
	if got != want {
		t.Errorf("generateFilename() = %q, want %q", got, want)
	}

	// Test with 720p quality.
	stream.Quality = 64
	got = d.generateFilename(info, stream)
	want = "My Great Video_720p.mp4"
	if got != want {
		t.Errorf("generateFilename() = %q, want %q", got, want)
	}

	// Test with unknown quality (no suffix).
	stream.Quality = 999
	got = d.generateFilename(info, stream)
	want = "My Great Video.mp4"
	if got != want {
		t.Errorf("generateFilename() = %q, want %q", got, want)
	}
}

func TestSelectStream(t *testing.T) {
	d := &Downloader{
		config: Config{Quality: "720p"},
	}

	streams := []*parser.StreamInfo{
		{Quality: 16, Resolution: "360p"},
		{Quality: 32, Resolution: "480p"},
		{Quality: 64, Resolution: "720p"},
		{Quality: 80, Resolution: "1080p"},
	}

	got := d.selectStream(streams)
	if got == nil {
		t.Fatal("selectStream returned nil")
	}
	if got.Quality != 64 {
		t.Errorf("selectStream quality = %d, want 64 (720p)", got.Quality)
	}

	// Test fallback to best when exact quality not found.
	d.config.Quality = "480p"
	streams = []*parser.StreamInfo{
		{Quality: 80, Resolution: "1080p"},
		{Quality: 64, Resolution: "720p"},
	}
	got = d.selectStream(streams)
	if got == nil || got.Quality != 80 {
		t.Errorf("selectStream fallback quality = %d, want 80", got.Quality)
	}

	// Test empty streams.
	d.config.Quality = "1080p"
	got = d.selectStream(nil)
	if got != nil {
		t.Error("selectStream should return nil for empty streams")
	}

	// Test "best" quality.
	d.config.Quality = "best"
	streams = []*parser.StreamInfo{
		{Quality: 16},
		{Quality: 80},
	}
	got = d.selectStream(streams)
	if got == nil || got.Quality != 80 {
		t.Errorf("selectStream best quality = %d, want 80", func() int {
			if got != nil {
				return got.Quality
			}
			return -1
		}())
	}
}

func TestProgressReader_Read(t *testing.T) {
	data := make([]byte, 3*1024*1024) // 3 MB
	for i := range data {
		data[i] = byte(i % 256)
	}
	reader := bytes.NewReader(data)

	progressChan := make(chan DownloadProgress, 10)
	pr := &ProgressReader{
		Reader:   reader,
		Total:    int64(len(data)),
		Progress: progressChan,
	}

	buf := make([]byte, 64*1024) // 64 KB buffer
	var totalRead int64
	for {
		n, err := pr.Read(buf)
		totalRead += int64(n)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected read error: %v", err)
		}
	}

	if totalRead != int64(len(data)) {
		t.Errorf("total read = %d, want %d", totalRead, len(data))
	}

	// Should have received progress updates.
	select {
	case progress := <-progressChan:
		if progress.TotalSize != int64(len(data)) {
			t.Errorf("TotalSize = %d, want %d", progress.TotalSize, len(data))
		}
		if progress.Downloaded == 0 {
			t.Error("Downloaded should be > 0")
		}
	default:
		// Might be empty if all updates were consumed, that's ok.
	}
}

func TestProgressReader_NoProgressChannel(t *testing.T) {
	data := []byte("hello world")
	reader := bytes.NewReader(data)

	pr := &ProgressReader{
		Reader: reader,
		Total:  int64(len(data)),
	}

	buf := make([]byte, len(data))
	n, err := pr.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(data) {
		t.Errorf("read %d bytes, want %d", n, len(data))
	}
}

func TestProgressReader_UnknownSize(t *testing.T) {
	data := []byte("some data")
	reader := bytes.NewReader(data)
	progressChan := make(chan DownloadProgress, 10)

	pr := &ProgressReader{
		Reader:   reader,
		Total:    0, // unknown size
		Progress: progressChan,
	}

	buf := make([]byte, len(data))
	_, err := pr.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("unexpected error: %v", err)
	}

	// With unknown size, percentage should remain 0.
	select {
	case p := <-progressChan:
		if p.Percentage != 0 {
			t.Errorf("percentage should be 0 for unknown size, got %f", p.Percentage)
		}
	default:
	}
}

func TestNewDownloader(t *testing.T) {
	config := Config{
		OutputDir: "/tmp/test",
		Threads:   4,
		Verbose:   true,
		Quality:   "1080p",
		Format:    "mp4",
	}
	d := NewDownloader(config)
	if d == nil {
		t.Fatal("NewDownloader returned nil")
	}
	if d.config.OutputDir != "/tmp/test" {
		t.Errorf("OutputDir = %q", d.config.OutputDir)
	}
	if d.config.Threads != 4 {
		t.Errorf("Threads = %d, want 4", d.config.Threads)
	}
	if d.client == nil {
		t.Error("HTTP client is nil")
	}
	if d.logger == nil {
		t.Error("logger is nil")
	}
	// Transport should be configured with timeouts (not DefaultTransport).
	if d.client.Transport == nil {
		t.Error("HTTP client transport is nil")
	}
}

func TestDownloadProgress(t *testing.T) {
	dp := DownloadProgress{
		TotalSize:  1024 * 1024,
		Downloaded: 512 * 1024,
		Percentage: 50.0,
		Speed:      1024 * 1024,
		ETA:        30 * time.Second,
	}

	if dp.TotalSize != 1024*1024 {
		t.Errorf("TotalSize = %d", dp.TotalSize)
	}
	if dp.Percentage != 50.0 {
		t.Errorf("Percentage = %f", dp.Percentage)
	}
}
