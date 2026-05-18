//go:build test

package useraccess_test

import (
	"context"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/hosts"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testdb"
	"github.com/DiegoGuidaF/PulseWeaver/internal/useraccess"
	"github.com/matryer/is"
)

type uaFixture struct {
	repo      *useraccess.Repository
	hostsRepo *hosts.Repository
	db        *database.DB
}

func setupUserAccessRepo(t *testing.T) uaFixture {
	t.Helper()
	sqlite, cleanup := testdb.Setup(t)
	t.Cleanup(cleanup)
	db := sqlite.DB()
	return uaFixture{
		repo:      useraccess.NewRepository(db),
		hostsRepo: hosts.NewRepository(db),
		db:        db,
	}
}

func insertUser(t *testing.T, db *database.DB, username string, bypass bool, deleted bool) ids.UserID {
	t.Helper()
	ctx := context.Background()
	var id ids.UserID

	deletedExpr := "NULL"
	if deleted {
		deletedExpr = "'2024-01-01 00:00:00'"
	}
	q := `INSERT INTO users (username, display_name, password_hash, role, deleted_at)
	      VALUES (?, ?, NULL, 'user', ` + deletedExpr + `) RETURNING id`
	if err := db.QueryRowxContext(ctx, q, username, username).Scan(&id); err != nil {
		t.Fatalf("insertUser(%q): %v", username, err)
	}

	if !deleted {
		bypassVal := 0
		if bypass {
			bypassVal = 1
		}
		sq := `INSERT INTO user_host_settings (user_id, bypass_host_check) VALUES (?, ?)`
		if _, err := db.ExecContext(ctx, sq, id, bypassVal); err != nil {
			t.Fatalf("insertUser(%q) settings: %v", username, err)
		}
	}
	return id
}

// ── SetUserAccess ─────────────────────────────────────────────────────────────

func TestRepository_SetUserAccess_HappyPath(t *testing.T) {
	is := is.New(t)
	fix := setupUserAccessRepo(t)
	ctx := context.Background()

	userID := insertUser(t, fix.db, "alice", false, false)
	hostID1, err := fix.hostsRepo.CreateHost(ctx, hosts.HostDraft{FQDN: "a.com"})
	is.NoErr(err)

	groupID, err := fix.hostsRepo.CreateHostGroup(ctx, hosts.HostGroupDraft{Name: "g", HostIDs: []ids.HostID{hostID1}})
	is.NoErr(err)

	err = fix.repo.SetUserAccess(ctx, userID, true, []ids.HostGroupID{groupID})
	is.NoErr(err)

	grants, err := fix.repo.GetAllUserHostGrants(ctx)
	is.NoErr(err)
	is.Equal(len(grants), 1)
	is.Equal(grants[0].FQDN, "a.com")

	settings, err := fix.repo.GetAllUserHostSettings(ctx)
	is.NoErr(err)
	is.Equal(len(settings), 1)
	is.True(settings[0].BypassHostCheck)
}

func TestRepository_SetUserAccess_ReplacesExistingAccess(t *testing.T) {
	is := is.New(t)
	fix := setupUserAccessRepo(t)
	ctx := context.Background()

	userID := insertUser(t, fix.db, "alice", false, false)

	hostID1, err := fix.hostsRepo.CreateHost(ctx, hosts.HostDraft{FQDN: "a.com"})
	is.NoErr(err)
	hostID2, err := fix.hostsRepo.CreateHost(ctx, hosts.HostDraft{FQDN: "b.com"})
	is.NoErr(err)

	hostGroupID1, err := fix.hostsRepo.CreateHostGroup(ctx, hosts.HostGroupDraft{Name: "g1", HostIDs: []ids.HostID{hostID1}})
	is.NoErr(err)
	hostGroupID2, err := fix.hostsRepo.CreateHostGroup(ctx, hosts.HostGroupDraft{Name: "g2", HostIDs: []ids.HostID{hostID2}})
	is.NoErr(err)

	is.NoErr(fix.repo.SetUserAccess(ctx, userID, false, []ids.HostGroupID{hostGroupID1}))
	is.NoErr(fix.repo.SetUserAccess(ctx, userID, false, []ids.HostGroupID{hostGroupID2}))

	grants, err := fix.repo.GetAllUserHostGrants(ctx)
	is.NoErr(err)
	is.Equal(len(grants), 1)
	is.Equal(grants[0].FQDN, "b.com")
}

// ── EnsureUserSettings ────────────────────────────────────────────────────────

func TestRepository_EnsureUserSettings_HappyPath(t *testing.T) {
	is := is.New(t)
	fix := setupUserAccessRepo(t)
	ctx := context.Background()

	userID := insertUser(t, fix.db, "no-settings", false, true)
	settings, err := fix.repo.GetAllUserHostSettings(ctx)
	is.Equal(len(settings), 0)
	is.NoErr(err)

	err = fix.repo.EnsureUserSettings(ctx, userID)
	is.NoErr(err)

	settings, err = fix.repo.GetAllUserHostSettings(ctx)
	is.NoErr(err)
	is.Equal(len(settings), 1)

	found := false
	for _, s := range settings {
		if s.UserID == userID {
			is.Equal(s.BypassHostCheck, false)
			found = true
		}
	}
	is.True(found)
}

func TestRepository_EnsureUserSettings_Idempotent(t *testing.T) {
	is := is.New(t)
	fix := setupUserAccessRepo(t)
	ctx := context.Background()

	userID := insertUser(t, fix.db, "alice", true, false)

	err := fix.repo.EnsureUserSettings(ctx, userID)
	is.NoErr(err)

	settings, err := fix.repo.GetAllUserHostSettings(ctx)
	is.NoErr(err)

	for _, s := range settings {
		if s.UserID == userID {
			is.Equal(s.BypassHostCheck, true) // unchanged
		}
	}
}

// ── DeleteUserData ────────────────────────────────────────────────────────────

func TestRepository_DeleteUserData_HappyPath(t *testing.T) {
	is := is.New(t)
	fix := setupUserAccessRepo(t)
	ctx := context.Background()

	userID := insertUser(t, fix.db, "alice", false, false)
	hostID1, err := fix.hostsRepo.CreateHost(ctx, hosts.HostDraft{FQDN: "a.com"})
	is.NoErr(err)

	groupID, err := fix.hostsRepo.CreateHostGroup(ctx, hosts.HostGroupDraft{Name: "g", HostIDs: []ids.HostID{hostID1}})
	is.NoErr(err)

	is.NoErr(fix.repo.SetUserAccess(ctx, userID, false, []ids.HostGroupID{groupID}))

	err = fix.repo.DeleteUserData(ctx, userID)
	is.NoErr(err)

	settings, err := fix.repo.GetAllUserHostSettings(ctx)
	is.NoErr(err)
	is.Equal(len(settings), 0)

	grants, err := fix.repo.GetAllUserHostGrants(ctx)
	is.NoErr(err)
	is.Equal(len(grants), 0)
}

// ── GetAllUserHostSettings ────────────────────────────────────────────────────

func TestRepository_GetAllUserHostSettings_Empty(t *testing.T) {
	is := is.New(t)
	fix := setupUserAccessRepo(t)

	result, err := fix.repo.GetAllUserHostSettings(context.Background())
	is.NoErr(err)
	is.Equal(len(result), 0)
}

func TestRepository_GetAllUserHostSettings_ReturnsBypassFlag(t *testing.T) {
	is := is.New(t)
	fix := setupUserAccessRepo(t)

	userA := insertUser(t, fix.db, "alice", false, false)
	userB := insertUser(t, fix.db, "bob", true, false)

	result, err := fix.repo.GetAllUserHostSettings(context.Background())
	is.NoErr(err)
	is.Equal(len(result), 2)

	byUser := make(map[ids.UserID]bool)
	for _, s := range result {
		byUser[s.UserID] = s.BypassHostCheck
	}
	is.Equal(byUser[userA], false)
	is.Equal(byUser[userB], true)
}

// ── GetAllUserHostGrants ──────────────────────────────────────────────────────

func TestRepository_GetAllUserHostGrants_Empty(t *testing.T) {
	is := is.New(t)
	fix := setupUserAccessRepo(t)

	result, err := fix.repo.GetAllUserHostGrants(context.Background())
	is.NoErr(err)
	is.Equal(len(result), 0)
}

func TestRepository_GetAllUserHostGrants_ReturnsGrants(t *testing.T) {
	is := is.New(t)
	fix := setupUserAccessRepo(t)
	ctx := context.Background()

	userID := insertUser(t, fix.db, "bob", false, false)
	hostID1, err := fix.hostsRepo.CreateHost(ctx, hosts.HostDraft{FQDN: "group-host.com"})
	is.NoErr(err)

	groupID, err := fix.hostsRepo.CreateHostGroup(ctx, hosts.HostGroupDraft{Name: "mygroup", HostIDs: []ids.HostID{hostID1}})
	is.NoErr(err)
	is.NoErr(fix.repo.SetUserAccess(ctx, userID, false, []ids.HostGroupID{groupID}))

	result, err := fix.repo.GetAllUserHostGrants(ctx)
	is.NoErr(err)
	is.Equal(len(result), 1)
	is.Equal(result[0].UserID, userID)
	is.Equal(result[0].FQDN, "group-host.com")
}
