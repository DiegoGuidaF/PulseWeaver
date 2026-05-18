//go:build test

package hostaccess_test

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/hostaccess"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testdb"
	"github.com/matryer/is"
)

func setupHostAccessRepo(t *testing.T) (*hostaccess.Repository, *database.DB) {
	t.Helper()
	db, cleanup := testdb.Setup(t)
	t.Cleanup(cleanup)
	return hostaccess.NewRepository(db.DB()), db.DB()
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

// ── DeleteHost ────────────────────────────────────────────────────────────────

func TestRepository_DeleteHost_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)
	ctx := context.Background()

	hostID, err := repo.CreateHost(ctx, hostaccess.HostDraft{FQDN: "example.com"})
	is.NoErr(err)

	err = repo.DeleteHost(ctx, hostID)
	is.NoErr(err)

	hosts, err := repo.ListHosts(ctx)
	is.NoErr(err)
	is.Equal(len(hosts), 0)
}

func TestRepository_DeleteHost_NotFound(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)

	err := repo.DeleteHost(context.Background(), ids.HostID(999))
	is.True(errors.Is(err, hostaccess.ErrHostNotFound))
}

// ── CreateHostGroup ───────────────────────────────────────────────────────────

func TestRepository_CreateHostGroup_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)
	ctx := context.Background()

	hostID1, err := repo.CreateHost(ctx, hostaccess.HostDraft{FQDN: "a.com"})
	is.NoErr(err)
	hostID2, err := repo.CreateHost(ctx, hostaccess.HostDraft{FQDN: "b.com"})
	is.NoErr(err)

	desc := "test group"
	groupID, err := repo.CreateHostGroup(ctx, hostaccess.HostGroupDraft{Name: "mygroup", Description: new(desc), HostIDs: []ids.HostID{hostID1, hostID2}})
	is.NoErr(err)
	is.True(groupID > 0)
}

func TestRepository_CreateHostGroup_EmptyMembers(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)
	ctx := context.Background()

	groupID, err := repo.CreateHostGroup(ctx, hostaccess.HostGroupDraft{Name: "empty-group", HostIDs: []ids.HostID{}})
	is.NoErr(err)
	is.True(groupID > 0)
}

// ── UpdateHostGroup ───────────────────────────────────────────────────────────

func TestRepository_UpdateHostGroup_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)
	ctx := context.Background()

	hostID1, err := repo.CreateHost(ctx, hostaccess.HostDraft{FQDN: "a.com"})
	is.NoErr(err)
	hostID2, err := repo.CreateHost(ctx, hostaccess.HostDraft{FQDN: "b.com"})
	is.NoErr(err)

	groupID, err := repo.CreateHostGroup(ctx, hostaccess.HostGroupDraft{Name: "original", HostIDs: []ids.HostID{hostID1}, Icon: "an icon"})
	is.NoErr(err)

	desc := "updated"
	updatedGroup := hostaccess.HostGroup{ID: groupID, Description: new(desc), HostIDs: []ids.HostID{hostID1, hostID2}}
	err = repo.UpdateHostGroup(ctx, updatedGroup)
	is.NoErr(err)
	is.Equal(*updatedGroup.Description, desc)
	is.Equal(updatedGroup.HostIDs, []ids.HostID{hostID1, hostID2})
}

// ── DeleteHostGroup ───────────────────────────────────────────────────────────

func TestRepository_DeleteHostGroup_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)
	ctx := context.Background()

	groupID, err := repo.CreateHostGroup(ctx, hostaccess.HostGroupDraft{Name: "to-delete", HostIDs: []ids.HostID{}})
	is.NoErr(err)

	err = repo.DeleteHostGroup(ctx, groupID)
	is.NoErr(err)

	groups, err := repo.ListHostGroups(ctx)
	is.NoErr(err)
	is.Equal(len(groups), 0)
}

// ── SetHostGroupMembership ────────────────────────────────────────────────────

func TestRepository_SetHostGroupMembership_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)
	ctx := context.Background()

	hostID, err := repo.CreateHost(ctx, hostaccess.HostDraft{FQDN: "host.example.com"})
	is.NoErr(err)
	groupID1, err := repo.CreateHostGroup(ctx, hostaccess.HostGroupDraft{Name: "g1", HostIDs: []ids.HostID{}})
	is.NoErr(err)
	groupID2, err := repo.CreateHostGroup(ctx, hostaccess.HostGroupDraft{Name: "g2", HostIDs: []ids.HostID{}})
	is.NoErr(err)

	err = repo.SetHostGroupMembership(ctx, hostID, []ids.HostGroupID{groupID1, groupID2})
	is.NoErr(err)

	groups, err := repo.ListHostGroups(ctx)
	is.NoErr(err)
	hostsByGroup := make(map[ids.HostGroupID][]ids.HostID)
	for _, g := range groups {
		hostsByGroup[g.ID] = g.HostIDs
	}
	is.Equal(hostsByGroup[groupID1], []ids.HostID{hostID})
	is.Equal(hostsByGroup[groupID2], []ids.HostID{hostID})

	// Replacing with a subset clears the old membership.
	err = repo.SetHostGroupMembership(ctx, hostID, []ids.HostGroupID{groupID1})
	is.NoErr(err)

	groups, err = repo.ListHostGroups(ctx)
	is.NoErr(err)
	for _, g := range groups {
		hostsByGroup[g.ID] = g.HostIDs
	}
	is.Equal(hostsByGroup[groupID1], []ids.HostID{hostID})
	is.Equal(len(hostsByGroup[groupID2]), 0)
}

func TestRepository_SetHostGroupMembership_UnknownGroupID_ReturnsReferenceNotFound(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)
	ctx := context.Background()

	hostID, err := repo.CreateHost(ctx, hostaccess.HostDraft{FQDN: "host.example.com"})
	is.NoErr(err)

	err = repo.SetHostGroupMembership(ctx, hostID, []ids.HostGroupID{9999})
	is.True(errors.Is(err, hostaccess.ErrReferenceNotFound))
}

// ── SetUserAccess ─────────────────────────────────────────────────────────────

func TestRepository_SetUserAccess_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, db := setupHostAccessRepo(t)
	ctx := context.Background()

	userID := insertUser(t, db, "alice", false, false)
	hostID1, err := repo.CreateHost(ctx, hostaccess.HostDraft{FQDN: "a.com"})
	is.NoErr(err)

	groupID, err := repo.CreateHostGroup(ctx, hostaccess.HostGroupDraft{Name: "g", HostIDs: []ids.HostID{hostID1}})
	is.NoErr(err)

	err = repo.SetUserAccess(ctx, userID, true, []ids.HostGroupID{groupID})
	is.NoErr(err)

	group, err := repo.GetAllUserHostGrants(ctx)
	is.NoErr(err)
	is.Equal(len(group), 1)
	is.Equal(group[0].FQDN, "a.com")

	settings, err := repo.GetAllUserHostSettings(ctx)
	is.NoErr(err)
	is.Equal(len(settings), 1)
	is.True(settings[0].BypassHostCheck)
}

func TestRepository_SetUserAccess_ReplacesExistingAccess(t *testing.T) {
	is := is.New(t)
	repo, db := setupHostAccessRepo(t)
	ctx := context.Background()

	userID := insertUser(t, db, "alice", false, false)

	hostID1, err := repo.CreateHost(ctx, hostaccess.HostDraft{FQDN: "a.com"})
	is.NoErr(err)
	hostID2, err := repo.CreateHost(ctx, hostaccess.HostDraft{FQDN: "b.com"})
	is.NoErr(err)

	hostGroupID1, err := repo.CreateHostGroup(ctx, hostaccess.HostGroupDraft{Name: "g1", HostIDs: []ids.HostID{hostID1}})
	is.NoErr(err)
	hostGroupID2, err := repo.CreateHostGroup(ctx, hostaccess.HostGroupDraft{Name: "g2", HostIDs: []ids.HostID{hostID2}})
	is.NoErr(err)

	// First grant: group g1 (a.com)
	is.NoErr(repo.SetUserAccess(ctx, userID, false, []ids.HostGroupID{hostGroupID1}))

	// Second grant: group g2 (b.com) — should replace, not append.
	is.NoErr(repo.SetUserAccess(ctx, userID, false, []ids.HostGroupID{hostGroupID2}))

	grants, err := repo.GetAllUserHostGrants(ctx)
	is.NoErr(err)
	is.Equal(len(grants), 1)
	is.Equal(grants[0].FQDN, "b.com")
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

	err = repo.RemoveIgnoredSuggestionByFQDN(ctx, "spam.example.com")
	is.True(errors.Is(err, hostaccess.ErrSuggestionNotFound))
}

// ── EnsureUserSettings ────────────────────────────────────────────────────────

func TestRepository_EnsureUserSettings_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, db := setupHostAccessRepo(t)
	ctx := context.Background()

	userID := insertUser(t, db, "no-settings", false, true)
	settings, err := repo.GetAllUserHostSettings(ctx)
	is.Equal(len(settings), 0)
	is.NoErr(err)

	err = repo.EnsureUserSettings(ctx, userID)
	is.NoErr(err)

	settings, err = repo.GetAllUserHostSettings(ctx)
	is.NoErr(err)
	is.Equal(len(settings), 1)

	found := false
	for _, s := range settings {
		if s.UserID == userID {
			is.Equal(s.BypassHostCheck, false) // default is 0
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

	err := repo.EnsureUserSettings(ctx, userID)
	is.NoErr(err)

	settings, err := repo.GetAllUserHostSettings(ctx)
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
	repo, db := setupHostAccessRepo(t)
	ctx := context.Background()

	userID := insertUser(t, db, "alice", false, false)
	hostID1, err := repo.CreateHost(ctx, hostaccess.HostDraft{FQDN: "a.com"})
	is.NoErr(err)

	groupID, err := repo.CreateHostGroup(ctx, hostaccess.HostGroupDraft{Name: "g", HostIDs: []ids.HostID{hostID1}})
	is.NoErr(err)

	is.NoErr(repo.SetUserAccess(ctx, userID, false, []ids.HostGroupID{groupID}))

	err = repo.DeleteUserData(ctx, userID)
	is.NoErr(err)

	settings, err := repo.GetAllUserHostSettings(ctx)
	is.NoErr(err)
	is.Equal(len(settings), 0)

	group, err := repo.GetAllUserHostGrants(ctx)
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

	byUser := make(map[ids.UserID]bool)
	for _, s := range result {
		byUser[s.UserID] = s.BypassHostCheck
	}
	is.Equal(byUser[userA], false)
	is.Equal(byUser[userB], true)
}

// ── GetAllUserHostGrants ─────────────────────────────────────────────────

func TestRepository_GetAllUserHostGrants_Empty(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)

	result, err := repo.GetAllUserHostGrants(context.Background())
	is.NoErr(err)
	is.Equal(len(result), 0)
}

func TestRepository_GetAllUserHostGrants_ReturnsGrants(t *testing.T) {
	is := is.New(t)
	repo, db := setupHostAccessRepo(t)
	ctx := context.Background()

	userID := insertUser(t, db, "bob", false, false)
	hostID1, err := repo.CreateHost(ctx, hostaccess.HostDraft{FQDN: "group-host.com"})
	is.NoErr(err)

	groupID, err := repo.CreateHostGroup(ctx, hostaccess.HostGroupDraft{Name: "mygroup", HostIDs: []ids.HostID{hostID1}})
	is.NoErr(err)
	is.NoErr(repo.SetUserAccess(ctx, userID, false, []ids.HostGroupID{groupID}))

	result, err := repo.GetAllUserHostGrants(ctx)
	is.NoErr(err)
	is.Equal(len(result), 1)
	is.Equal(result[0].UserID, userID)
	is.Equal(result[0].FQDN, "group-host.com")
}

func TestRepository_GetAllUserHostGrants_MultipleUsers(t *testing.T) {
	is := is.New(t)
	repo, db := setupHostAccessRepo(t)
	ctx := context.Background()

	userA := insertUser(t, db, "alice", false, false)
	userB := insertUser(t, db, "bob", false, false)

	hostID1, err := repo.CreateHost(ctx, hostaccess.HostDraft{FQDN: "g1.com"})
	is.NoErr(err)
	hostID2, err := repo.CreateHost(ctx, hostaccess.HostDraft{FQDN: "g2.com"})
	is.NoErr(err)

	groupID1, err := repo.CreateHostGroup(ctx, hostaccess.HostGroupDraft{Name: "group1", HostIDs: []ids.HostID{hostID1}})
	is.NoErr(err)
	groupID2, err := repo.CreateHostGroup(ctx, hostaccess.HostGroupDraft{Name: "group2", HostIDs: []ids.HostID{hostID1, hostID2}})
	is.NoErr(err)

	is.NoErr(repo.SetUserAccess(ctx, userA, false, []ids.HostGroupID{groupID1}))
	is.NoErr(repo.SetUserAccess(ctx, userB, false, []ids.HostGroupID{groupID2}))

	result, err := repo.GetAllUserHostGrants(ctx)
	is.NoErr(err)

	byUser := make(map[ids.UserID][]string)
	for _, g := range result {
		byUser[g.UserID] = append(byUser[g.UserID], g.FQDN)
	}
	is.Equal(byUser[userA], []string{"g1.com"})
	sort.Strings(byUser[userB])
	is.Equal(byUser[userB], []string{"g1.com", "g2.com"})
}

// ── ListHosts ─────────────────────────────────────────────────────────────────

func TestRepository_ListHosts_Empty(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)

	hosts, err := repo.ListHosts(context.Background())
	is.NoErr(err)
	is.Equal(len(hosts), 0)
}

func TestRepository_ListHosts_ReturnsDeterministicOrder(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)
	ctx := context.Background()

	_, err := repo.CreateHost(ctx, hostaccess.HostDraft{FQDN: "b.example.com"})
	is.NoErr(err)
	_, err = repo.CreateHost(ctx, hostaccess.HostDraft{FQDN: "a.example.com"})
	is.NoErr(err)

	hosts, err := repo.ListHosts(ctx)
	is.NoErr(err)
	is.Equal(len(hosts), 2)
	// ORDER BY id guarantees ascending insertion order.
	is.True(hosts[0].ID < hosts[1].ID)
	is.Equal(hosts[0].FQDN, "b.example.com")
	is.Equal(hosts[1].FQDN, "a.example.com")
}

// ── CreateHost ────────────────────────────────────────────────────────────────

func TestRepository_CreateHost_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)
	ctx := context.Background()

	_, err := repo.CreateHost(ctx, hostaccess.HostDraft{FQDN: "new.example.com"})
	is.NoErr(err)

	hosts, err := repo.ListHosts(ctx)
	is.NoErr(err)
	is.Equal(len(hosts), 1)
	is.Equal(hosts[0].FQDN, "new.example.com")
}

func TestRepository_CreateHost_UniqueViolation_ErrHostConflict(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostAccessRepo(t)
	ctx := context.Background()

	_, err := repo.CreateHost(ctx, hostaccess.HostDraft{FQDN: "dup.example.com"})
	is.NoErr(err)

	_, err = repo.CreateHost(ctx, hostaccess.HostDraft{FQDN: "dup.example.com"})
	is.True(errors.Is(err, hostaccess.ErrHostConflict))
}

// ── FK cascade on delete ──────────────────────────────────────────────────────

func TestRepository_DeleteHost_CascadesToGroupMembers(t *testing.T) {
	is := is.New(t)
	repo, db := setupHostAccessRepo(t)
	ctx := context.Background()

	hostID1, err := repo.CreateHost(ctx, hostaccess.HostDraft{FQDN: "cascade.example.com"})
	is.NoErr(err)

	groupID, err := repo.CreateHostGroup(ctx, hostaccess.HostGroupDraft{Name: "g", HostIDs: []ids.HostID{hostID1}})
	is.NoErr(err)

	is.NoErr(repo.DeleteHost(ctx, hostID1))

	// Verify group members cascaded.
	var memberCount int
	is.NoErr(db.QueryRowxContext(ctx,
		`SELECT COUNT(*) FROM host_group_members WHERE host_group_id = ?`, groupID,
	).Scan(&memberCount))
	is.Equal(memberCount, 0)
}
