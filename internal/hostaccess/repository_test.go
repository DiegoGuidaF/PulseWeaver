//go:build test

package hostaccess_test

import (
	"context"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/hostaccess"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testdb"
	"github.com/matryer/is"
)

func setupHostAccessRepo(t *testing.T) (*hostaccess.Repository, *database.DB) {
	t.Helper()
	db, cleanup := testdb.Setup(t)
	t.Cleanup(cleanup)
	return hostaccess.NewRepository(db.DB()), db.DB()
}

// insertUser inserts a raw user row and returns its ID.
func insertUser(t *testing.T, db *database.DB, username string, bypass bool, deleted bool) auth.UserID {
	t.Helper()
	var id auth.UserID
	deletedExpr := "NULL"
	if deleted {
		deletedExpr = "'2024-01-01 00:00:00'"
	}
	bypassVal := 0
	if bypass {
		bypassVal = 1
	}
	q := `INSERT INTO users (username, display_name, password_hash, role, bypass_host_allowlist, deleted_at)
	      VALUES (?, ?, NULL, 'user', ?, ` + deletedExpr + `) RETURNING id`
	err := db.QueryRowxContext(context.Background(), q, username, username, bypassVal).Scan(&id)
	if err != nil {
		t.Fatalf("insertUser(%q): %v", username, err)
	}
	return id
}

func TestRepository_GetAllUserHostAccess_Empty(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)

	result, err := repo.GetAllUserHostAccess(context.Background())
	is.NoErr(err)
	is.Equal(len(result), 0)
}

func TestRepository_GetAllUserHostAccess_DirectGrant(t *testing.T) {
	is := is.New(t)
	repo, db := setupHostAccessRepo(t)
	ctx := context.Background()

	userID := insertUser(t, db, "alice", false, false)

	host, err := repo.CreateKnownHost(ctx, "example.com", nil)
	is.NoErr(err)
	is.NoErr(repo.GrantUserHost(ctx, userID, host.ID))

	result, err := repo.GetAllUserHostAccess(ctx)
	is.NoErr(err)
	is.Equal(len(result), 1)
	is.Equal(result[0].UserID, userID)
	is.True(!result[0].BypassAllowlist)
	is.Equal(len(result[0].AllowedHosts), 1)
	is.Equal(result[0].AllowedHosts[0], "example.com")
}

func TestRepository_GetAllUserHostAccess_GroupGrant(t *testing.T) {
	is := is.New(t)
	repo, db := setupHostAccessRepo(t)
	ctx := context.Background()

	userID := insertUser(t, db, "bob", false, false)

	host, err := repo.CreateKnownHost(ctx, "group-host.com", nil)
	is.NoErr(err)

	group, err := repo.CreateHostGroup(ctx, "mygroup", nil, nil)
	is.NoErr(err)
	is.NoErr(repo.AddHostToGroup(ctx, group.ID, host.ID))
	is.NoErr(repo.GrantUserHostGroup(ctx, userID, group.ID))

	result, err := repo.GetAllUserHostAccess(ctx)
	is.NoErr(err)
	is.Equal(len(result), 1)
	is.Equal(result[0].UserID, userID)
	is.Equal(len(result[0].AllowedHosts), 1)
	is.Equal(result[0].AllowedHosts[0], "group-host.com")
}

func TestRepository_GetAllUserHostAccess_BypassOnly(t *testing.T) {
	is := is.New(t)
	repo, db := setupHostAccessRepo(t)
	ctx := context.Background()

	userID := insertUser(t, db, "carol", true, false)

	result, err := repo.GetAllUserHostAccess(ctx)
	is.NoErr(err)
	is.Equal(len(result), 1)
	is.Equal(result[0].UserID, userID)
	is.True(result[0].BypassAllowlist)
	is.Equal(len(result[0].AllowedHosts), 0)
}

func TestRepository_GetAllUserHostAccess_ExcludesDeletedUsers(t *testing.T) {
	is := is.New(t)
	repo, db := setupHostAccessRepo(t)
	ctx := context.Background()

	// A deleted user with bypass=true — must not appear.
	insertUser(t, db, "deleted-user", true, true)

	result, err := repo.GetAllUserHostAccess(ctx)
	is.NoErr(err)
	is.Equal(len(result), 0)
}

func TestRepository_GetAllUserHostAccess_DirectAndGroupDeduplication(t *testing.T) {
	// Same host granted both directly and via group — must appear once.
	is := is.New(t)
	repo, db := setupHostAccessRepo(t)
	ctx := context.Background()

	userID := insertUser(t, db, "dave", false, false)

	host, err := repo.CreateKnownHost(ctx, "shared.com", nil)
	is.NoErr(err)

	// Direct grant.
	is.NoErr(repo.GrantUserHost(ctx, userID, host.ID))

	// Group grant for same host.
	group, err := repo.CreateHostGroup(ctx, "g", nil, nil)
	is.NoErr(err)
	is.NoErr(repo.AddHostToGroup(ctx, group.ID, host.ID))
	is.NoErr(repo.GrantUserHostGroup(ctx, userID, group.ID))

	result, err := repo.GetAllUserHostAccess(ctx)
	is.NoErr(err)
	is.Equal(len(result), 1)

	// UNION removes duplicates — AllowedHosts should have exactly one "shared.com".
	var count int
	for _, h := range result[0].AllowedHosts {
		if h == "shared.com" {
			count++
		}
	}
	is.Equal(count, 1)
}

// ensure Repository satisfies the HostAccessProvider interface used by policy.
var _ interface {
	GetAllUserHostAccess(ctx context.Context) ([]policy.UserHostAccess, error)
} = (*hostaccess.Repository)(nil)
