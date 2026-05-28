//go:build test

package testutils

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/netip"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/app"
	"github.com/DiegoGuidaF/PulseWeaver/internal/config"
)

const TestAdminPassword = "AdminPass123!"

// SetupIntegrationServer creates a complete integration test server with database,
// services, and handlers configured.
func SetupIntegrationServer(t *testing.T) *app.App {
	t.Helper()

	conf := &config.Conf{
		Server: config.ConfServer{
			Port:          2000,
			AdminPassword: TestAdminPassword,
			TrustedProxy:  netip.MustParseAddr("127.0.0.1"),
		},
		DB: config.ConfDB{
			// cache=shared is required so that multiple connections from the pool
			// share the same named in-memory database. Without it, each new
			// connection gets a fresh empty database — harmless for single-threaded
			// tests but breaks SetupRunningIntegrationServer which opens concurrent
			// connections from background service goroutines.
			Dsn: fmt.Sprintf("file:%s?mode=memory&cache=shared&_loc=auto", t.Name()),
		},
		Rules: config.ConfRules{
			CheckInterval: time.Minute,
		},
		Policy: config.ConfPolicy{
			APISecret: "test-policy-secret",
		},
	}

	logger := slog.New(slog.NewTextHandler(testWriter{t}, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
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

// SetupRunningIntegrationServer creates a complete integration test server and
// starts all background services (policy listener, lease runner, max-addr enforcer,
// scheduler, access log sink), matching the production startup of app.Run minus
// the HTTP server. The services stop when the test ends via t.Cleanup.
//
// Use this for cross-domain integration tests that exercise the reactive event
// pipeline (e.g. policy cache eviction after device deletion). For static-state
// tests that only assert on the outcome of a single operation, SetupIntegrationServer
// is sufficient and avoids the async-refresh concern entirely.
func SetupRunningIntegrationServer(t *testing.T) *app.App {
	t.Helper()
	srv := SetupIntegrationServer(t)
	go func() {
		if err := srv.RunBackground(t.Context()); err != nil {
			t.Errorf("background services error: %v", err)
		}
	}()
	return srv
}

type testWriter struct{ t testing.TB }

func (tw testWriter) Write(p []byte) (n int, err error) {
	// Trim the newline since t.Log automatically adds one
	tw.t.Log(string(bytes.TrimSpace(p)))
	return len(p), nil
}
