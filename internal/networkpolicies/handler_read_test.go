//go:build test

package networkpolicies_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

// ── ListNetworkPolicies ───────────────────────────────────────────────────────

func TestHandler_ListNetworkPolicies_Empty(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	resp, err := client.ListNetworkPoliciesWithResponse(ctx)
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	is.Equal(len(*resp.JSON200), 0)
}

func TestHandler_ListNetworkPolicies_ReturnsSummary(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	testutils.NewSeeder(t).
		WithGroup(testutils.GroupFixture{Name: "lan"}).
		WithHost(testutils.HostFixture{FQDN: "a.lan", Groups: []string{"lan"}}).
		WithHost(testutils.HostFixture{FQDN: "b.lan", Groups: []string{"lan"}}).
		WithPolicy(testutils.PolicyFixture{Name: "home", CIDR: "192.168.1.0/24"}).
		AssignGroupsToPolicy("home", "lan").
		WithPolicy(testutils.PolicyFixture{Name: "ops", CIDR: "10.0.0.0/16"}).
		WithPolicyBypassHostCheck("ops").
		Build(srv)

	resp, err := client.ListNetworkPoliciesWithResponse(ctx)
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	is.Equal(len(*resp.JSON200), 2)

	home := findPolicy(*resp.JSON200, "home")
	is.True(home != nil)
	is.Equal(home.Cidr, "192.168.1.0/24")
	is.Equal(home.BypassHostCheck, false)
	is.Equal(home.HostCount, 2)
	is.Equal(len(home.Groups), 1)
	is.True(!time.Time(home.CreatedAt).IsZero())

	ops := findPolicy(*resp.JSON200, "ops")
	is.True(ops != nil)
	is.Equal(ops.BypassHostCheck, true)
	is.Equal(len(ops.Groups), 0)
}

func findPolicy(policies []httpapi.NetworkPolicyListItem, name string) *httpapi.NetworkPolicyListItem {
	for i := range policies {
		if policies[i].Name == name {
			return &policies[i]
		}
	}
	return nil
}

func TestHandler_ListNetworkPolicies_Unauthenticated(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAPIClient(t, srv)

	resp, err := client.ListNetworkPoliciesWithResponse(ctx)
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusUnauthorized)
}

// ── GetNetworkPolicy ──────────────────────────────────────────────────────────

func TestHandler_GetNetworkPolicy_ReturnsDetail(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	seed := testutils.NewSeeder(t).
		WithGroup(testutils.GroupFixture{Name: "assigned"}).
		WithGroup(testutils.GroupFixture{Name: "unassigned"}).
		WithHost(testutils.HostFixture{FQDN: "a.lan", Groups: []string{"assigned"}}).
		WithHost(testutils.HostFixture{FQDN: "b.lan", Groups: []string{"assigned"}}).
		WithPolicy(testutils.PolicyFixture{Name: "edge", CIDR: "203.0.113.0/24", Desc: "edge network"}).
		AssignGroupsToPolicy("edge", "assigned").
		Build(srv)

	resp, err := client.GetNetworkPolicyWithResponse(ctx, seed.Policy("edge").Int64())
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)

	detail := resp.JSON200
	is.Equal(detail.Name, "edge")
	is.Equal(detail.Cidr, "203.0.113.0/24")
	is.True(detail.Description != nil)
	is.Equal(*detail.Description, "edge network")
	is.Equal(detail.BypassHostCheck, false)
	is.True(!time.Time(detail.UpdatedAt).IsZero())

	// Both groups returned; only "assigned" is granted.
	is.Equal(len(detail.Groups), 2)
	assigned := findSubjectGroup(detail.Groups, "assigned")
	is.True(assigned != nil)
	is.True(assigned.Granted)
	is.Equal(len(assigned.Hosts), 2)

	unassigned := findSubjectGroup(detail.Groups, "unassigned")
	is.True(unassigned != nil)
	is.Equal(unassigned.Granted, false)
	is.Equal(len(unassigned.Hosts), 0)
}

func findSubjectGroup(groups []httpapi.SubjectGroupDetail, name string) *httpapi.SubjectGroupDetail {
	for i := range groups {
		if groups[i].Name == name {
			return &groups[i]
		}
	}
	return nil
}

func TestHandler_GetNetworkPolicy_NotFound(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	resp, err := client.GetNetworkPolicyWithResponse(ctx, 99999)
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNotFound)
}

func TestHandler_GetNetworkPolicy_Unauthenticated(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAPIClient(t, srv)

	resp, err := client.GetNetworkPolicyWithResponse(ctx, 1)
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusUnauthorized)
}
