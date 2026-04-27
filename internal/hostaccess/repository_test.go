//go:build test

package hostaccess_test

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

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

// // createGroup is a thin helper that adapts the repo's draft-shaped API to the
// // positional shape the legacy repository tests were written against.
//
//	func createGroup(ctx context.Context, repo *hostaccess.Repository, name string, desc *string, _ *string, hostIDs []hostaccess.KnownHostID) (hostaccess.HostGroupID, error) {
//		return repo.CreateHostGroup(ctx, hostaccess.HostGroupDraft{
//			Name:        name,
//			Description: desc,
//			HostIDs:     hostIDs,
//		})
//	}
//
// // updateGroup adapts repo.UpdateHostGroup (which takes a full HostGroup) to a
// // positional helper that mirrors the legacy "*WithMembers" / "*Metadata" shape.
//
//	func updateGroup(ctx context.Context, repo *hostaccess.Repository, id hostaccess.HostGroupID, name string, desc *string, icon *string, hostIDs []hostaccess.KnownHostID) error {
//		return repo.UpdateHostGroup(ctx, hostaccess.HostGroup{
//			ID:          id,
//			Name:        name,
//			Description: desc,
//			Icon:        icon,
//			HostIDs:     hostIDs,
//		})
//	}
//
// // insertUser inserts a raw user row and returns its ID.
// // For active users a user_host_settings row is also created (mirroring the observer behaviour).
// // Deleted users intentionally have no settings row, simulating post-observer cleanup.
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

func TestRepository_UpdateKnownHost_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, db := setupHostAccessRepo(t)
	ctx := context.Background()

	createdAtMockTime := time.Now().Add(-time.Hour).Truncate(time.Second) // truncate to second for easier equality checks

	hostID1, err := repo.CreateKnownHost(ctx, hostaccess.KnownHostDraft{FQDN: "example.com"})
	is.NoErr(err)

	// Set created_at to a manual value so we can ensure updated_at changes but created at doesn't
	sq := `UPDATE known_hosts set created_at = ?, updated_at = ? where id = ?`
	if _, err := db.ExecContext(ctx, sq, createdAtMockTime, createdAtMockTime, hostID1); err != nil {
		t.Fatalf("update host(%q) created at: %v", hostID1, err)
	}

	updated, err := repo.UpdateKnownHost(ctx, hostID1, new("🌐"))
	is.NoErr(err)
	is.Equal(updated.ID.Int64(), int64(1))
	is.Equal(updated.FQDN, "example.com")
	is.Equal(*updated.Icon, "🌐")
	is.True(updated.UpdatedAt.After(updated.CreatedAt))
	is.Equal(updated.CreatedAt, createdAtMockTime)
}

func TestRepository_DeleteKnownHost_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)
	ctx := context.Background()

	hostID1, err := repo.CreateKnownHost(ctx, hostaccess.KnownHostDraft{FQDN: "example.com"})
	is.NoErr(err)

	err = repo.DeleteKnownHost(ctx, hostID1)
	is.NoErr(err)

	// Verify it's gone by trying to update it.
	_, err = repo.UpdateKnownHost(ctx, hostID1, nil)
	is.True(errors.Is(err, hostaccess.ErrKnownHostNotFound))
}

func TestRepository_CreateHostGroup_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)
	ctx := context.Background()

	hostID1, err := repo.CreateKnownHost(ctx, hostaccess.KnownHostDraft{FQDN: "a.com"})
	hostID2, err := repo.CreateKnownHost(ctx, hostaccess.KnownHostDraft{FQDN: "b.com"})
	is.NoErr(err)

	desc := "test group"
	groupID, err := repo.CreateHostGroup(ctx, hostaccess.HostGroupDraft{Name: "mygroup", Description: new(desc), HostIDs: []hostaccess.KnownHostID{hostID1, hostID2}})
	is.NoErr(err)
	is.True(groupID > 0)
}

func TestRepository_CreateHostGroup_EmptyMembers(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)
	ctx := context.Background()

	groupID, err := repo.CreateHostGroup(ctx, hostaccess.HostGroupDraft{Name: "empty-group", HostIDs: []hostaccess.KnownHostID{}})
	is.NoErr(err)
	is.True(groupID > 0)
}

func TestRepository_UpdateHostGroup_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)
	ctx := context.Background()

	hostID1, err := repo.CreateKnownHost(ctx, hostaccess.KnownHostDraft{FQDN: "a.com"})
	hostID2, err := repo.CreateKnownHost(ctx, hostaccess.KnownHostDraft{FQDN: "b.com"})
	is.NoErr(err)

	groupID, err := repo.CreateHostGroup(ctx, hostaccess.HostGroupDraft{Name: "original", HostIDs: []hostaccess.KnownHostID{1}, Icon: new("an icon")})
	is.NoErr(err)

	desc := "updated"

	updatedGroup := hostaccess.HostGroup{ID: groupID, Description: new(desc), HostIDs: []hostaccess.KnownHostID{hostID1, hostID2}}
	err = repo.UpdateHostGroup(ctx, updatedGroup)
	is.NoErr(err)
	is.Equal(*updatedGroup.Description, desc)
	is.Equal(updatedGroup.HostIDs, []hostaccess.KnownHostID{1, 2})
}

// ── DeleteHostGroup ──────────────────────────────────────────────────────────
func TestRepository_DeleteHostGroup_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)
	ctx := context.Background()

	groupID, err := repo.CreateHostGroup(ctx, hostaccess.HostGroupDraft{Name: "to-delete", HostIDs: []hostaccess.KnownHostID{}})
	is.NoErr(err)

	err = repo.DeleteHostGroup(ctx, groupID)
	is.NoErr(err)

	// Verify it's gone
	groups, err := repo.ListHostGroups(ctx)
	is.NoErr(err)
	is.Equal(len(groups), 0)
}

// ── SetFullUserGrants ────────────────────────────────────────────────────────
func TestRepository_SetFullUserGrants_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, db := setupHostAccessRepo(t)
	ctx := context.Background()

	userID := insertUser(t, db, "alice", false, false)
	hostId1, err := repo.CreateKnownHost(ctx, hostaccess.KnownHostDraft{FQDN: "a.com"})
	hostId2, err := repo.CreateKnownHost(ctx, hostaccess.KnownHostDraft{FQDN: "b.com"})
	is.NoErr(err)

	groupID, err := repo.CreateHostGroup(ctx, hostaccess.HostGroupDraft{Name: "g", HostIDs: []hostaccess.KnownHostID{hostId1}})
	is.NoErr(err)

	err = repo.SetFullUserGrants(ctx, userID, new(true),
		[]hostaccess.KnownHostID{hostId2},
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
	is.Equal(len(settings), 1)
	is.True(settings[0].BypassAllowlist)
}

func TestRepository_SetFullUserGrants_ReplacesExistingGrants(t *testing.T) {
	is := is.New(t)
	repo, db := setupHostAccessRepo(t)
	ctx := context.Background()

	userID := insertUser(t, db, "alice", false, false)
	hostID1, err := repo.CreateKnownHost(ctx, hostaccess.KnownHostDraft{FQDN: "a.com"})
	hostID2, err := repo.CreateKnownHost(ctx, hostaccess.KnownHostDraft{FQDN: "b.com"})
	is.NoErr(err)

	// First grant: a.com
	is.NoErr(repo.SetFullUserGrants(ctx, userID, nil, []hostaccess.KnownHostID{hostID1}, nil))

	// Second grant: b.com (should replace, not append)
	is.NoErr(repo.SetFullUserGrants(ctx, userID, nil, []hostaccess.KnownHostID{hostID2}, nil))

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
	settings, err := repo.GetAllUserHostSettings(ctx)
	is.Equal(len(settings), 0)
	is.NoErr(err)

	err = repo.EnsureUserSettings(ctx, userID)
	is.NoErr(err)

	// Should now appear in settings.
	settings, err = repo.GetAllUserHostSettings(ctx)
	is.NoErr(err)
	is.Equal(len(settings), 1)

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
	hostID1, err := repo.CreateKnownHost(ctx, hostaccess.KnownHostDraft{FQDN: "a.com"})
	is.NoErr(err)

	groupID, err := repo.CreateHostGroup(ctx, hostaccess.HostGroupDraft{Name: "g", HostIDs: []hostaccess.KnownHostID{hostID1}})
	is.NoErr(err)

	is.NoErr(repo.SetFullUserGrants(ctx, userID, nil, []hostaccess.KnownHostID{hostID1}, []hostaccess.HostGroupID{groupID}))

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

	hostID1, err := repo.CreateKnownHost(ctx, hostaccess.KnownHostDraft{FQDN: "example.com"})
	hostID2, err := repo.CreateKnownHost(ctx, hostaccess.KnownHostDraft{FQDN: "test.com"})
	is.NoErr(err)

	is.NoErr(repo.SetFullUserGrants(ctx, userID, nil, []hostaccess.KnownHostID{hostID1, hostID2}, nil))

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

	hostID1, err := repo.CreateKnownHost(ctx, hostaccess.KnownHostDraft{FQDN: "a.com"})
	hostID2, err := repo.CreateKnownHost(ctx, hostaccess.KnownHostDraft{FQDN: "b.com"})
	hostID3, err := repo.CreateKnownHost(ctx, hostaccess.KnownHostDraft{FQDN: "c.com"})
	is.NoErr(err)

	is.NoErr(repo.SetFullUserGrants(ctx, userA, nil, []hostaccess.KnownHostID{hostID1, hostID2}, nil))
	is.NoErr(repo.SetFullUserGrants(ctx, userB, nil, []hostaccess.KnownHostID{hostID3}, nil))

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
	hostID1, err := repo.CreateKnownHost(ctx, hostaccess.KnownHostDraft{FQDN: "example.com"})
	is.NoErr(err)
	is.NoErr(repo.SetFullUserGrants(ctx, userID, nil, []hostaccess.KnownHostID{hostID1}, nil))

	// Remove user
	err = repo.DeleteUserData(ctx, userID)
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
	hostID1, err := repo.CreateKnownHost(ctx, hostaccess.KnownHostDraft{FQDN: "group-host.com"})
	is.NoErr(err)

	groupID, err := repo.CreateHostGroup(ctx, hostaccess.HostGroupDraft{Name: "mygroup", HostIDs: []hostaccess.KnownHostID{hostID1}})
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

	hostID1, err := repo.CreateKnownHost(ctx, hostaccess.KnownHostDraft{FQDN: "g1.com"})
	hostID2, err := repo.CreateKnownHost(ctx, hostaccess.KnownHostDraft{FQDN: "g2.com"})
	is.NoErr(err)

	groupID1, err := repo.CreateHostGroup(ctx, hostaccess.HostGroupDraft{Name: "group1", HostIDs: []hostaccess.KnownHostID{hostID1}})
	is.NoErr(err)
	groupID2, err := repo.CreateHostGroup(ctx, hostaccess.HostGroupDraft{Name: "group2", HostIDs: []hostaccess.KnownHostID{hostID1, hostID2}})
	is.NoErr(err)

	is.NoErr(repo.SetFullUserGrants(ctx, userA, nil, nil, []hostaccess.HostGroupID{groupID1}))
	is.NoErr(repo.SetFullUserGrants(ctx, userB, nil, nil, []hostaccess.HostGroupID{groupID2}))

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
	hostID1, err := repo.CreateKnownHost(ctx, hostaccess.KnownHostDraft{FQDN: "group-host.com"})
	is.NoErr(err)

	groupID, err := repo.CreateHostGroup(ctx, hostaccess.HostGroupDraft{Name: "mygroup", HostIDs: []hostaccess.KnownHostID{hostID1}})
	is.NoErr(err)
	is.NoErr(repo.SetFullUserGrants(ctx, userID, nil, nil, []hostaccess.HostGroupID{groupID}))

	// Delete user
	err = repo.DeleteUserData(ctx, userID)
	is.NoErr(err)

	result, err := repo.GetAllUserGroupHostGrants(ctx)
	is.NoErr(err)
	is.Equal(len(result), 0)
}

// ── ListKnownHosts ────────────────────────────────────────────────────────────

func TestRepository_ListKnownHosts_Empty(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)

	hosts, err := repo.ListKnownHosts(context.Background())
	is.NoErr(err)
	is.Equal(len(hosts), 0)
}

func TestRepository_ListKnownHosts_ReturnsDeterministicOrder(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)
	ctx := context.Background()

	_, err := repo.CreateKnownHost(ctx, hostaccess.KnownHostDraft{FQDN: "b.example.com"})
	_, err = repo.CreateKnownHost(ctx, hostaccess.KnownHostDraft{FQDN: "a.example.com"})
	is.NoErr(err)

	hosts, err := repo.ListKnownHosts(ctx)
	is.NoErr(err)
	is.Equal(len(hosts), 2)
	// ORDER BY id guarantees ascending insertion order.
	is.True(hosts[0].ID < hosts[1].ID)
	is.Equal(hosts[0].FQDN, "b.example.com")
	is.Equal(hosts[1].FQDN, "a.example.com")
}

// ── CreateKnownHost ───────────────────────────────────────────────────────────

func TestRepository_CreateKnownHost_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)
	ctx := context.Background()

	icon := "server"
	_, err := repo.CreateKnownHost(ctx, hostaccess.KnownHostDraft{FQDN: "new.example.com", Icon: new(icon)})
	is.NoErr(err)

	hosts, err := repo.ListKnownHosts(ctx)
	is.NoErr(err)
	is.Equal(len(hosts), 1)
	is.Equal(hosts[0].FQDN, "new.example.com")
	is.Equal(*hosts[0].Icon, "server")
}

func TestRepository_CreateKnownHost_UniqueViolation_ErrKnownHostConflict(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)
	ctx := context.Background()

	_, err := repo.CreateKnownHost(ctx, hostaccess.KnownHostDraft{FQDN: "dup.example.com"})
	is.NoErr(err)

	_, err = repo.CreateKnownHost(ctx, hostaccess.KnownHostDraft{FQDN: "dup.example.com"})
	is.True(errors.Is(err, hostaccess.ErrKnownHostConflict))
}

// ── FK cascade on delete ──────────────────────────────────────────────────────

func TestRepository_DeleteKnownHost_CascadesToGroupMembersAndUserGrants(t *testing.T) {
	is := is.New(t)
	repo, db := setupHostAccessRepo(t)
	ctx := context.Background()

	hostID1, err := repo.CreateKnownHost(ctx, hostaccess.KnownHostDraft{FQDN: "cascade.example.com"})
	is.NoErr(err)

	groupID, err := repo.CreateHostGroup(ctx, hostaccess.HostGroupDraft{Name: "g", HostIDs: []hostaccess.KnownHostID{hostID1}})
	is.NoErr(err)

	userID := insertUser(t, db, "alice", false, false)
	is.NoErr(repo.SetFullUserGrants(ctx, userID, nil, []hostaccess.KnownHostID{hostID1}, nil))

	is.NoErr(repo.DeleteKnownHost(ctx, hostID1))

	// Verify group members cascaded.
	var memberCount int
	is.NoErr(db.QueryRowxContext(ctx,
		`SELECT COUNT(*) FROM host_group_members WHERE host_group_id = ?`, groupID,
	).Scan(&memberCount))
	is.Equal(memberCount, 0)

	// Verify user grants cascaded.
	var grantCount int
	is.NoErr(db.QueryRowxContext(ctx,
		`SELECT COUNT(*) FROM user_allowed_hosts WHERE user_id = ?`, userID,
	).Scan(&grantCount))
	is.Equal(grantCount, 0)
}
