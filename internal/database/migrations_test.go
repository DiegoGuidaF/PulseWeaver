//go:build test

package database

import (
	_ "embed"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/config"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migration_test_seed.sql
var migrationTestSeedSQL string

// TestSQLiteMigrate_Idempotent verifies that calling Migrate twice on the same DB
// is safe and does not return an error (second run should be a no-op). This also
// implicitly exercises that the migration history is consistent for repeated runs.
func TestSQLiteMigrate_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "idempotent.db")

	sqliteDB, err := NewSQLite(config.ConfDB{Dsn: fmt.Sprintf("file:%s?_time_format=sqlite&_texttotime=1&_timezone=UTC", dbPath)})
	if err != nil {
		t.Fatalf("NewSQLite: %v", err)
	}
	t.Cleanup(func() {
		_ = sqliteDB.Close()
	})

	if err := sqliteDB.Migrate(); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}

	// Second migrate should also succeed (ErrNoChange is swallowed inside Migrate).
	if err := sqliteDB.Migrate(); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
}

// TestMigrations_DownToZeroAndBackUp verifies that we can migrate all the way
// down to version 0 and then back up again on the same database.
func TestMigrations_DownToZeroAndBackUp(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "down-up.db")

	sqliteDB, err := NewSQLite(config.ConfDB{Dsn: fmt.Sprintf("file:%s?_time_format=sqlite&_texttotime=1&_timezone=UTC", dbPath)})
	if err != nil {
		t.Fatalf("NewSQLite: %v", err)
	}
	t.Cleanup(func() {
		_ = sqliteDB.Close()
	})

	// Apply all up migrations using the production helper.
	if err := sqliteDB.Migrate(); err != nil {
		t.Fatalf("Migrate (up): %v", err)
	}

	// Re-create the migrator against the same DB so we can exercise Down().
	driver, err := sqlite.WithInstance(sqliteDB.DB().DB, &sqlite.Config{})
	if err != nil {
		t.Fatalf("create driver: %v", err)
	}

	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		t.Fatalf("load migrations: %v", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "sqlite", driver)
	if err != nil {
		t.Fatalf("create migrator: %v", err)
	}
	t.Cleanup(func() {
		_, _ = m.Close()
	})

	// Migrate all the way down to version 0.
	for {
		err = m.Down()
		if err != nil {
			if errors.Is(err, migrate.ErrNoChange) {
				break
			}
			t.Fatalf("Down: %v", err)
		}
	}

	// Running Up again after Down should succeed (or report ErrNoChange if already current).
	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		t.Fatalf("Up after Down: %v", err)
	}
}

// TestMigrations_FinalMigration_WithData rolls back the latest migration, seeds
// representative rows, then re-applies it. This catches bugs that only surface
// when tables already contain data — e.g. ALTER TABLE ADD COLUMN NOT NULL with a
// non-constant DEFAULT fails silently on an empty table but errors at runtime.
//
// When a new migration is added: check whether seedBeforeLatestMigration needs
// updating to insert valid rows for the new penultimate schema.
func TestMigrations_FinalMigration_WithData(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "seeded.db")

	sqliteDB, err := NewSQLite(config.ConfDB{Dsn: fmt.Sprintf("file:%s?_time_format=sqlite&_texttotime=1&_timezone=UTC", dbPath)})
	if err != nil {
		t.Fatalf("NewSQLite: %v", err)
	}
	t.Cleanup(func() { _ = sqliteDB.Close() })

	if err := sqliteDB.Migrate(); err != nil {
		t.Fatalf("Migrate (up): %v", err)
	}

	driver, err := sqlite.WithInstance(sqliteDB.DB().DB, &sqlite.Config{})
	if err != nil {
		t.Fatalf("create driver: %v", err)
	}
	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		t.Fatalf("load migrations: %v", err)
	}
	m, err := migrate.NewWithInstance("iofs", source, "sqlite", driver)
	if err != nil {
		t.Fatalf("create migrator: %v", err)
	}
	t.Cleanup(func() { _, _ = m.Close() })

	// Roll back the latest migration so the DB is at schema N-1.
	if err := m.Steps(-1); err != nil {
		t.Fatalf("Steps(-1): %v", err)
	}

	// Seed representative rows valid for the N-1 schema.
	seedBeforeLatestMigration(t, sqliteDB)

	// Re-apply the latest migration against the seeded data.
	if err := m.Steps(1); err != nil {
		t.Fatalf("Steps(+1) with data present: %v", err)
	}

	// Verify seed data survived intact.
	var count int
	if err := sqliteDB.DB().Get(&count, `SELECT COUNT(*) FROM devices WHERE name = 'seed-router'`); err != nil {
		t.Fatalf("verify seed device: %v", err)
	}
	if count != 1 {
		t.Fatalf("seed device did not survive migration: want 1 row, got %d", count)
	}
}

// seedBeforeLatestMigration executes migration_test_seed.sql against the DB.
// The seed file contains one representative row per table in FK dependency order,
// valid for the penultimate schema version (after Steps(-1)).
func seedBeforeLatestMigration(t *testing.T, db *SQLite) {
	t.Helper()
	if _, err := db.DB().Exec(migrationTestSeedSQL); err != nil {
		t.Fatalf("seed: %v", err)
	}
}
