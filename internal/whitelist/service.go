package whitelist

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/config"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/logging"
)

// EnabledIPsProvider is an interface for providers that can return enabled IP addresses.
type EnabledIPsProvider interface {
	GetEnabledUniqueIPs(ctx context.Context) ([]string, error)
}

type Service struct {
	provider      EnabledIPsProvider
	filePath      string
	debounceDelay time.Duration
	eventChan     chan struct{} // buffered, size 1
}

// NewService creates a new whitelist service.
// Receives the whole ConfWhitelist struct since it is domain-specific.
func NewService(provider EnabledIPsProvider, conf config.ConfWhitelist) *Service {
	return &Service{
		provider:      provider,
		filePath:      conf.FilePath,
		debounceDelay: conf.DebounceDelay,
		eventChan:     make(chan struct{}, 1), // buffer size 1 for debounce
	}
}

// Updates returns the write-only version of the event channel.
// Called during wiring to give the device service a channel to send on.
func (s *Service) Updates() chan<- struct{} {
	return s.eventChan
}

// Run is the main event loop goroutine.
// Uses channel-based timer with select for debouncing.
// Runs until context is cancelled.
func (s *Service) Run(ctx context.Context) error {
	var timer *time.Timer
	var timerC <-chan time.Time
	logging.Enrich(ctx, slog.String(AttrKeyComponent, "whitelist"))

	for {
		select {
		case <-s.eventChan:
			// Stop existing timer if any
			if timer != nil {
				timer.Stop()
			}
			// Reset timer for debounce delay
			timer = time.NewTimer(s.debounceDelay)
			timerC = timer.C
		case <-timerC:
			// Timer fired, regenerate whitelist
			timerC = nil
			logger := logging.FromCtx(ctx)
			if err := s.Regenerate(ctx); err != nil {
				// Error is logged inside Regenerate, continue listening
				logger.Error("whitelist regeneration failed", slog.Any(AttrKeyError, err))
			}
		case <-ctx.Done():
			// Clean shutdown: stop timer and exit
			if timer != nil {
				timer.Stop()
			}
			return nil
		}
	}
}

// generateContent generates the file content as bytes from a list of IP addresses.
// Each IP is written on its own line with a trailing newline, matching the format written to disk.
func generateContent(ips []string) []byte {
	if len(ips) == 0 {
		return nil
	}
	// Preallocate capacity: estimate average IP length + newline per IP
	// Using a conservative estimate of 15 chars per IP (IPv4) + 1 for newline
	estimatedCapacity := (15 + 1) * len(ips)
	content := make([]byte, 0, estimatedCapacity)
	for _, ip := range ips {
		content = append(content, []byte(ip)...)
		content = append(content, '\n')
	}
	return content
}

// hashFileContent computes the SHA256 hash of file content.
func hashFileContent(content []byte) [32]byte {
	return sha256.Sum256(content)
}

// hashExistingFile reads the existing whitelist file and returns its SHA256 hash.
// If the file does not exist, returns a zero hash.
// If there's an error reading the file, logs the error and returns the error.
func (s *Service) hashExistingFile() ([32]byte, error) {
	var zeroHash [32]byte

	content, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist - return zero hash (will never match real content hash)
			return zeroHash, nil
		}
		// Other read errors - return error so caller can decide how to handle
		return zeroHash, fmt.Errorf("read existing file: %w", err)
	}

	return hashFileContent(content), nil
}

// Regenerate queries enabled IPs from the provider and writes them to the whitelist file.
// Uses hash-based comparison to skip writes when content hasn't changed.
// Uses atomic file write pattern: temp file -> fsync -> rename.
// Each IP is written on its own line with a trailing newline.
func (s *Service) Regenerate(ctx context.Context) error {
	logger := logging.FromCtx(ctx)

	// Query enabled IPs from provider
	ips, err := s.provider.GetEnabledUniqueIPs(ctx)
	if err != nil {
		logger.Error("failed to query enabled IPs", slog.Any(AttrKeyError, err))
		return fmt.Errorf("query enabled IPs: %w", err)
	}

	// Generate new content and hash
	newContent := generateContent(ips)
	newHash := hashFileContent(newContent)

	// Get hash of existing file
	existingHash, err := s.hashExistingFile()
	if err != nil {
		// Log error but proceed with write (safer to regenerate than skip)
		logger.Warn("failed to read existing file for comparison, proceeding with write",
			slog.String(AttrKeyWhitelistFile, s.filePath),
			slog.Any(AttrKeyError, err),
		)
	} else if newHash == existingHash {
		// Content unchanged - skip write
		logger.Info("whitelist unchanged, skipping write",
			slog.String(AttrKeyWhitelistFile, s.filePath),
			slog.Int(AttrKeyIPCount, len(ips)),
		)
		return nil
	}

	// Prepare temp file path
	tempPath := s.filePath + ".tmp"

	// Create directory if it doesn't exist
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		logger.Error("failed to create directory", slog.String(AttrKeyWhitelistFile, dir), slog.Any(AttrKeyError, err))
		return fmt.Errorf("create directory: %w", err)
	}

	// Open temp file for writing
	file, err := os.Create(tempPath)
	if err != nil {
		logger.Error("failed to create temp file", slog.String(AttrKeyWhitelistFile, tempPath), slog.Any(AttrKeyError, err))
		return fmt.Errorf("create temp file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Error("failed to close temp file", slog.String(AttrKeyWhitelistFile, tempPath), slog.Any(AttrKeyError, err))
		}
	}()

	// Write content to temp file
	if _, err := file.Write(newContent); err != nil {
		logger.Error("failed to write content to temp file", slog.String(AttrKeyWhitelistFile, tempPath), slog.Any(AttrKeyError, err))
		return fmt.Errorf("write content: %w", err)
	}

	// Sync to disk
	if err := file.Sync(); err != nil {
		logger.Error("failed to sync temp file", slog.String(AttrKeyWhitelistFile, tempPath), slog.Any(AttrKeyError, err))
		return fmt.Errorf("sync temp file: %w", err)
	}

	// Close file before rename
	if err := file.Close(); err != nil {
		logger.Error("failed to close temp file", slog.String(AttrKeyWhitelistFile, tempPath), slog.Any(AttrKeyError, err))
		return fmt.Errorf("close temp file: %w", err)
	}

	// Atomic rename: temp file -> final file
	if err := os.Rename(tempPath, s.filePath); err != nil {
		logger.Error("failed to rename temp file", slog.String(AttrKeyWhitelistFile, s.filePath), slog.Any(AttrKeyError, err))
		return fmt.Errorf("rename temp file: %w", err)
	}

	// Log success with IP count
	logger.Info("whitelist regenerated",
		slog.String(AttrKeyWhitelistFile, s.filePath),
		slog.Int(AttrKeyIPCount, len(ips)),
	)

	return nil
}
