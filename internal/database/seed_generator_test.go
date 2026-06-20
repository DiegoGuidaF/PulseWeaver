//go:build test && dbseed

// Package database_test hosts the seed-DB generator. It lives behind the
// `test && dbseed` build tags so it is excluded from `make test` (which builds
// `-tags=test` only) and from the production binary (neither tag). Being a test
// gives it a *testing.T, so it reuses testutils.SeedFullWorld with no refactor.
//
// Run via `make seed-db`, which sets SEED_OUT_DIR and writes a clean,
// latest-schema, self-contained SQLite artifact to db-test-seeds/seed-<ts>.db.
// Consumers load it by copying the file (no migrations — it is already at the
// latest schema). The bootstrap admin password is testutils.TestAdminPassword
// and the seeded admin (erin) uses testutils.SeededAdminPassword.
package database_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/app"
	"github.com/DiegoGuidaF/PulseWeaver/internal/config"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
)

// defaultAccessLogVolume gives the artifact enough access-log history to exercise
// pagination and the LOG audit cases. Override with SEED_ACCESS_LOG_VOLUME.
const defaultAccessLogVolume = 250

// TestGenerateSeedDB materialises SeedFullWorld into a file-backed SQLite DB.
//
// It boots a real app.App against a file DSN (journal_mode=DELETE → a single
// self-contained file, no -wal/-shm sidecars; GeoIP disabled so no GeoIP data is
// needed on disk). App construction runs db.Migrate(), so the file is at the
// latest schema and loaders never migrate. The output file is append-only: a new
// seed-<unixnano>.db is written each run and nothing is ever removed.
func TestGenerateSeedDB(t *testing.T) {
	outDir := os.Getenv("SEED_OUT_DIR")
	if outDir == "" {
		t.Fatal("SEED_OUT_DIR must be set to an absolute path (use `make seed-db`)")
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("create output dir %q: %v", outDir, err)
	}
	outPath := filepath.Join(outDir, fmt.Sprintf("seed-%d.db", time.Now().UnixNano()))

	conf := &config.Conf{
		Server: config.ConfServer{
			Port:          2000,
			AdminPassword: testutils.TestAdminPassword,
			TrustedProxy:  netip.MustParseAddr("127.0.0.1"),
		},
		DB: config.ConfDB{
			// seedDBDSN carries the production time-format params so seeded
			// timestamps are SQLite-parseable (see seed_dsn_test.go, PW-68).
			Dsn: seedDBDSN(outPath),
		},
		Rules:  config.ConfRules{CheckInterval: time.Minute},
		Policy: config.ConfPolicy{APISecret: testutils.TestPolicySecret},
		// GeoIP left zero-valued (Enabled=false) → geoip.New no-ops, no disk access.
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	application, err := app.NewWithConfigAndLogger(ctx, conf, logger)
	if err != nil {
		t.Fatalf("boot app: %v", err)
	}

	// No background services started — generation is a synchronous seed + flush.
	// SEED_WORLD selects the dataset: the default "full" world is the test-shaped
	// SeedFullWorld (stable entity counts for cross-domain query assertions);
	// "showcase" is the presentable demo world for screenshots and walkthroughs.
	switch os.Getenv("SEED_WORLD") {
	case "showcase":
		// Self-contained demo world; its traffic profile replaces the synthetic
		// access-log volume, so WithAccessLogVolume is intentionally not chained.
		testutils.SeedShowcaseWorld(t).Build(application)
	case "", "full":
		// SeedFullWorld plus extras the base world lacks: observed host suggestions
		// and an IPv6 grant + address. Chained here rather than added to SeedFullWorld
		// so the cross-domain query tests' entity-count assertions stay stable.
		testutils.SeedFullWorld(t).
			WithAccessLogVolume(accessLogVolume(t)).
			WithObservedHost("photos.internal", 3).        // real service to promote
			WithObservedHost("crawler-bot.junk", 2).       // crawler noise to ignore
			WithObservedHost("wp-login.php.scan.test", 1). // attack scan noise
			WithPolicy(testutils.FixturePolicyIPv6).
			WithAddress(testutils.FixtureAddressIPv6).
			Build(application)
	default:
		t.Fatalf("SEED_WORLD=%q is not recognised (use \"full\" or \"showcase\")", os.Getenv("SEED_WORLD"))
	}

	if err := application.Close(); err != nil {
		t.Fatalf("close app (flush DB): %v", err)
	}

	t.Logf("seed DB written: %s", outPath)
}

func accessLogVolume(t *testing.T) int {
	t.Helper()
	raw := os.Getenv("SEED_ACCESS_LOG_VOLUME")
	if raw == "" {
		return defaultAccessLogVolume
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		t.Fatalf("SEED_ACCESS_LOG_VOLUME must be a non-negative integer, got %q", raw)
	}
	return n
}
