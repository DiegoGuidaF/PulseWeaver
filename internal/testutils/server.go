//go:build test

package testutils

import (
	"context"
	"fmt"
	"log/slog"
	"net/netip"
	"testing"
	"time"

	"github.com/DiegoGuidaF/WallyDex/internal/app"
	"github.com/DiegoGuidaF/WallyDex/internal/config"
)

// SetupIntegrationServer creates a complete integration test server with database,
// services, and handlers configured.
func SetupIntegrationServer(t *testing.T) *app.App {
	t.Helper()

	conf := &config.Conf{
		Server: config.ConfServer{
			Port:          2000,
			AdminPassword: "AdminPass123!",
			TrustedProxy:  netip.MustParseAddr("127.0.0.1"),
		},
		DB: config.ConfDB{
			Dsn: fmt.Sprintf("file:%s?mode=memory&_loc=auto", t.Name()),
		},
		Rules: config.ConfRules{
			CheckInterval: time.Minute,
		},
		Policy: config.ConfPolicy{
			APISecret: "test-policy-secret",
		},
	}

	logger := slog.New(slog.DiscardHandler)
	ctx, cancel := context.WithCancel(context.Background())

	application, err := app.NewWithConfigAndLogger(ctx, conf, logger)
	if err != nil {
		cancel()
		t.Fatalf("setup app: %v", err)
	}

	t.Cleanup(func() {
		cancel() // cancel context first so background goroutines exit before Close waits on them
		if err := application.Close(); err != nil {
			t.Logf("error closing app: %v", err)
		}
	})

	return application
}
