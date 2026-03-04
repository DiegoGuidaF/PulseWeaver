//go:build test

package testutils

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/app"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/config"
)

// SetupIntegrationServer creates a complete integration test server with database,
// services, and handlers configured.
func SetupIntegrationServer(t *testing.T) *app.App {
	t.Helper()
	// Create a temporary directory that Go will automatically clean up
	tmpDir := t.TempDir()

	conf := &config.Conf{
		Server: config.ConfServer{
			Port:          2000,
			AdminPassword: "AdminPass123!",
			TrustedProxy:  "127.0.0.1",
		},
		DB: config.ConfDB{
			Dsn:   fmt.Sprintf("file:%s?mode=memory&_loc=auto", t.Name()),
			Debug: false,
		},
		Whitelist: config.ConfWhitelist{
			FilePath:  filepath.Join(tmpDir, "whitelist.txt"),
			RateLimit: 50 * time.Millisecond, // Fast debounce for tests
		},
		Rules: config.ConfRules{
			CheckInterval: time.Minute,
		},
	}

	logger := slog.New(slog.DiscardHandler)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	application, err := app.NewWithConfigAndLogger(ctx, conf, logger)
	if err != nil {
		t.Fatalf("setup app: %v", err)
	}

	t.Cleanup(func() {
		if err := application.Close(); err != nil {
			t.Logf("error closing app: %v", err)
		}
	})

	return application
}
