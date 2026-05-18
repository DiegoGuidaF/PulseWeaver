//go:build test

package hosts_test

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/hosts"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testdb"
	"github.com/matryer/is"
)

func setupHostsRepo(t *testing.T) (*hosts.Repository, *database.DB) {
	t.Helper()
	db, cleanup := testdb.Setup(t)
	t.Cleanup(cleanup)
	return hosts.NewRepository(db.DB()), db.DB()
}

// ── DeleteHost ────────────────────────────────────────────────────────────────

func TestRepository_DeleteHost_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostsRepo(t)
	ctx := context.Background()

	hostID, err := repo.CreateHost(ctx, hosts.HostDraft{FQDN: "example.com"})
	is.NoErr(err)

	err = repo.DeleteHost(ctx, hostID)
	is.NoErr(err)

	hs, err := repo.ListHosts(ctx)
	is.NoErr(err)
	is.Equal(len(hs), 0)
}

func TestRepository_DeleteHost_NotFound(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostsRepo(t)

	err := repo.DeleteHost(context.Background(), ids.HostID(999))
	is.True(errors.Is(err, hosts.ErrHostNotFound))
}

// ── CreateHostGroup ───────────────────────────────────────────────────────────

func TestRepository_CreateHostGroup_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostsRepo(t)
	ctx := context.Background()

	hostID1, err := repo.CreateHost(ctx, hosts.HostDraft{FQDN: "a.com"})
	is.NoErr(err)
	hostID2, err := repo.CreateHost(ctx, hosts.HostDraft{FQDN: "b.com"})
	is.NoErr(err)

	desc := "test group"
	groupID, err := repo.CreateHostGroup(ctx, hosts.HostGroupDraft{Name: "mygroup", Description: new(desc), HostIDs: []ids.HostID{hostID1, hostID2}})
	is.NoErr(err)
	is.True(groupID > 0)
}

func TestRepository_CreateHostGroup_EmptyMembers(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostsRepo(t)
	ctx := context.Background()

	groupID, err := repo.CreateHostGroup(ctx, hosts.HostGroupDraft{Name: "empty-group", HostIDs: []ids.HostID{}})
	is.NoErr(err)
	is.True(groupID > 0)
}

// ── UpdateHostGroup ───────────────────────────────────────────────────────────

func TestRepository_UpdateHostGroup_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostsRepo(t)
	ctx := context.Background()

	hostID1, err := repo.CreateHost(ctx, hosts.HostDraft{FQDN: "a.com"})
	is.NoErr(err)
	hostID2, err := repo.CreateHost(ctx, hosts.HostDraft{FQDN: "b.com"})
	is.NoErr(err)

	groupID, err := repo.CreateHostGroup(ctx, hosts.HostGroupDraft{Name: "original", HostIDs: []ids.HostID{hostID1}, Icon: "an icon"})
	is.NoErr(err)

	desc := "updated"
	updatedGroup := hosts.HostGroup{ID: groupID, Description: new(desc), HostIDs: []ids.HostID{hostID1, hostID2}}
	err = repo.UpdateHostGroup(ctx, updatedGroup)
	is.NoErr(err)
	is.Equal(*updatedGroup.Description, desc)
	is.Equal(updatedGroup.HostIDs, []ids.HostID{hostID1, hostID2})
}

// ── DeleteHostGroup ───────────────────────────────────────────────────────────

func TestRepository_DeleteHostGroup_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostsRepo(t)
	ctx := context.Background()

	groupID, err := repo.CreateHostGroup(ctx, hosts.HostGroupDraft{Name: "to-delete", HostIDs: []ids.HostID{}})
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
	repo, _ := setupHostsRepo(t)
	ctx := context.Background()

	hostID, err := repo.CreateHost(ctx, hosts.HostDraft{FQDN: "host.example.com"})
	is.NoErr(err)
	groupID1, err := repo.CreateHostGroup(ctx, hosts.HostGroupDraft{Name: "g1", HostIDs: []ids.HostID{}})
	is.NoErr(err)
	groupID2, err := repo.CreateHostGroup(ctx, hosts.HostGroupDraft{Name: "g2", HostIDs: []ids.HostID{}})
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
	repo, _ := setupHostsRepo(t)
	ctx := context.Background()

	hostID, err := repo.CreateHost(ctx, hosts.HostDraft{FQDN: "host.example.com"})
	is.NoErr(err)

	err = repo.SetHostGroupMembership(ctx, hostID, []ids.HostGroupID{9999})
	is.True(errors.Is(err, hosts.ErrReferenceNotFound))
}

// ── AddIgnoredSuggestion ─────────────────────────────────────────────────────

func TestRepository_AddIgnoredSuggestion_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostsRepo(t)

	s, err := repo.AddIgnoredSuggestion(context.Background(), "Spam.Example.COM")
	is.NoErr(err)
	is.True(s.ID > 0)
	is.Equal(s.FQDN, "spam.example.com")
	is.True(!s.CreatedAt.IsZero())
}

// ── RemoveIgnoredSuggestionByFQDN ────────────────────────────────────────────

func TestRepository_RemoveIgnoredSuggestionByFQDN_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostsRepo(t)
	ctx := context.Background()

	_, err := repo.AddIgnoredSuggestion(ctx, "spam.example.com")
	is.NoErr(err)

	err = repo.RemoveIgnoredSuggestionByFQDN(ctx, "spam.example.com")
	is.NoErr(err)

	err = repo.RemoveIgnoredSuggestionByFQDN(ctx, "spam.example.com")
	is.True(errors.Is(err, hosts.ErrSuggestionNotFound))
}

// ── ListHosts ─────────────────────────────────────────────────────────────────

func TestRepository_ListHosts_Empty(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostsRepo(t)

	hs, err := repo.ListHosts(context.Background())
	is.NoErr(err)
	is.Equal(len(hs), 0)
}

func TestRepository_ListHosts_ReturnsDeterministicOrder(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostsRepo(t)
	ctx := context.Background()

	_, err := repo.CreateHost(ctx, hosts.HostDraft{FQDN: "b.example.com"})
	is.NoErr(err)
	_, err = repo.CreateHost(ctx, hosts.HostDraft{FQDN: "a.example.com"})
	is.NoErr(err)

	hs, err := repo.ListHosts(ctx)
	is.NoErr(err)
	is.Equal(len(hs), 2)
	is.True(hs[0].ID < hs[1].ID)
	is.Equal(hs[0].FQDN, "b.example.com")
	is.Equal(hs[1].FQDN, "a.example.com")
}

// ── CreateHost ────────────────────────────────────────────────────────────────

func TestRepository_CreateHost_HappyPath(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostsRepo(t)
	ctx := context.Background()

	_, err := repo.CreateHost(ctx, hosts.HostDraft{FQDN: "new.example.com"})
	is.NoErr(err)

	hs, err := repo.ListHosts(ctx)
	is.NoErr(err)
	is.Equal(len(hs), 1)
	is.Equal(hs[0].FQDN, "new.example.com")
}

func TestRepository_CreateHost_UniqueViolation_ErrHostConflict(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostsRepo(t)
	ctx := context.Background()

	_, err := repo.CreateHost(ctx, hosts.HostDraft{FQDN: "dup.example.com"})
	is.NoErr(err)

	_, err = repo.CreateHost(ctx, hosts.HostDraft{FQDN: "dup.example.com"})
	is.True(errors.Is(err, hosts.ErrHostConflict))
}

// ── FK cascade on delete ──────────────────────────────────────────────────────

func TestRepository_DeleteHost_CascadesToGroupMembers(t *testing.T) {
	is := is.New(t)
	repo, db := setupHostsRepo(t)
	ctx := context.Background()

	hostID1, err := repo.CreateHost(ctx, hosts.HostDraft{FQDN: "cascade.example.com"})
	is.NoErr(err)

	groupID, err := repo.CreateHostGroup(ctx, hosts.HostGroupDraft{Name: "g", HostIDs: []ids.HostID{hostID1}})
	is.NoErr(err)

	is.NoErr(repo.DeleteHost(ctx, hostID1))

	var memberCount int
	is.NoErr(db.QueryRowxContext(ctx,
		`SELECT COUNT(*) FROM host_group_members WHERE host_group_id = ?`, groupID,
	).Scan(&memberCount))
	is.Equal(memberCount, 0)
}

// ── ListHostGroups (order stability) ─────────────────────────────────────────

func TestRepository_ListHostGroups_ReturnsMembersPerGroup(t *testing.T) {
	is := is.New(t)
	repo, _ := setupHostsRepo(t)
	ctx := context.Background()

	h1, _ := repo.CreateHost(ctx, hosts.HostDraft{FQDN: "h1.com"})
	h2, _ := repo.CreateHost(ctx, hosts.HostDraft{FQDN: "h2.com"})
	g, _ := repo.CreateHostGroup(ctx, hosts.HostGroupDraft{Name: "grp", HostIDs: []ids.HostID{h1, h2}})

	groups, err := repo.ListHostGroups(ctx)
	is.NoErr(err)
	is.Equal(len(groups), 1)

	memberIDs := groups[0].HostIDs
	sort.Slice(memberIDs, func(i, j int) bool { return memberIDs[i] < memberIDs[j] })
	_ = g
	is.Equal(memberIDs, []ids.HostID{h1, h2})
}
