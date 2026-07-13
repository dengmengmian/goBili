// Package downloader provides retry logic for resilient HTTP downloads.
package downloader

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"net"
	"strings"
	"time"
)

// retryConfig holds parameters for the retry loop.
type retryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
}

// defaultRetryConfig returns the default retry configuration.
func defaultRetryConfig() retryConfig {
	return retryConfig{
		MaxRetries: 3,
		BaseDelay:  1 * time.Second,
		MaxDelay:   30 * time.Second,
	}
}

// isRetryable checks whether an error or HTTP response status warrants a retry.
func isRetryable(err error, statusCode int) bool {
	if err == nil && statusCode > 0 && statusCode < 400 {
		return false
	}

	// Retry on server errors and rate limiting.
	if statusCode >= 500 || statusCode == 429 {
		return true
	}

	// Retry on transient network errors.
	if err != nil {
		// Check for timeout and temporary errors.
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return true
		}
		// Check for connection reset and other temporary conditions.
		msg := err.Error()
		for _, substr := range []string{
			"connection reset by peer",
			"connection refused",
			"broken pipe",
			"EOF",
			"no such host",
			"TLS handshake timeout",
			"i/o timeout",
			"use of closed network connection",
		} {
			if strings.Contains(strings.ToLower(msg), strings.ToLower(substr)) {
				return true
			}
		}
	}

	return false
}

// retry executes fn with exponential backoff. fn returns (statusCode, error).
// It respects ctx cancellation.
func retry(ctx context.Context, cfg retryConfig, fn func() (int, error)) error {
	var lastErr error
	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		// Check context before each attempt.
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return fmt.Errorf("%w (after %d retries): %v", ctx.Err(), attempt, lastErr)
			}
			return ctx.Err()
		default:
		}

		statusCode, err := fn()
		if err == nil && statusCode > 0 && statusCode < 400 {
			return nil
		}

		if !isRetryable(err, statusCode) {
			if err != nil {
				return err
			}
			return fmt.Errorf("HTTP %d", statusCode)
		}

		lastErr = err
		if lastErr == nil {
			lastErr = fmt.Errorf("HTTP %d", statusCode)
		}

		if attempt == cfg.MaxRetries {
			break
		}

		// Exponential backoff with jitter.
		delay := time.Duration(math.Min(
			float64(cfg.BaseDelay)*math.Pow(2, float64(attempt)),
			float64(cfg.MaxDelay),
		))
		// Add jitter: ±25%.
		jitter := time.Duration(rand.Int63n(int64(delay)/2 + 1))
		delay = delay - delay/4 + jitter

		select {
		case <-ctx.Done():
			return fmt.Errorf("%w (after %d retries): %v", ctx.Err(), attempt, lastErr)
		case <-time.After(delay):
		}
	}

	return fmt.Errorf("failed after %d retries: %w", cfg.MaxRetries, lastErr)
}
