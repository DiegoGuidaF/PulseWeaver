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
	driver, err := sqlite.WithInstance(sqliteDB.DB().pool.DB, &sqlite.Config{NoTxWrap: true})
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

// TestMigrations_FinalMigration_WithData seeds representative rows at schema N
// (the latest), then rolls back and re-applies the final migration. This catches
// bugs that only surface when tables already contain data — e.g. a migration that
// changes a column type but fails to cast existing values, or a NOT NULL column
// added without a proper DEFAULT.
//
// The seed file should cover every table with diverse, realistic values so that
// any new migration is automatically validated against pre-existing data. Update
// the seed when your migration changes the schema (adds/removes tables or columns)
// so it stays insertable at the latest schema.
//
// Tables introduced by the latest migration are dropped during rollback and
// recreated empty on re-apply — that is expected and matches a real upgrade path.
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

	// Seed representative rows at schema N (latest).
	seedAtLatestSchema(t, sqliteDB)

	driver, err := sqlite.WithInstance(sqliteDB.DB().pool.DB, &sqlite.Config{NoTxWrap: true})
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

	// Roll back the latest migration (schema N → N-1).
	if err := m.Steps(-1); err != nil {
		t.Fatalf("Steps(-1): %v", err)
	}

	// Re-apply the latest migration against the rolled-back data (N-1 → N).
	// If this succeeds, the migration correctly handles pre-existing data —
	// type casts, constraints, and FK integrity are all validated by the
	// migration itself (foreign_keys is ON and migrations run foreign_key_check).
	if err := m.Steps(1); err != nil {
		t.Fatalf("Steps(+1) with data present: %v", err)
	}
}

// seedAtLatestSchema executes migration_test_seed.sql against the DB at schema N.
// The seed file contains representative rows for every table in FK dependency order.
func seedAtLatestSchema(t *testing.T, db *SQLite) {
	t.Helper()
	if _, err := db.DB().ExecContext(t.Context(), migrationTestSeedSQL); err != nil {
		t.Fatalf("seed: %v", err)
	}
}
