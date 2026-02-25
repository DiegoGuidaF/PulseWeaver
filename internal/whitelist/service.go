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
	provider  EnabledIPsProvider
	filePath  string
	rateLimit time.Duration
}

// NewService creates a new whitelist service.
// Receives the whole ConfWhitelist struct since it is domain-specific.
func NewService(provider EnabledIPsProvider, conf config.ConfWhitelist) *Service {
	return &Service{
		provider:  provider,
		filePath:  conf.FilePath,
		rateLimit: conf.RateLimit,
	}
}

// RunListener is the main event loop goroutine.
// First signal (or signal after cooldown) runs Regenerate immediately.
// Signals within rateLimit of last execution are deferred to a single run at lastExecution+rateLimit.
// Runs until context is cancelled.
func (s *Service) RunListener(ctx context.Context, deviceEvents <-chan struct{}) error {
	var timer *time.Timer
	var timerC <-chan time.Time
	var lastExecution time.Time
	ctx, _ = logging.Enrich(ctx, slog.String(AttrKeyComponent, "whitelist"))
	logger := logging.FromCtx(ctx)

	for {
		select {
		case <-deviceEvents:
			if lastExecution.IsZero() || time.Since(lastExecution) >= s.rateLimit {
				// Outside cooldown: run immediately
				if timer != nil {
					timer.Stop()
					timer = nil
					timerC = nil
				}
				lastExecution = time.Now()

				if err := s.Regenerate(ctx); err != nil {
					logger.Error("whitelist regeneration failed", slog.Any(AttrKeyError, err))
				}
			} else {
				// Inside cooldown: ensure single timer at lastExecution+rateLimit
				if timer == nil {
					fireAt := lastExecution.Add(s.rateLimit)
					d := time.Until(fireAt)
					if d <= 0 {
						d = time.Millisecond
					}
					timer = time.NewTimer(d)
					timerC = timer.C
				}
			}
		case <-timerC:
			timer = nil
			timerC = nil
			if err := s.Regenerate(ctx); err != nil {
				logger.Error("whitelist regeneration failed", slog.Any(AttrKeyError, err))
			}
			lastExecution = time.Now()
		case <-ctx.Done():
			if timer != nil {
				timer.Stop()
			}
			return nil
		}
	}
}

// generateContent generates the file content as bytes from a list of IP addresses.
// Each IP is written on its own line with a trailing newline, matching the format written to disk.
// Returns empty slice (not nil) when ips is empty.
func generateContent(ips []string) []byte {
	if len(ips) == 0 {
		return []byte{}
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
func (s *Service) Regenerate(ctx context.Context) error {
	logger := logging.FromCtx(ctx)

	ips, err := s.provider.GetEnabledUniqueIPs(ctx)
	if err != nil {
		logger.Error("failed to query enabled IPs", slog.Any(AttrKeyError, err))
		return fmt.Errorf("query enabled IPs: %w", err)
	}

	newContent := generateContent(ips)
	if len(newContent) == 0 {
		logger.Warn("no enabled IPs found, writing empty whitelist")
	}

	newHash := hashFileContent(newContent)
	existingHash, err := s.hashExistingFile()
	if err != nil {
		logger.Warn("failed to read existing file for comparison, proceeding with write",
			slog.String(AttrKeyWhitelistFile, s.filePath),
			slog.Any(AttrKeyError, err),
		)
	} else if newHash == existingHash {
		logger.Info("whitelist unchanged, skipping write",
			slog.String(AttrKeyWhitelistFile, s.filePath),
			slog.Int(AttrKeyIPCount, len(ips)),
		)
		return nil
	}

	if err := s.atomicWrite(ctx, newContent); err != nil {
		return err
	}

	logger.Info("whitelist regenerated",
		slog.String(AttrKeyWhitelistFile, s.filePath),
		slog.Int(AttrKeyIPCount, len(ips)),
	)
	return nil
}

// atomicWrite writes content to the whitelist file using a temp file, fsync, and atomic rename.
func (s *Service) atomicWrite(ctx context.Context, content []byte) error {
	logger := logging.FromCtx(ctx)
	tempPath := s.filePath + ".tmp"

	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		logger.Error("failed to create directory", slog.String(AttrKeyWhitelistFile, dir), slog.Any(AttrKeyError, err))
		return fmt.Errorf("create directory: %w", err)
	}

	file, err := os.Create(tempPath)
	if err != nil {
		logger.Error("failed to create temp file", slog.String(AttrKeyWhitelistFile, tempPath), slog.Any(AttrKeyError, err))
		return fmt.Errorf("create temp file: %w", err)
	}

	if _, err := file.Write(content); err != nil {
		_ = file.Close()
		_ = os.Remove(tempPath)
		logger.Error("failed to write content to temp file", slog.String(AttrKeyWhitelistFile, tempPath), slog.Any(AttrKeyError, err))
		return fmt.Errorf("write content: %w", err)
	}

	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(tempPath)
		logger.Error("failed to sync temp file", slog.String(AttrKeyWhitelistFile, tempPath), slog.Any(AttrKeyError, err))
		return fmt.Errorf("sync temp file: %w", err)
	}

	if err := file.Close(); err != nil {
		logger.Error("failed to close temp file", slog.String(AttrKeyWhitelistFile, tempPath), slog.Any(AttrKeyError, err))
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tempPath, s.filePath); err != nil {
		logger.Error("failed to rename temp file", slog.String(AttrKeyWhitelistFile, s.filePath), slog.Any(AttrKeyError, err))
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}
