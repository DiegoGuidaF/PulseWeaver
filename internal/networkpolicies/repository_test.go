//go:build test

package networkpolicies_test

import (
	"context"
	"errors"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/hosts"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/networkpolicies"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testdb"
	"github.com/matryer/is"
)

type repoFixture struct {
	repo   *networkpolicies.Repository
	haRepo *hosts.Repository
}

func setupRepoTest(t *testing.T) repoFixture {
	t.Helper()
	sqlite, cleanup := testdb.Setup(t)
	t.Cleanup(cleanup)
	db := sqlite.DB()
	return repoFixture{
		repo:   networkpolicies.NewRepository(db),
		haRepo: hosts.NewRepository(db),
	}
}

func insertPolicy(t *testing.T, repo *networkpolicies.Repository, name, cidr string) networkpolicies.NetworkPolicy {
	t.Helper()
	p, err := repo.CreatePolicy(context.Background(), networkpolicies.NetworkPolicy{
		Name:    name,
		CIDR:    cidr,
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("insertPolicy %q: %v", name, err)
	}
	return p
}

func insertHostGroup(t *testing.T, repo *hosts.Repository, name string) ids.HostGroupID {
	t.Helper()
	id, err := repo.CreateHostGroup(context.Background(), hosts.HostGroupDraft{Name: name})
	if err != nil {
		t.Fatalf("insertHostGroup %q: %v", name, err)
	}
	return id
}

func insertHostInGroup(t *testing.T, repo *hosts.Repository, fqdn string, groupID ids.HostGroupID) ids.HostID {
	t.Helper()
	hostID, err := repo.CreateHost(context.Background(), hosts.HostDraft{FQDN: fqdn})
	if err != nil {
		t.Fatalf("insertHost %q: %v", fqdn, err)
	}
	if err := repo.SetHostGroupMembership(context.Background(), hostID, []ids.HostGroupID{groupID}); err != nil {
		t.Fatalf("SetHostGroupMembership: %v", err)
	}
	return hostID
}

// ── CreatePolicy ─────────────────────────────────────────────────────────────

func TestRepository_CreatePolicy_ReturnsPersistedPolicy(t *testing.T) {
	is := is.New(t)
	fix := setupRepoTest(t)
	ctx := context.Background()

	desc := "home network"
	p, err := fix.repo.CreatePolicy(ctx, networkpolicies.NetworkPolicy{
		Name:        "home",
		CIDR:        "192.168.1.0/24",
		Description: &desc,
		Enabled:     true,
	})

	is.NoErr(err)
	is.True(p.ID != 0)
	is.Equal(p.Name, "home")
	is.Equal(p.CIDR, "192.168.1.0/24")
	is.True(p.Description != nil)
	is.Equal(*p.Description, "home network")
	is.True(!p.CreatedAt.IsZero())
}

func TestRepository_CreatePolicy_DuplicateCIDR_ReturnsErrCIDRConflict(t *testing.T) {
	is := is.New(t)
	fix := setupRepoTest(t)
	ctx := context.Background()

	insertPolicy(t, fix.repo, "first", "10.0.0.0/8")

	_, err := fix.repo.CreatePolicy(ctx, networkpolicies.NetworkPolicy{Name: "second", CIDR: "10.0.0.0/8", Enabled: true})

	is.True(errors.Is(err, networkpolicies.ErrCIDRConflict))
}

// ── GetPolicy ─────────────────────────────────────────────────────────────────

func TestRepository_GetPolicy_ReturnsAllFields(t *testing.T) {
	is := is.New(t)
	fix := setupRepoTest(t)
	ctx := context.Background()

	created := insertPolicy(t, fix.repo, "vpn", "10.8.0.0/16")

	got, err := fix.repo.GetPolicy(ctx, created.ID)

	is.NoErr(err)
	is.Equal(got.ID, created.ID)
	is.Equal(got.Name, "vpn")
	is.Equal(got.CIDR, "10.8.0.0/16")
	is.True(!got.CreatedAt.IsZero())
}

func TestRepository_GetPolicy_NotFound_ReturnsErrNotFound(t *testing.T) {
	is := is.New(t)
	fix := setupRepoTest(t)

	_, err := fix.repo.GetPolicy(context.Background(), ids.NetworkPolicyID(99999))

	is.True(errors.Is(err, networkpolicies.ErrNotFound))
}

// ── UpdatePolicy ─────────────────────────────────────────────────────────────

func TestRepository_UpdatePolicy_PersistsChanges(t *testing.T) {
	is := is.New(t)
	fix := setupRepoTest(t)
	ctx := context.Background()

	p := insertPolicy(t, fix.repo, "original", "172.16.0.0/12")

	desc := "updated"
	p.Name = "renamed"
	p.Description = &desc
	p.Enabled = false

	updated, err := fix.repo.UpdatePolicy(ctx, p)

	is.NoErr(err)
	is.Equal(updated.Name, "renamed")
	is.Equal(updated.Enabled, false)
	is.True(updated.Description != nil)
	is.Equal(*updated.Description, "updated")
	// UpdatedAt must not be before CreatedAt
	is.True(!updated.UpdatedAt.Before(updated.CreatedAt))
}

func TestRepository_UpdatePolicy_ClearDescription(t *testing.T) {
	is := is.New(t)
	fix := setupRepoTest(t)
	ctx := context.Background()

	desc := "will be cleared"
	p, err := fix.repo.CreatePolicy(ctx, networkpolicies.NetworkPolicy{
		Name: "with-desc", CIDR: "10.1.0.0/16", Description: &desc, Enabled: true,
	})
	is.NoErr(err)

	p.Description = nil
	updated, err := fix.repo.UpdatePolicy(ctx, p)

	is.NoErr(err)
	is.True(updated.Description == nil)
}

func TestRepository_UpdatePolicy_DuplicateCIDR_ReturnsErrCIDRConflict(t *testing.T) {
	is := is.New(t)
	fix := setupRepoTest(t)
	ctx := context.Background()

	insertPolicy(t, fix.repo, "other", "10.2.0.0/16")
	p := insertPolicy(t, fix.repo, "target", "10.3.0.0/16")

	p.CIDR = "10.2.0.0/16"
	_, err := fix.repo.UpdatePolicy(ctx, p)

	is.True(errors.Is(err, networkpolicies.ErrCIDRConflict))
}

func TestRepository_UpdatePolicy_NotFound_ReturnsErrNotFound(t *testing.T) {
	is := is.New(t)
	fix := setupRepoTest(t)

	_, err := fix.repo.UpdatePolicy(context.Background(), networkpolicies.NetworkPolicy{
		ID: ids.NetworkPolicyID(99999), Name: "ghost", CIDR: "10.4.0.0/16",
	})

	is.True(errors.Is(err, networkpolicies.ErrNotFound))
}

// ── DeletePolicy ─────────────────────────────────────────────────────────────

func TestRepository_DeletePolicy_RemovesRow(t *testing.T) {
	is := is.New(t)
	fix := setupRepoTest(t)
	ctx := context.Background()

	p := insertPolicy(t, fix.repo, "to-delete", "10.5.0.0/16")

	err := fix.repo.DeletePolicy(ctx, p.ID)
	is.NoErr(err)

	_, err = fix.repo.GetPolicy(ctx, p.ID)
	is.True(errors.Is(err, networkpolicies.ErrNotFound))
}

func TestRepository_DeletePolicy_NotFound_ReturnsErrNotFound(t *testing.T) {
	is := is.New(t)
	fix := setupRepoTest(t)

	err := fix.repo.DeletePolicy(context.Background(), ids.NetworkPolicyID(99999))

	is.True(errors.Is(err, networkpolicies.ErrNotFound))
}

// ── SetHostAccess ─────────────────────────────────────────────────────────────

func TestRepository_SetHostAccess_ReplacesGroups(t *testing.T) {
	is := is.New(t)
	fix := setupRepoTest(t)
	ctx := context.Background()

	p := insertPolicy(t, fix.repo, "policy", "10.6.0.0/16")
	g1 := insertHostGroup(t, fix.haRepo, "group-a")
	g2 := insertHostGroup(t, fix.haRepo, "group-b")

	// Assign g1 then replace with g2
	err := fix.repo.SetHostAccess(ctx, p.ID, false, []ids.HostGroupID{g1})
	is.NoErr(err)

	err = fix.repo.SetHostAccess(ctx, p.ID, true, []ids.HostGroupID{g2})
	is.NoErr(err)

	// Verify via cache entries: only g2's hosts should appear and bypass_host_check updated
	entries, err := fix.repo.GetEnabledCacheEntries(ctx)
	is.NoErr(err)
	is.Equal(len(entries), 1)
	is.Equal(entries[0].BypassHostCheck, true)
}

func TestRepository_SetHostAccess_ClearsGroups_EmptySlice(t *testing.T) {
	is := is.New(t)
	fix := setupRepoTest(t)
	ctx := context.Background()

	p := insertPolicy(t, fix.repo, "policy", "10.7.0.0/16")
	g := insertHostGroup(t, fix.haRepo, "group-c")

	err := fix.repo.SetHostAccess(ctx, p.ID, false, []ids.HostGroupID{g})
	is.NoErr(err)

	err = fix.repo.SetHostAccess(ctx, p.ID, false, []ids.HostGroupID{})
	is.NoErr(err)

	// Verify via cache entries — no hosts should appear (empty AllowedHostFQDNs)
	entries, err := fix.repo.GetEnabledCacheEntries(ctx)
	is.NoErr(err)
	is.Equal(len(entries), 1)
	is.True(entries[0].AllowedHostFQDNs == nil)
}

func TestRepository_SetHostAccess_NotFound_ReturnsErrNotFound(t *testing.T) {
	is := is.New(t)
	fix := setupRepoTest(t)

	err := fix.repo.SetHostAccess(context.Background(), ids.NetworkPolicyID(99999), false, nil)

	is.True(errors.Is(err, networkpolicies.ErrNotFound))
}

// ── GetEnabledCacheEntries ────────────────────────────────────────────────────

func TestRepository_GetEnabledCacheEntries_ReturnsOnlyEnabled(t *testing.T) {
	is := is.New(t)
	fix := setupRepoTest(t)
	ctx := context.Background()

	insertPolicy(t, fix.repo, "active", "10.10.0.0/16")
	disabled, err := fix.repo.CreatePolicy(ctx, networkpolicies.NetworkPolicy{
		Name: "inactive", CIDR: "10.11.0.0/16", Enabled: false,
	})
	is.NoErr(err)
	_ = disabled

	entries, err := fix.repo.GetEnabledCacheEntries(ctx)

	is.NoErr(err)
	is.Equal(len(entries), 1)
	is.Equal(entries[0].CIDR, "10.10.0.0/16")
}

func TestRepository_GetEnabledCacheEntries_AggregatesFQDNsByGroup(t *testing.T) {
	is := is.New(t)
	fix := setupRepoTest(t)
	ctx := context.Background()

	p := insertPolicy(t, fix.repo, "multi-host", "10.12.0.0/16")
	g := insertHostGroup(t, fix.haRepo, "group-multi")
	insertHostInGroup(t, fix.haRepo, "host1.example.com", g)
	insertHostInGroup(t, fix.haRepo, "host2.example.com", g)

	err := fix.repo.SetHostAccess(ctx, p.ID, false, []ids.HostGroupID{g})
	is.NoErr(err)

	entries, err := fix.repo.GetEnabledCacheEntries(ctx)

	is.NoErr(err)
	is.Equal(len(entries), 1)
	is.Equal(len(entries[0].AllowedHostFQDNs), 2)
}

func TestRepository_GetEnabledCacheEntries_DeduplicatesFQDNsAcrossGroups(t *testing.T) {
	is := is.New(t)
	fix := setupRepoTest(t)
	ctx := context.Background()

	p := insertPolicy(t, fix.repo, "shared-host", "10.14.0.0/16")
	g1 := insertHostGroup(t, fix.haRepo, "group-x")
	g2 := insertHostGroup(t, fix.haRepo, "group-y")

	// One host belongs to both groups, and both groups are assigned to the policy.
	hostID, err := fix.haRepo.CreateHost(ctx, hosts.HostDraft{FQDN: "shared.example.com"})
	is.NoErr(err)
	is.NoErr(fix.haRepo.SetHostGroupMembership(ctx, hostID, []ids.HostGroupID{g1, g2}))

	is.NoErr(fix.repo.SetHostAccess(ctx, p.ID, false, []ids.HostGroupID{g1, g2}))

	entries, err := fix.repo.GetEnabledCacheEntries(ctx)

	is.NoErr(err)
	is.Equal(len(entries), 1)
	is.Equal(len(entries[0].AllowedHostFQDNs), 1) // deduped, not counted once per group
}

func TestRepository_GetEnabledCacheEntries_NilFQDNs_WhenNoHostsAssigned(t *testing.T) {
	is := is.New(t)
	fix := setupRepoTest(t)
	ctx := context.Background()

	insertPolicy(t, fix.repo, "no-hosts", "10.13.0.0/16")

	entries, err := fix.repo.GetEnabledCacheEntries(ctx)

	is.NoErr(err)
	is.Equal(len(entries), 1)
	is.True(entries[0].AllowedHostFQDNs == nil)
}
