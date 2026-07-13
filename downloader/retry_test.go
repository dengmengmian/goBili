package downloader

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"
)

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		statusCode int
		want       bool
	}{
		// Server errors → retryable.
		{"500", nil, 500, true},
		{"502", nil, 502, true},
		{"503", nil, 503, true},
		{"429 rate limit", nil, 429, true},

		// Client errors → not retryable.
		{"400", nil, 400, false},
		{"401", nil, 401, false},
		{"403", nil, 403, false},
		{"404", nil, 404, false},

		// Success → not retryable.
		{"200", nil, 200, false},

		// Transient network errors → retryable.
		{"timeout", &net.DNSError{IsTimeout: true}, 0, true},
		{"connection reset", errors.New("connection reset by peer"), 0, true},
		{"connection refused", errors.New("connection refused"), 0, true},
		{"EOF", errors.New("unexpected EOF"), 0, true},
		{"i/o timeout", errors.New("i/o timeout"), 0, true},
		{"broken pipe", errors.New("broken pipe"), 0, true},

		// Non-transient errors → not retryable.
		{"random error", errors.New("something bad"), 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRetryable(tt.err, tt.statusCode); got != tt.want {
				t.Errorf("isRetryable(%v, %d) = %v, want %v", tt.err, tt.statusCode, got, tt.want)
			}
		})
	}
}

func TestRetry_Success(t *testing.T) {
	ctx := context.Background()
	cfg := retryConfig{MaxRetries: 2, BaseDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond}

	attempts := 0
	err := retry(ctx, cfg, func() (int, error) {
		attempts++
		return 200, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
}

func TestRetry_EventualSuccess(t *testing.T) {
	ctx := context.Background()
	cfg := retryConfig{MaxRetries: 3, BaseDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond}

	attempts := 0
	err := retry(ctx, cfg, func() (int, error) {
		attempts++
		if attempts < 3 {
			return 503, fmt.Errorf("temporary server error")
		}
		return 200, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestRetry_Exhausted(t *testing.T) {
	ctx := context.Background()
	cfg := retryConfig{MaxRetries: 2, BaseDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond}

	attempts := 0
	err := retry(ctx, cfg, func() (int, error) {
		attempts++
		return 500, fmt.Errorf("persistent server error")
	})
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if attempts != 3 { // initial attempt + 2 retries
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestRetry_NonRetryable(t *testing.T) {
	ctx := context.Background()
	cfg := retryConfig{MaxRetries: 3, BaseDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond}

	attempts := 0
	err := retry(ctx, cfg, func() (int, error) {
		attempts++
		return 404, fmt.Errorf("not found")
	})
	if err == nil {
		t.Fatal("expected error for non-retryable status")
	}
	if attempts != 1 { // should not retry
		t.Errorf("attempts = %d, want 1", attempts)
	}
}

func TestRetry_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := retryConfig{MaxRetries: 5, BaseDelay: 100 * time.Millisecond, MaxDelay: time.Second}

	// Cancel after first attempt.
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := retry(ctx, cfg, func() (int, error) {
		return 503, fmt.Errorf("error")
	})
	if err == nil {
		t.Fatal("expected context error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestFormatSpeed(t *testing.T) {
	tests := []struct {
		bytesPerSec int64
		want        string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1024 * 1024, "1.00 MB"},
		{5 * 1024 * 1024, "5.00 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatSpeed(tt.bytesPerSec)
			if got != tt.want {
				t.Errorf("formatSpeed(%d) = %q, want %q", tt.bytesPerSec, got, tt.want)
			}
		})
	}
}
