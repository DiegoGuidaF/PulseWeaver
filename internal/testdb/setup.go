//go:build test

package testdb

import (
	"fmt"
	"testing"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/config"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/database"
)

// Setup creates a new in-memory SQLite database for testing.
// Returns the database instance and a cleanup function.
// This package only imports database/config to avoid import cycles.
func Setup(t *testing.T) (*database.SQLite, func()) {
	t.Helper()

	conf := config.ConfDB{
		Dsn:   fmt.Sprintf("file:%s?mode=memory&_loc=auto&_time_format=sqlite&_texttotime=1", t.Name()),
		Debug: false,
	}

	db, err := database.NewSQLite(conf)
	if err != nil {
		t.Fatalf("setup db: %v", err)
	}

	if err := db.Migrate(); err != nil {
		db.Close()
		t.Fatalf("migrate: %v", err)
	}

	return db, func() {
		db.Close()
	}
}
