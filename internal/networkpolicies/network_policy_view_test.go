//go:build test

package networkpolicies_test

import (
	"errors"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/hosts"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/networkpolicies"
	"github.com/matryer/is"
)

// ── GetNetworkPolicySummaries ─────────────────────────────────────────────────

func TestRepository_GetNetworkPolicySummaries_EmptyList(t *testing.T) {
	is := is.New(t)
	fix := setupRepoTest(t)

	summaries, err := fix.repo.GetNetworkPolicySummaries(t.Context())

	is.NoErr(err)
	is.Equal(len(summaries), 0)
}

func TestRepository_GetNetworkPolicySummaries_EffectiveCountViaGroupsOnly(t *testing.T) {
	is := is.New(t)
	fix := setupRepoTest(t)
	ctx := t.Context()

	p := insertPolicy(t, fix.repo, "home", "192.168.1.0/24")
	g := insertHostGroup(t, fix.haRepo, "lan")
	insertHostInGroup(t, fix.haRepo, "app.lan", g)
	insertHostInGroup(t, fix.haRepo, "db.lan", g)
	is.NoErr(fix.repo.SetHostAccess(ctx, p.ID, false, []ids.HostGroupID{g}))

	summaries, err := fix.repo.GetNetworkPolicySummaries(ctx)

	is.NoErr(err)
	is.Equal(len(summaries), 1)
	is.Equal(summaries[0].EffectiveHostCount, 2)
	is.Equal(summaries[0].TotalHostCount, 2)
}

func TestRepository_GetNetworkPolicySummaries_BypassReturnsTotal(t *testing.T) {
	is := is.New(t)
	fix := setupRepoTest(t)
	ctx := t.Context()

	// bypass policy with no group assignments
	p := insertPolicy(t, fix.repo, "vpn", "10.0.0.0/8")
	is.NoErr(fix.repo.SetHostAccess(ctx, p.ID, true, nil))
	_, err := fix.haRepo.CreateHost(ctx, hosts.HostDraft{FQDN: "srv1.internal"})
	is.NoErr(err)
	_, err = fix.haRepo.CreateHost(ctx, hosts.HostDraft{FQDN: "srv2.internal"})
	is.NoErr(err)

	summaries, err := fix.repo.GetNetworkPolicySummaries(ctx)

	is.NoErr(err)
	is.Equal(len(summaries), 1)
	is.Equal(summaries[0].TotalHostCount, 2)
	is.Equal(summaries[0].EffectiveHostCount, 2) // bypass → same as total
}

func TestRepository_GetNetworkPolicySummaries_NoGroupAssignment_ZeroEffective(t *testing.T) {
	is := is.New(t)
	fix := setupRepoTest(t)

	insertPolicy(t, fix.repo, "isolated", "172.16.0.0/12")

	summaries, err := fix.repo.GetNetworkPolicySummaries(t.Context())

	is.NoErr(err)
	is.Equal(len(summaries), 1)
	is.Equal(summaries[0].EffectiveHostCount, 0)
	is.Equal(len(summaries[0].Groups), 0) // no assignment → empty group list
}

func TestRepository_GetNetworkPolicySummaries_GroupsPopulated(t *testing.T) {
	is := is.New(t)
	fix := setupRepoTest(t)
	ctx := t.Context()

	p := insertPolicy(t, fix.repo, "home", "192.168.1.0/24")
	g := insertHostGroup(t, fix.haRepo, "lan")
	is.NoErr(fix.repo.SetHostAccess(ctx, p.ID, false, []ids.HostGroupID{g}))

	summaries, err := fix.repo.GetNetworkPolicySummaries(ctx)

	is.NoErr(err)
	is.Equal(len(summaries), 1)
	is.Equal(len(summaries[0].Groups), 1)
	is.Equal(summaries[0].Groups[0].ID, g)
	is.Equal(summaries[0].Groups[0].Name, "lan")
}

func TestRepository_GetNetworkPolicySummaries_CreatedAtNonZero(t *testing.T) {
	is := is.New(t)
	fix := setupRepoTest(t)

	insertPolicy(t, fix.repo, "ts-check", "10.0.0.0/8")

	summaries, err := fix.repo.GetNetworkPolicySummaries(t.Context())

	is.NoErr(err)
	is.Equal(len(summaries), 1)
	is.True(!summaries[0].CreatedAt.IsZero())
}

// ── GetNetworkPolicyDetail ───────────────────────────────────────────────────

func TestRepository_GetNetworkPolicyDetail_NotFound(t *testing.T) {
	is := is.New(t)
	fix := setupRepoTest(t)

	_, err := fix.repo.GetNetworkPolicyDetail(t.Context(), ids.NetworkPolicyID(99999))

	is.True(errors.Is(err, networkpolicies.ErrNotFound))
}

func TestRepository_GetNetworkPolicyDetail_ReturnsGroupsWithHosts(t *testing.T) {
	is := is.New(t)
	fix := setupRepoTest(t)
	ctx := t.Context()

	p := insertPolicy(t, fix.repo, "edge", "203.0.113.0/24")
	g := insertHostGroup(t, fix.haRepo, "public")
	insertHostInGroup(t, fix.haRepo, "cdn.example.com", g)
	is.NoErr(fix.repo.SetHostAccess(ctx, p.ID, false, []ids.HostGroupID{g}))

	detail, err := fix.repo.GetNetworkPolicyDetail(ctx, p.ID)

	is.NoErr(err)
	is.Equal(detail.Name, "edge")
	is.Equal(detail.EffectiveHostCount, 1)
	is.Equal(len(detail.HostGroups), 1)
	is.Equal(detail.HostGroups[0].Assigned, true)
	is.Equal(len(detail.HostGroups[0].Hosts), 1)
	is.Equal(detail.HostGroups[0].Hosts[0].FQDN, "cdn.example.com")
}

func TestRepository_GetNetworkPolicyDetail_UnassignedGroupsIncluded(t *testing.T) {
	is := is.New(t)
	fix := setupRepoTest(t)
	ctx := t.Context()

	p := insertPolicy(t, fix.repo, "partial", "198.51.100.0/24")
	assigned := insertHostGroup(t, fix.haRepo, "assigned-group")
	insertHostGroup(t, fix.haRepo, "unassigned-group")
	is.NoErr(fix.repo.SetHostAccess(ctx, p.ID, false, []ids.HostGroupID{assigned}))

	detail, err := fix.repo.GetNetworkPolicyDetail(ctx, p.ID)

	is.NoErr(err)
	// Both groups appear; only one is marked assigned
	is.Equal(len(detail.HostGroups), 2)
	assignedCount := 0
	for _, g := range detail.HostGroups {
		if g.Assigned {
			assignedCount++
		}
	}
	is.Equal(assignedCount, 1)
}

func TestRepository_GetNetworkPolicyDetail_EffectiveCountDeduplicatesSharedHosts(t *testing.T) {
	is := is.New(t)
	fix := setupRepoTest(t)
	ctx := t.Context()

	p := insertPolicy(t, fix.repo, "dedup", "100.64.0.0/16")
	g1 := insertHostGroup(t, fix.haRepo, "g1")
	g2 := insertHostGroup(t, fix.haRepo, "g2")
	shared, err := fix.haRepo.CreateHost(ctx, hosts.HostDraft{FQDN: "shared.example.com"})
	is.NoErr(err)
	// shared host appears in both groups
	is.NoErr(fix.haRepo.SetHostGroupMembership(ctx, shared, []ids.HostGroupID{g1, g2}))
	is.NoErr(fix.repo.SetHostAccess(ctx, p.ID, false, []ids.HostGroupID{g1, g2}))

	detail, err := fix.repo.GetNetworkPolicyDetail(ctx, p.ID)

	is.NoErr(err)
	// COUNT(DISTINCT host_id) should deduplicate the shared host
	is.Equal(detail.EffectiveHostCount, 1)
}
