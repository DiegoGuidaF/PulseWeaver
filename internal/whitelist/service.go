package whitelist

import (
	"context"
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

// Regenerate queries enabled IPs from the provider and writes them to the whitelist file.
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
	defer file.Close()

	// Write IPs, one per line with trailing newline
	for _, ip := range ips {
		if _, err := fmt.Fprintf(file, "%s\n", ip); err != nil {
			logger.Error("failed to write IP to temp file", slog.String(AttrKeyWhitelistFile, tempPath), slog.Any(AttrKeyError, err))
			return fmt.Errorf("write IP: %w", err)
		}
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
