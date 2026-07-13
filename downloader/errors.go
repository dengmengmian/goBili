package downloader

import (
	"errors"
	"fmt"
)

// Sentinel errors for common failure modes.
var (
	ErrAuthRequired   = errors.New("authentication required: please login first using 'goBili login'")
	ErrNetworkTimeout = errors.New("network timeout: check your internet connection and try again")
	ErrServerError    = errors.New("server error: the remote server is temporarily unavailable, try again later")
	ErrDiskFull       = errors.New("disk full or write permission denied: check available space and permissions")
	ErrInvalidURL     = errors.New("invalid URL: the provided Bilibili URL could not be parsed")
	ErrFileExists     = errors.New("output file already exists: use --force to overwrite")
)

// DownloadError wraps an error with a user-friendly message and a suggested action.
type DownloadError struct {
	Op     string // The operation that failed (e.g., "download", "merge", "parse").
	URL    string // The URL being downloaded, if applicable.
	Err    error  // The underlying error.
	Action string // A suggested action for the user.
}

func (e *DownloadError) Error() string {
	msg := fmt.Sprintf("%s failed", e.Op)
	if e.URL != "" {
		msg += fmt.Sprintf(" for %s", e.URL)
	}
	if e.Err != nil {
		msg += fmt.Sprintf(": %v", e.Err)
	}
	if e.Action != "" {
		msg += fmt.Sprintf("\n  → %s", e.Action)
	}
	return msg
}

func (e *DownloadError) Unwrap() error {
	return e.Err
}

// NewDownloadError creates a DownloadError with the given parameters.
func NewDownloadError(op, url string, err error, action string) *DownloadError {
	return &DownloadError{
		Op:     op,
		URL:    url,
		Err:    err,
		Action: action,
	}
}

// UserError returns an error that is clearly attributable to user input.
// These errors should not show stack traces or debug info.
func UserError(msg string) error {
	return &DownloadError{
		Op:     "user input",
		Err:    errors.New(msg),
		Action: "Check the command syntax with 'goBili help'",
	}
}
