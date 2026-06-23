//go:build test

package testdb

import (
	"fmt"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/config"
	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
)

// Setup creates a new in-memory SQLite database for testing.
// Returns the database instance and a cleanup function.
// This package only imports database/config to avoid import cycles.
// Accepts testing.TB so benchmarks can share the same setup.
func Setup(t testing.TB) (*database.SQLite, func()) {
	t.Helper()

	conf := config.ConfDB{
		Dsn: fmt.Sprintf("file:%s?mode=memory&_loc=auto&_time_format=sqlite&_texttotime=1", t.Name()),
	}

	db, err := database.NewSQLite(conf)
	if err != nil {
		t.Fatalf("setup db: %v", err)
	}

	if err := db.Migrate(); err != nil {
		_ = db.Close()
		t.Fatalf("migrate: %v", err)
	}

	return db, func() {
		_ = db.Close()
	}
}
