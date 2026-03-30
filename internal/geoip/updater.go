package geoip

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

const (
	maxAge                  = 30 * 24 * time.Hour
	updaterScheduleInterval = 24 * time.Hour

	dbipCountryURL = "https://download.db-ip.com/free/dbip-country-lite-%s.mmdb.gz"
	dbipAsnURL     = "https://download.db-ip.com/free/dbip-asn-lite-%s.mmdb.gz"
)

// SyncDBIPFiles checks whether the MMDB files exist and are fresh (<30 days old).
// If missing or stale, it downloads the current month's files from DB-IP.
// Returns resolved country and ASN paths, or an error if download fails and no file exists.
func SyncDBIPFiles(ctx context.Context, dataDir string, logger *slog.Logger) (countryPath, asnPath string, err error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return "", "", fmt.Errorf("create geoip cache dir: %w", err)
	}

	month := time.Now().Format("2006-01")

	countryDst := filepath.Join(dataDir, "dbip-country.mmdb")
	asnDst := filepath.Join(dataDir, "dbip-asn.mmdb")

	countryURL := fmt.Sprintf(dbipCountryURL, month)
	asnURL := fmt.Sprintf(dbipAsnURL, month)

	countryPath, err = syncFile(ctx, countryDst, countryURL, logger)
	if err != nil {
		return "", "", err
	}

	asnPath, err = syncFile(ctx, asnDst, asnURL, logger)
	if err != nil {
		return "", "", err
	}

	return countryPath, asnPath, nil
}

// RunUpdater runs a background loop that re-checks MMDB freshness every 24 hours.
// It is a no-op when the lookup was opened with user-provided paths (not managed by DB-IP).
func (l *Lookup) RunUpdater(ctx context.Context, logger *slog.Logger) error {
	defer func(l *Lookup) {
		_ = l.Close()
	}(l)
	if l.dataDir == "" {
		<-ctx.Done()
		return ctx.Err()
	}
	ticker := time.NewTicker(updaterScheduleInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			cp, ap, err := SyncDBIPFiles(ctx, l.dataDir, logger)
			if err != nil {
				logger.Error("geoip: auto-update failed", slog.Any(logging.AttrKeyError, err))
				continue
			}
			if err := l.Reload(cp, ap); err != nil {
				logger.Error("geoip: reload after auto-update failed", slog.Any(logging.AttrKeyError, err))
			} else {
				logger.Info("geoip: MMDB files refreshed")
			}
		}
	}
}

// syncFile returns cachedPath if it exists and is fresh; otherwise downloads from url.
// Fail-open: if download fails but cached file exists, returns the stale path with a warning.
func syncFile(ctx context.Context, cachedPath, url string, logger *slog.Logger) (string, error) {
	if isFileFresh(cachedPath) {
		return cachedPath, nil
	}
	stale := fileExists(cachedPath)
	logger.Info("geoip: downloading DB-IP file", slog.String("url", url))
	if err := downloadGZ(ctx, url, cachedPath); err != nil {
		if stale {
			logger.Warn("geoip: download failed, using stale cache", slog.Any(logging.AttrKeyError, err), slog.String("path", cachedPath))
			return cachedPath, nil
		}
		return "", fmt.Errorf("download %s: %w", url, err)
	}
	return cachedPath, nil
}

func isFileFresh(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return time.Since(info.ModTime()) < maxAge
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// httpClient is used for MMDB downloads. Timeout caps the entire request.
var httpClient = &http.Client{Timeout: 2 * time.Minute}

// downloadGZ downloads a .gz file from url, decompresses it, and writes it atomically to dst.
func downloadGZ(ctx context.Context, url, dst string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer func() { _ = gr.Close() }()

	tmp := dst + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer func() {
		_ = f.Close()
		_ = os.Remove(tmp) // no-op if renamed
	}()

	if _, err := io.Copy(f, gr); err != nil {
		return fmt.Errorf("write MMDB: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	return os.Rename(tmp, dst)
}
