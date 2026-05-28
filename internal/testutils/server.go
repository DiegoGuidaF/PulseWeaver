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

const (
	TestAdminPassword = "AdminPass123!"
	TestPolicySecret  = "test-policy-secret"
)

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
			// cache=shared: all pool connections share the same named in-memory DB.
			// Without it each new connection gets its own empty database, which
			// breaks SetupRunningIntegrationServer (concurrent background goroutines).
			// foreign_keys: enforce FK constraints, matching production behaviour.
			// busy_timeout: retry on SQLITE_BUSY instead of immediately failing under
			// light contention (e.g. background goroutines during seeding).
			Dsn: fmt.Sprintf("file:%s?mode=memory&cache=shared&_loc=auto&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)", t.Name()),
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

// StartBackground starts all background services (policy listener, lease runner,
// max-addr enforcer, scheduler, access log sink) for srv, tying their lifetime
// to t. Matches the production startup of app.Run minus the HTTP server.
//
// Call this AFTER seeding to avoid SQLite lock contention: background goroutines
// perform DB reads immediately on start, which can deadlock with concurrent seeder
// writes under SQLite's shared-cache locking model.
func StartBackground(t *testing.T, srv *app.App) {
	t.Helper()
	go func() {
		if err := srv.RunBackground(t.Context()); err != nil {
			t.Errorf("background services error: %v", err)
		}
	}()
}

// SetupRunningIntegrationServer creates a server, seeds via the provided Seeder,
// then starts all background services. This ordering prevents SQLite lock
// contention between the seeder and the background goroutines.
//
// Use this for cross-domain integration tests that exercise the reactive event
// pipeline (e.g. policy cache eviction after device deletion). For static-state
// tests, SetupIntegrationServer is sufficient.
func SetupRunningIntegrationServer(t *testing.T, seeder *Seeder) (*app.App, *SeedResult) {
	t.Helper()
	srv := SetupIntegrationServer(t)
	seed := seeder.Build(srv)
	StartBackground(t, srv)
	return srv, seed
}

type testWriter struct{ t testing.TB }

func (tw testWriter) Write(p []byte) (n int, err error) {
	// Trim the newline since t.Log automatically adds one
	tw.t.Log(string(bytes.TrimSpace(p)))
	return len(p), nil
}
