//go:build test

package hostaccess_test

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/hostaccess"
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
// For active users a user_host_settings row is also created (mirroring the observer behaviour).
// Deleted users intentionally have no settings row, simulating post-observer cleanup.
func insertUser(t *testing.T, db *database.DB, username string, bypass bool, deleted bool) auth.UserID {
	t.Helper()
	ctx := context.Background()
	var id auth.UserID

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
		sq := `INSERT INTO user_host_settings (user_id, bypass_host_allowlist) VALUES (?, ?)`
		if _, err := db.ExecContext(ctx, sq, id, bypassVal); err != nil {
			t.Fatalf("insertUser(%q) settings: %v", username, err)
		}
	}
	return id
}

// ── BulkCreateKnownHosts ──────────────────────────────────────────────────────
func TestRepository_BulkCreateKnownHosts_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)

	hosts, err := repo.BulkCreateKnownHosts(context.Background(), []string{"Example.COM", "test.org"})
	is.NoErr(err)
	is.Equal(len(hosts), 2)
	is.Equal(hosts[0].FQDN, "example.com") // lowercased
	is.Equal(hosts[1].FQDN, "test.org")
	is.True(hosts[0].ID > 0)
	is.True(hosts[1].ID > 0)
	is.True(hosts[0].ID != hosts[1].ID)
}

// ── UpdateKnownHost ──────────────────────────────────────────────────────────
func TestRepository_UpdateKnownHost_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)
	ctx := context.Background()

	hosts, err := repo.BulkCreateKnownHosts(ctx, []string{"example.com"})
	is.NoErr(err)

	icon := "🌐"
	updated, err := repo.UpdateKnownHost(ctx, hosts[0].ID, &icon)
	is.NoErr(err)
	is.Equal(updated.ID, hosts[0].ID)
	is.Equal(updated.FQDN, "example.com")
	is.Equal(*updated.Icon, "🌐")
	is.True(updated.UpdatedAt.After(hosts[0].CreatedAt) || updated.UpdatedAt.Equal(hosts[0].CreatedAt))
}

// ── DeleteKnownHost ──────────────────────────────────────────────────────────
func TestRepository_DeleteKnownHost_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)
	ctx := context.Background()

	hosts, err := repo.BulkCreateKnownHosts(ctx, []string{"example.com"})
	is.NoErr(err)

	err = repo.DeleteKnownHost(ctx, hosts[0].ID)
	is.NoErr(err)

	// Verify it's gone by trying to update it.
	_, err = repo.UpdateKnownHost(ctx, hosts[0].ID, nil)
	is.True(errors.Is(err, hostaccess.ErrKnownHostNotFound))
}

// ── CreateHostGroupWithMembers ───────────────────────────────────────────────
func TestRepository_CreateHostGroupWithMembers_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)
	ctx := context.Background()

	hosts, err := repo.BulkCreateKnownHosts(ctx, []string{"a.com", "b.com"})
	is.NoErr(err)

	desc := "test group"
	groupID, err := repo.CreateHostGroupWithMembers(ctx, "mygroup", &desc, nil, []hostaccess.KnownHostID{hosts[0].ID, hosts[1].ID})
	is.NoErr(err)
	is.True(groupID > 0)
}

func TestRepository_CreateHostGroupWithMembers_EmptyMembers(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)

	groupID, err := repo.CreateHostGroupWithMembers(context.Background(), "empty-group", nil, nil, nil)
	is.NoErr(err)
	is.True(groupID > 0)
}

// ── UpdateHostGroupWithMembers ───────────────────────────────────────────────
func TestRepository_UpdateHostGroupWithMembers_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)
	ctx := context.Background()

	hosts, err := repo.BulkCreateKnownHosts(ctx, []string{"a.com", "b.com"})
	is.NoErr(err)

	groupID, err := repo.CreateHostGroupWithMembers(ctx, "original", nil, nil, []hostaccess.KnownHostID{hosts[0].ID})
	is.NoErr(err)

	desc := "updated"
	err = repo.UpdateHostGroupWithMembers(ctx, groupID, "renamed", &desc, nil, []hostaccess.KnownHostID{hosts[1].ID})
	is.NoErr(err)
}

// ── UpdateHostGroupMetadata ──────────────────────────────────────────────────
func TestRepository_UpdateHostGroupMetadata_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)
	ctx := context.Background()

	groupID, err := repo.CreateHostGroupWithMembers(ctx, "original", nil, nil, nil)
	is.NoErr(err)

	desc := "new desc"
	icon := "🔒"
	err = repo.UpdateHostGroupMetadata(ctx, groupID, "renamed", &desc, &icon)
	is.NoErr(err)
}

// ── DeleteHostGroup ──────────────────────────────────────────────────────────
func TestRepository_DeleteHostGroup_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)
	ctx := context.Background()

	groupID, err := repo.CreateHostGroupWithMembers(ctx, "to-delete", nil, nil, nil)
	is.NoErr(err)

	err = repo.DeleteHostGroup(ctx, groupID)
	is.NoErr(err)

	// Verify it's gone.
	err = repo.UpdateHostGroupMetadata(ctx, groupID, "gone", nil, nil)
	is.True(errors.Is(err, hostaccess.ErrHostGroupNotFound))
}

// ── SetFullUserGrants ────────────────────────────────────────────────────────
func TestRepository_SetFullUserGrants_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, db := setupHostAccessRepo(t)
	ctx := context.Background()

	userID := insertUser(t, db, "alice", false, false)
	hosts, err := repo.BulkCreateKnownHosts(ctx, []string{"a.com", "b.com"})
	is.NoErr(err)
	groupID, err := repo.CreateHostGroupWithMembers(ctx, "g", nil, nil, []hostaccess.KnownHostID{hosts[0].ID})
	is.NoErr(err)

	bypass := true
	err = repo.SetFullUserGrants(ctx, userID, &bypass,
		[]hostaccess.KnownHostID{hosts[1].ID},
		[]hostaccess.HostGroupID{groupID},
	)
	is.NoErr(err)

	// Verify grants are visible via the feed queries.
	direct, err := repo.GetAllUserDirectHostGrants(ctx)
	is.NoErr(err)
	is.Equal(len(direct), 1)
	is.Equal(direct[0].FQDN, "b.com")

	group, err := repo.GetAllUserGroupHostGrants(ctx)
	is.NoErr(err)
	is.Equal(len(group), 1)
	is.Equal(group[0].FQDN, "a.com")

	settings, err := repo.GetAllUserHostSettings(ctx)
	is.NoErr(err)
	byUser := make(map[auth.UserID]bool)
	for _, s := range settings {
		byUser[s.UserID] = s.BypassAllowlist
	}
	is.Equal(byUser[userID], true)
}

func TestRepository_SetFullUserGrants_ReplacesExistingGrants(t *testing.T) {
	is := is.New(t)
	repo, db := setupHostAccessRepo(t)
	ctx := context.Background()

	userID := insertUser(t, db, "alice", false, false)
	hosts, err := repo.BulkCreateKnownHosts(ctx, []string{"a.com", "b.com"})
	is.NoErr(err)

	// First grant: a.com
	is.NoErr(repo.SetFullUserGrants(ctx, userID, nil, []hostaccess.KnownHostID{hosts[0].ID}, nil))

	// Second grant: b.com (should replace, not append)
	is.NoErr(repo.SetFullUserGrants(ctx, userID, nil, []hostaccess.KnownHostID{hosts[1].ID}, nil))

	direct, err := repo.GetAllUserDirectHostGrants(ctx)
	is.NoErr(err)
	is.Equal(len(direct), 1)
	is.Equal(direct[0].FQDN, "b.com")
}

// ── AddIgnoredSuggestion ─────────────────────────────────────────────────────
func TestRepository_AddIgnoredSuggestion_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)

	s, err := repo.AddIgnoredSuggestion(context.Background(), "Spam.Example.COM")
	is.NoErr(err)
	is.True(s.ID > 0)
	is.Equal(s.FQDN, "spam.example.com") // lowercased
	is.True(!s.CreatedAt.IsZero())
}

// ── RemoveIgnoredSuggestionByFQDN ────────────────────────────────────────────
func TestRepository_RemoveIgnoredSuggestionByFQDN_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)
	ctx := context.Background()

	_, err := repo.AddIgnoredSuggestion(ctx, "spam.example.com")
	is.NoErr(err)

	err = repo.RemoveIgnoredSuggestionByFQDN(ctx, "spam.example.com")
	is.NoErr(err)

	// Verify it's gone.
	err = repo.RemoveIgnoredSuggestionByFQDN(ctx, "spam.example.com")
	is.True(errors.Is(err, hostaccess.ErrSuggestionNotFound))
}

// ── EnsureUserSettings ───────────────────────────────────────────────────────
func TestRepository_EnsureUserSettings_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, db := setupHostAccessRepo(t)
	ctx := context.Background()

	// Insert user without settings (deleted=true skips settings in our helper).
	userID := insertUser(t, db, "no-settings", false, true)

	err := repo.EnsureUserSettings(ctx, userID)
	is.NoErr(err)

	// Should now appear in settings.
	settings, err := repo.GetAllUserHostSettings(ctx)
	is.NoErr(err)

	found := false
	for _, s := range settings {
		if s.UserID == userID {
			is.Equal(s.BypassAllowlist, false) // default is 0
			found = true
		}
	}
	is.True(found)
}

func TestRepository_EnsureUserSettings_Idempotent(t *testing.T) {
	is := is.New(t)
	repo, db := setupHostAccessRepo(t)
	ctx := context.Background()

	userID := insertUser(t, db, "alice", true, false) // bypass=true

	// Calling again should not error or overwrite.
	err := repo.EnsureUserSettings(ctx, userID)
	is.NoErr(err)

	settings, err := repo.GetAllUserHostSettings(ctx)
	is.NoErr(err)

	for _, s := range settings {
		if s.UserID == userID {
			is.Equal(s.BypassAllowlist, true) // unchanged
		}
	}
}

// ── DeleteUserData ───────────────────────────────────────────────────────────

func TestRepository_DeleteUserData_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, db := setupHostAccessRepo(t)
	ctx := context.Background()

	userID := insertUser(t, db, "alice", false, false)
	hosts, err := repo.BulkCreateKnownHosts(ctx, []string{"a.com"})
	is.NoErr(err)
	groupID, err := repo.CreateHostGroupWithMembers(ctx, "g", nil, nil, []hostaccess.KnownHostID{hosts[0].ID})
	is.NoErr(err)
	is.NoErr(repo.SetFullUserGrants(ctx, userID, nil, []hostaccess.KnownHostID{hosts[0].ID}, []hostaccess.HostGroupID{groupID}))

	err = repo.DeleteUserData(ctx, userID)
	is.NoErr(err)

	// Settings, direct grants, and group grants should all be gone.
	settings, err := repo.GetAllUserHostSettings(ctx)
	is.NoErr(err)
	is.Equal(len(settings), 0)

	direct, err := repo.GetAllUserDirectHostGrants(ctx)
	is.NoErr(err)
	is.Equal(len(direct), 0)

	group, err := repo.GetAllUserGroupHostGrants(ctx)
	is.NoErr(err)
	is.Equal(len(group), 0)
}

// ── GetAllUserHostSettings ────────────────────────────────────────────────────

func TestRepository_GetAllUserHostSettings_Empty(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)

	result, err := repo.GetAllUserHostSettings(context.Background())
	is.NoErr(err)
	is.Equal(len(result), 0)
}

func TestRepository_GetAllUserHostSettings_ReturnsBypassFlag(t *testing.T) {
	is := is.New(t)
	repo, db := setupHostAccessRepo(t)

	userA := insertUser(t, db, "alice", false, false)
	userB := insertUser(t, db, "bob", true, false)

	result, err := repo.GetAllUserHostSettings(context.Background())
	is.NoErr(err)
	is.Equal(len(result), 2)

	byUser := make(map[auth.UserID]bool)
	for _, s := range result {
		byUser[s.UserID] = s.BypassAllowlist
	}
	is.Equal(byUser[userA], false)
	is.Equal(byUser[userB], true)
}

// ── GetAllUserDirectHostGrants ────────────────────────────────────────────────

func TestRepository_GetAllUserDirectHostGrants_Empty(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)

	result, err := repo.GetAllUserDirectHostGrants(context.Background())
	is.NoErr(err)
	is.Equal(len(result), 0)
}

func TestRepository_GetAllUserDirectHostGrants_ReturnsGrants(t *testing.T) {
	is := is.New(t)
	repo, db := setupHostAccessRepo(t)
	ctx := context.Background()

	userID := insertUser(t, db, "alice", false, false)
	hosts, err := repo.BulkCreateKnownHosts(ctx, []string{"example.com", "test.com"})
	is.NoErr(err)
	is.NoErr(repo.SetFullUserGrants(ctx, userID, nil, []hostaccess.KnownHostID{hosts[0].ID, hosts[1].ID}, nil))

	result, err := repo.GetAllUserDirectHostGrants(ctx)
	is.NoErr(err)
	is.Equal(len(result), 2)

	fqdns := make(map[string]bool)
	for _, g := range result {
		is.Equal(g.UserID, userID)
		fqdns[g.FQDN] = true
	}
	is.True(fqdns["example.com"])
	is.True(fqdns["test.com"])
}

func TestRepository_GetAllUserDirectHostGrants_MultipleUsers(t *testing.T) {
	is := is.New(t)
	repo, db := setupHostAccessRepo(t)
	ctx := context.Background()

	userA := insertUser(t, db, "alice", false, false)
	userB := insertUser(t, db, "bob", false, false)

	hosts, err := repo.BulkCreateKnownHosts(ctx, []string{"a.com", "b.com", "c.com"})
	is.NoErr(err)
	is.NoErr(repo.SetFullUserGrants(ctx, userA, nil, []hostaccess.KnownHostID{hosts[0].ID, hosts[1].ID}, nil))
	is.NoErr(repo.SetFullUserGrants(ctx, userB, nil, []hostaccess.KnownHostID{hosts[2].ID}, nil))

	result, err := repo.GetAllUserDirectHostGrants(ctx)
	is.NoErr(err)
	is.Equal(len(result), 3)

	byUser := make(map[auth.UserID][]string)
	for _, g := range result {
		byUser[g.UserID] = append(byUser[g.UserID], g.FQDN)
	}
	sort.Strings(byUser[userA])
	is.Equal(byUser[userA], []string{"a.com", "b.com"})
	is.Equal(byUser[userB], []string{"c.com"})
}

func TestRepository_GetAllUserDirectHostGrants_ExcludesDeletedUsers(t *testing.T) {
	is := is.New(t)
	repo, db := setupHostAccessRepo(t)
	ctx := context.Background()

	// Create user, grant host, then simulate deletion (remove settings row).
	userID := insertUser(t, db, "deleted-user", false, false)
	hosts, err := repo.BulkCreateKnownHosts(ctx, []string{"example.com"})
	is.NoErr(err)
	is.NoErr(repo.SetFullUserGrants(ctx, userID, nil, []hostaccess.KnownHostID{hosts[0].ID}, nil))

	// Remove settings row to simulate deleted user.
	_, err = db.ExecContext(ctx, `DELETE FROM user_host_settings WHERE user_id = ?`, userID)
	is.NoErr(err)

	result, err := repo.GetAllUserDirectHostGrants(ctx)
	is.NoErr(err)
	is.Equal(len(result), 0)
}

// ── GetAllUserGroupHostGrants ─────────────────────────────────────────────────

func TestRepository_GetAllUserGroupHostGrants_Empty(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)

	result, err := repo.GetAllUserGroupHostGrants(context.Background())
	is.NoErr(err)
	is.Equal(len(result), 0)
}

func TestRepository_GetAllUserGroupHostGrants_ReturnsGrants(t *testing.T) {
	is := is.New(t)
	repo, db := setupHostAccessRepo(t)
	ctx := context.Background()

	userID := insertUser(t, db, "bob", false, false)
	hosts, err := repo.BulkCreateKnownHosts(ctx, []string{"group-host.com"})
	is.NoErr(err)

	groupID, err := repo.CreateHostGroupWithMembers(ctx, "mygroup", nil, nil, []hostaccess.KnownHostID{hosts[0].ID})
	is.NoErr(err)
	is.NoErr(repo.SetFullUserGrants(ctx, userID, nil, nil, []hostaccess.HostGroupID{groupID}))

	result, err := repo.GetAllUserGroupHostGrants(ctx)
	is.NoErr(err)
	is.Equal(len(result), 1)
	is.Equal(result[0].UserID, userID)
	is.Equal(result[0].FQDN, "group-host.com")
}

func TestRepository_GetAllUserGroupHostGrants_MultipleUsers(t *testing.T) {
	is := is.New(t)
	repo, db := setupHostAccessRepo(t)
	ctx := context.Background()

	userA := insertUser(t, db, "alice", false, false)
	userB := insertUser(t, db, "bob", false, false)

	hosts, err := repo.BulkCreateKnownHosts(ctx, []string{"g1.com", "g2.com"})
	is.NoErr(err)

	group1, err := repo.CreateHostGroupWithMembers(ctx, "group1", nil, nil, []hostaccess.KnownHostID{hosts[0].ID})
	is.NoErr(err)
	group2, err := repo.CreateHostGroupWithMembers(ctx, "group2", nil, nil, []hostaccess.KnownHostID{hosts[0].ID, hosts[1].ID})
	is.NoErr(err)

	is.NoErr(repo.SetFullUserGrants(ctx, userA, nil, nil, []hostaccess.HostGroupID{group1}))
	is.NoErr(repo.SetFullUserGrants(ctx, userB, nil, nil, []hostaccess.HostGroupID{group2}))

	result, err := repo.GetAllUserGroupHostGrants(ctx)
	is.NoErr(err)

	byUser := make(map[auth.UserID][]string)
	for _, g := range result {
		byUser[g.UserID] = append(byUser[g.UserID], g.FQDN)
	}
	is.Equal(byUser[userA], []string{"g1.com"})
	sort.Strings(byUser[userB])
	is.Equal(byUser[userB], []string{"g1.com", "g2.com"})
}

func TestRepository_GetAllUserGroupHostGrants_ExcludesDeletedUsers(t *testing.T) {
	is := is.New(t)
	repo, db := setupHostAccessRepo(t)
	ctx := context.Background()

	userID := insertUser(t, db, "deleted-user", false, false)
	hosts, err := repo.BulkCreateKnownHosts(ctx, []string{"group-host.com"})
	is.NoErr(err)

	groupID, err := repo.CreateHostGroupWithMembers(ctx, "mygroup", nil, nil, []hostaccess.KnownHostID{hosts[0].ID})
	is.NoErr(err)
	is.NoErr(repo.SetFullUserGrants(ctx, userID, nil, nil, []hostaccess.HostGroupID{groupID}))

	_, err = db.ExecContext(ctx, `DELETE FROM user_host_settings WHERE user_id = ?`, userID)
	is.NoErr(err)

	result, err := repo.GetAllUserGroupHostGrants(ctx)
	is.NoErr(err)
	is.Equal(len(result), 0)
}
