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
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/logging"
)

const whitelistDebounceDelay = 50 * time.Millisecond

// EnabledIPsProvider is an interface for providers that can return enabled IP addresses.
type EnabledIPsProvider interface {
	GetEnabledUniqueIPs(ctx context.Context) ([]string, error)
}

// ChangeNotifier used to signal that the whitelist has successfully changed
type ChangeNotifier interface {
	NotifyChange(ctx context.Context)
}

type Service struct {
	provider            EnabledIPsProvider
	filePath            string
	rateLimit           time.Duration
	changeNotifier      ChangeNotifier
	addressChangeSignal chan struct{}
	logger              *slog.Logger
}

// NewService creates a new whitelist service.
// Receives the whole ConfWhitelist struct since it is domain-specific.
func NewService(provider EnabledIPsProvider, conf config.ConfWhitelist, notifier ChangeNotifier, logger *slog.Logger) *Service {
	return &Service{
		provider:            provider,
		filePath:            conf.FilePath,
		rateLimit:           conf.RateLimit,
		changeNotifier:      notifier,
		addressChangeSignal: make(chan struct{}, 1),
		logger:              logger.With(slog.String(logging.AttrKeyComponent, "whitelist")),
	}
}

// OnAddressEvent implements device.AddressObserver.
// The context is discarded: non-blocking drop, context not needed for a buffered signal.
func (s *Service) OnAddressEvent(_ context.Context, _ device.AddressEvent) {
	select {
	case s.addressChangeSignal <- struct{}{}:
	default:
	}
}

// RunListener is the main event loop goroutine.
// Address change signals trigger debounced and rate-limited Regenerate calls.
// Runs until context is cancelled.
func (s *Service) RunListener(ctx context.Context) error {
	var timer *time.Timer
	var timerC <-chan time.Time
	var lastRunAt time.Time

	// debounce calculates the required delay and configures the timer,
	debounce := func() {
		runAt := time.Now().Add(whitelistDebounceDelay)
		if !lastRunAt.IsZero() {
			earliestByRateLimit := lastRunAt.Add(s.rateLimit)
			if runAt.Before(earliestByRateLimit) {
				runAt = earliestByRateLimit
			}
		}

		delay := time.Until(runAt)
		if delay < 0 {
			delay = 0
		}

		if timer == nil {
			timer = time.NewTimer(delay)
			timerC = timer.C
		} else {
			// Safe to Reset a running timer here: this runs in a single goroutine
			// and the select cases are mutually exclusive, so timerC cannot fire
			// concurrently with this branch.
			timer.Reset(delay)
		}
	}

	for {
		select {
		case <-s.addressChangeSignal:
			debounce()
		case <-timerC:
			timer = nil
			timerC = nil

			// Each regeneration cycle gets a unique flow ID so all log lines
			// from that regeneration can be correlated.
			regenCtx := logging.WithRequestID(ctx, "regen-"+logging.NewShortID())
			if err := s.Regenerate(regenCtx); err != nil {
				s.logger.ErrorContext(regenCtx, "whitelist regeneration failed", slog.Any(AttrKeyError, err))
			}
			lastRunAt = time.Now()
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
	ips, err := s.provider.GetEnabledUniqueIPs(ctx)
	if err != nil {
		return fmt.Errorf("query enabled IPs: %w", err)
	}

	newContent := generateContent(ips)
	if len(newContent) == 0 {
		s.logger.WarnContext(ctx, "no enabled IPs found, writing empty whitelist")
	}

	newHash := hashFileContent(newContent)
	existingHash, err := s.hashExistingFile()
	if err != nil {
		s.logger.WarnContext(ctx, "failed to read existing file for comparison, proceeding with write",
			slog.String(AttrKeyWhitelistFile, s.filePath),
			slog.Any(AttrKeyError, err),
		)
	} else if newHash == existingHash {
		s.logger.DebugContext(ctx, "whitelist unchanged, skipping write",
			slog.String(AttrKeyWhitelistFile, s.filePath),
			slog.Int(AttrKeyIPCount, len(ips)),
		)
		return nil
	}

	if err := s.atomicWrite(ctx, newContent); err != nil {
		return err
	}

	s.changeNotifier.NotifyChange(ctx)

	s.logger.InfoContext(ctx, "whitelist regenerated",
		slog.String(AttrKeyWhitelistFile, s.filePath),
		slog.Int(AttrKeyIPCount, len(ips)),
	)
	return nil
}

// atomicWrite writes content to the whitelist file using a temp file, fsync, and atomic rename.
func (s *Service) atomicWrite(_ context.Context, content []byte) error {
	tempPath := s.filePath + ".tmp"

	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	file, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("create temp file %s: %w", tempPath, err)
	}

	if _, err := file.Write(content); err != nil {
		_ = file.Close()
		_ = os.Remove(tempPath)
		return fmt.Errorf("write content: %w", err)
	}

	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(tempPath)
		return fmt.Errorf("sync temp file: %w", err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tempPath, s.filePath); err != nil {
		return fmt.Errorf("rename %s to %s: %w", tempPath, s.filePath, err)
	}

	return nil
}
