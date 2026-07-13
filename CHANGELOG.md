# Changelog

All notable changes to goBili are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Concurrent chunked downloads**: the `--threads` (`-t`) flag now performs
  multi-threaded downloads via HTTP Range requests. The downloader probes the
  server with a HEAD request; if the server advertises `Accept-Ranges: bytes`,
  the file is split into equal chunks and fetched in parallel goroutines, then
  reassembled with `WriteAt` calls.
- **HTTP Range resume support**: chunked download pre-allocates the file via
  `Truncate(contentLength)`, enabling partial-file detection and selective
  chunk re-fetch in a future iteration.
- **Exponential-backoff retry**: all HTTP downloads transparently retry up to
  3 times with exponential backoff (1s, 2s, 4s) and +-25% jitter. Retried:
  5xx, 429, and transient network errors (connection reset, timeout, EOF,
  broken pipe, TLS handshake timeout). Non-retryable 4xx errors fail immediately.
- **Proper HTTP transport timeouts**: `DialContext` 30s, `TLSHandshake` 15s,
  `ResponseHeader` 30s, `KeepAlive` 30s, `IdleConn` 90s. Replaces the previous
  `Timeout: 0` (infinite), which could hang forever on a stalled connection.
- **Context-based cancellation**: the download pipeline threads `context.Context`
  through every layer. Cancelling the context (e.g. Ctrl+C) stops in-flight
  downloads and cleans up partial temporary files. Concurrent video+audio
  downloads share a derived context: if either fails, the sibling is cancelled.
- **Shell completion**: `goBili completion [bash|zsh|fish|powershell]` generates
  autocompletion scripts via cobra.
- **GoReleaser configuration** (`.goreleaser.yml`): cross-platform release
  automation for linux/darwin/windows x amd64/arm64, with tar.gz/zip archives,
  SHA256 checksums, and changelog generation.
- **golangci-lint configuration** (`.golangci.yml`): 14 linters enabled
  (errcheck, staticcheck, gosec, revive, errorlint, misspell, whitespace,
  nilerr, nilnil, unconvert, bodyclose, gosimple, govet, ineffassign).
- **golangci-lint CI job**: runs before build and test in GitHub Actions.
- **Dependabot configuration** (`.github/dependabot.yml`): automated weekly
  updates for Go modules and GitHub Actions.
- **Structured error types** (`downloader/errors.go`): `DownloadError` wraps
  underlying errors with a user-facing `Action` suggestion (e.g. "please login
  first using 'goBili login'").
- **`debug.BuildInfo` fallback** in `goBili version`: when built via
  `go install` (no ldflags), the command prints `vcs.revision` and `vcs.time`
  from the Go runtime build metadata.
- **Test suite**: unit tests for auth (cookie I/O, authentication checks,
  user-info API), parser (URL routing, JSON unmarshaling, stream selection),
  and downloader (filename sanitization, stream selection, progress reader,
  retry logic, speed formatting) with race detection.
- **CI matrix**: GitHub Actions tests against Go 1.21, 1.22, and 1.23.
- **SECURITY.md**: vulnerability reporting policy.

### Changed
- **Module path renamed** from `goBili` to `github.com/dengmengmian/goBili`
  (**BREAKING**). Import paths and Makefile ldflags updated accordingly.
  Required for `go install` and Go module proxy compatibility.
- **CLI language unified to English**: `goBili legal`, `goBili login`
  (browser flow), and QR-code auth messages are now in English. Chinese
  documentation remains in `README.md`.
- **`goBili version` output**: changed from Chinese bullet format to standard
  `binary version` + `build time` + `git commit` one-liner.
- **Progress bar**: replaced per-1MB `\\r` line with a 500ms-windowed display
  showing instantaneous speed (B/KB/MB/GB) and ETA based on windowed throughput
  rather than average speed over the entire download lifetime.
- **Release ldflags**: added `-s -w` (strip debug info, omit symbol table)
  for smaller release binaries.

### Fixed
- **Operator precedence in progress display**: the original code wrote
  `pr.ReadBytes%1024*1024`, which Go parses as `(pr.ReadBytes % 1024) * 1024`,
  causing the progress line to print every 1 KB instead of every 1 MB.
  Fixed to `pr.ReadBytes % (1024 * 1024)`.
- **HTTP client had no timeout**: `NewDownloader` set `Timeout: 0`, meaning
  an HTTP request that hung (DNS, connect, or body read) would block
  indefinitely. Replaced with per-operation transport-level deadlines.
- **Filename path traversal**: the original `generateFilename` used eight
  bare `strings.ReplaceAll` calls without guarding against `..`, null bytes,
  absolute paths, or control characters. Replaced with `sanitizeFilename`
  that uses `filepath.Base`, explicit replacer for dangerous characters,
  and truncation with rune awareness.
- **Error returns silently discarded**: `cmd.Flags().GetString("quality")`
  and three `viper.BindPFlag` calls discarded their error return values;
  all are now checked or explicitly handled.
- **`fmt.Errorf` used `%v` for wrapped errors**: ~40 call sites across
  `auth`, `downloader`, `parser`, and `cmd` packages used `%v` instead of
  `%w`, breaking `errors.Is`/`errors.As` unwrapping.
- **`fmt.Scanln` return values ignored**: `login` and `logout` commands
  discarded Scanln errors; they are now checked.
- **Misspelling**: "cancelled" changed to "canceled" in `logout` output.
- **Cobra completion generators' return values ignored**: `GenBashCompletion`
  et al. were called without checking errors; switched to `RunE` with proper
  error propagation.
- **Unused parameters**: `runVersion`, `runLegal`, `runLogin`, `runLogout`,
  `parsePageRange`, `loginWithBrowser`, `GetVideoStreams` had unused
  parameters; renamed to `_` or removed where possible.
- **`logrus.Errorf` used `%w`**: two log statements accidentally used `%w`
  (which is for `fmt.Errorf` only, not for logging).
- **YAML config files blocked by `.gitignore`**: the `*.yml` glob prevented
  `.golangci.yml`, `.goreleaser.yml`, and `.github/dependabot.yml` from
  being tracked. Added explicit `!` exceptions.

### Security
- **Path traversal prevented**: `sanitizeFilename` now calls `filepath.Base`,
  rejects `.` and `..`, strips control characters, and enforces a length cap,
  preventing writes outside the output directory.
- **gosec enabled**: code is scanned for common Go security issues in CI.
