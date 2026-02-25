package database

import (
	"errors"
	"path/filepath"
	"testing"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/config"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// TestSQLiteMigrate_Idempotent verifies that calling Migrate twice on the same DB
// is safe and does not return an error (second run should be a no-op). This also
// implicitly exercises that the migration history is consistent for repeated runs.
func TestSQLiteMigrate_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "idempotent.db")

	sqliteDB, err := NewSQLite(config.ConfDB{File: dbPath})
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

	sqliteDB, err := NewSQLite(config.ConfDB{File: dbPath})
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
		m.Close()
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

