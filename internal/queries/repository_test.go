//go:build test

package queries_test

import (
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/accesslog"
	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/lease"
	"github.com/DiegoGuidaF/PulseWeaver/internal/queries"
	"github.com/DiegoGuidaF/PulseWeaver/internal/rule"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testdb"
)

// testRepos groups all repositories used by the queries package tests.
type testRepos struct {
	queries     *queries.Repository
	devices     *device.Repository
	leases      *lease.Repository
	rules       *rule.Repository
	accessLog   *accesslog.Repository
	db          *database.DB
	testOwnerID ids.UserID
}

// setupRepos creates an in-memory SQLite DB and returns all repositories sharing it.
func setupRepos(t *testing.T) testRepos {
	t.Helper()

	dbWrapper, cleanup := testdb.Setup(t)
	t.Cleanup(cleanup)

	sqlxDB := dbWrapper.DB()

	// Insert a test owner user (all devices need an owner since migration 000010).
	var ownerID ids.UserID
	err := sqlxDB.QueryRowxContext(
		t.Context(),
		`INSERT INTO users (username, display_name, password_hash, role) VALUES ('testadmin', 'Test Admin', 'x', 'admin') RETURNING id`,
	).Scan(&ownerID)
	if err != nil {
		t.Fatalf("setupRepos: insert test user: %v", err)
	}

	return testRepos{
		queries:     queries.NewRepository(sqlxDB),
		devices:     device.NewRepository(sqlxDB),
		leases:      lease.NewRepository(sqlxDB),
		rules:       rule.NewRepository(sqlxDB),
		accessLog:   accesslog.NewRepository(sqlxDB),
		db:          sqlxDB,
		testOwnerID: ownerID,
	}
}
