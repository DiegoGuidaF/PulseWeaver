//go:build test

package queries_test

import (
	"errors"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/networkpolicies"
	"github.com/matryer/is"
)

// ── test helpers ──────────────────────────────────────────────────────────────

func insertTestPolicy(t *testing.T, db *database.DB, name, cidr string, bypass bool) ids.NetworkPolicyID {
	t.Helper()
	var id ids.NetworkPolicyID
	if err := db.QueryRowxContext(t.Context(),
		`INSERT INTO network_policies (name, cidr, enabled, bypass_host_check) VALUES (?, ?, 1, ?) RETURNING id`,
		name, cidr, bypass,
	).Scan(&id); err != nil {
		t.Fatalf("insertTestPolicy(%q): %v", name, err)
	}
	return id
}

func assignGroupToPolicy(t *testing.T, db *database.DB, policyID ids.NetworkPolicyID, groupID ids.HostGroupID) {
	t.Helper()
	if _, err := db.ExecContext(t.Context(),
		`INSERT INTO network_policy_allowed_host_groups (policy_id, host_group_id) VALUES (?, ?)`,
		policyID, groupID,
	); err != nil {
		t.Fatalf("assignGroupToPolicy: %v", err)
	}
}

// ── GetNetworkPolicySummaries ─────────────────────────────────────────────────

func TestRepository_GetNetworkPolicySummaries_EmptyList(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	summaries, err := repos.queries.GetNetworkPolicySummaries(t.Context())

	is.NoErr(err)
	is.Equal(len(summaries), 0)
}

func TestRepository_GetNetworkPolicySummaries_EffectiveCountViaGroupsOnly(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	policyID := insertTestPolicy(t, repos.db, "home", "192.168.1.0/24", false)
	groupID := insertTestHostGroup(t, repos.db, "lan")
	h1 := insertTestHost(t, repos.db, "app.lan")
	h2 := insertTestHost(t, repos.db, "db.lan")
	addHostToGroup(t, repos.db, groupID, h1)
	addHostToGroup(t, repos.db, groupID, h2)
	assignGroupToPolicy(t, repos.db, policyID, groupID)

	summaries, err := repos.queries.GetNetworkPolicySummaries(t.Context())

	is.NoErr(err)
	is.Equal(len(summaries), 1)
	is.Equal(summaries[0].EffectiveHostCount, 2)
	is.Equal(summaries[0].TotalHostCount, 2)
}

func TestRepository_GetNetworkPolicySummaries_BypassReturnsTotal(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	// bypass policy with no group assignments
	insertTestPolicy(t, repos.db, "vpn", "10.0.0.0/8", true)
	insertTestHost(t, repos.db, "srv1.internal")
	insertTestHost(t, repos.db, "srv2.internal")

	summaries, err := repos.queries.GetNetworkPolicySummaries(t.Context())

	is.NoErr(err)
	is.Equal(len(summaries), 1)
	is.Equal(summaries[0].TotalHostCount, 2)
	is.Equal(summaries[0].EffectiveHostCount, 2) // bypass → same as total
}

func TestRepository_GetNetworkPolicySummaries_NoGroupAssignment_ZeroEffective(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	insertTestPolicy(t, repos.db, "isolated", "172.16.0.0/12", false)

	summaries, err := repos.queries.GetNetworkPolicySummaries(t.Context())

	is.NoErr(err)
	is.Equal(len(summaries), 1)
	is.Equal(summaries[0].EffectiveHostCount, 0)
}

// ── GetNetworkPolicyDetail ───────────────────────────────────────────────────

func TestRepository_GetNetworkPolicyDetail_NotFound(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	_, err := repos.queries.GetNetworkPolicyDetail(t.Context(), ids.NetworkPolicyID(99999))

	is.True(errors.Is(err, networkpolicies.ErrNotFound))
}

func TestRepository_GetNetworkPolicyDetail_ReturnsGroupsWithHosts(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	policyID := insertTestPolicy(t, repos.db, "edge", "203.0.113.0/24", false)
	groupID := insertTestHostGroup(t, repos.db, "public")
	h := insertTestHost(t, repos.db, "cdn.example.com")
	addHostToGroup(t, repos.db, groupID, h)
	assignGroupToPolicy(t, repos.db, policyID, groupID)

	detail, err := repos.queries.GetNetworkPolicyDetail(t.Context(), policyID)

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
	repos := setupRepos(t)

	policyID := insertTestPolicy(t, repos.db, "partial", "198.51.100.0/24", false)
	assigned := insertTestHostGroup(t, repos.db, "assigned-group")
	unassigned := insertTestHostGroup(t, repos.db, "unassigned-group")
	_ = unassigned
	assignGroupToPolicy(t, repos.db, policyID, assigned)

	detail, err := repos.queries.GetNetworkPolicyDetail(t.Context(), policyID)

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
	repos := setupRepos(t)

	policyID := insertTestPolicy(t, repos.db, "dedup", "100.64.0.0/16", false)
	g1 := insertTestHostGroup(t, repos.db, "g1")
	g2 := insertTestHostGroup(t, repos.db, "g2")
	shared := insertTestHost(t, repos.db, "shared.example.com")
	// shared host appears in both groups
	addHostToGroup(t, repos.db, g1, shared)
	addHostToGroup(t, repos.db, g2, shared)
	assignGroupToPolicy(t, repos.db, policyID, g1)
	assignGroupToPolicy(t, repos.db, policyID, g2)

	detail, err := repos.queries.GetNetworkPolicyDetail(t.Context(), policyID)

	is.NoErr(err)
	// COUNT(DISTINCT host_id) should deduplicate the shared host
	is.Equal(detail.EffectiveHostCount, 1)
}
