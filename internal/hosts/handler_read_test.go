//go:build test

package hosts_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

// ── ListHosts ─────────────────────────────────────────────────────────────────

func TestHandler_ListHosts_HappyPath(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	testutils.NewSeeder(t).
		WithGroup(testutils.GroupFixture{Name: "media"}).
		WithHost(testutils.HostFixture{FQDN: "grouped.example.com", Groups: []string{"media"}}).
		WithHost(testutils.HostFixture{FQDN: "unassigned.example.com"}).
		Build(srv)

	resp, err := client.ListHostsWithResponse(ctx)
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	is.Equal(len(resp.JSON200.Hosts), 2)

	grouped := findHost(resp.JSON200.Hosts, "grouped.example.com")
	is.True(grouped != nil)
	is.Equal(len(grouped.Groups), 1)
	is.Equal(grouped.Groups[0].Name, "media")

	unassigned := findHost(resp.JSON200.Hosts, "unassigned.example.com")
	is.True(unassigned != nil)
	is.Equal(len(unassigned.Groups), 0)
}

// TestHandler_ListHosts_MultipleGroups confirms a host in more than one group
// returns every membership, not just one.
func TestHandler_ListHosts_MultipleGroups(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	testutils.NewSeeder(t).
		WithGroup(testutils.GroupFixture{Name: "media"}).
		WithGroup(testutils.GroupFixture{Name: "backend"}).
		WithHost(testutils.HostFixture{FQDN: "multi.example.com", Groups: []string{"media", "backend"}}).
		Build(srv)

	resp, err := client.ListHostsWithResponse(ctx)
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)

	multi := findHost(resp.JSON200.Hosts, "multi.example.com")
	is.True(multi != nil)
	is.Equal(len(multi.Groups), 2)
	names := []string{multi.Groups[0].Name, multi.Groups[1].Name}
	is.True(slices.Contains(names, "media"))
	is.True(slices.Contains(names, "backend"))
}

func TestHandler_ListHosts_Empty(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	resp, err := client.ListHostsWithResponse(ctx)
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	is.Equal(len(resp.JSON200.Hosts), 0)
}

func TestHandler_ListHosts_Unauthenticated(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAPIClient(t, srv)

	resp, err := client.ListHostsWithResponse(ctx)
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusUnauthorized)
}

func findHost(hosts []httpapi.Host, fqdn string) *httpapi.Host {
	for i := range hosts {
		if hosts[i].Fqdn == fqdn {
			return &hosts[i]
		}
	}
	return nil
}

// ── ListHostGroups ────────────────────────────────────────────────────────────

func TestHandler_ListHostGroups_HappyPath(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	testutils.NewSeeder(t).
		WithGroup(testutils.GroupFixture{Name: "media"}).
		WithGroup(testutils.GroupFixture{Name: "empty-group"}).
		WithHost(testutils.HostFixture{FQDN: "a.internal", Groups: []string{"media"}}).
		WithHost(testutils.HostFixture{FQDN: "b.internal", Groups: []string{"media"}}).
		Build(srv)

	resp, err := client.ListHostGroupsWithResponse(ctx)
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	is.Equal(len(resp.JSON200.Groups), 2)

	media := findGroupListItem(resp.JSON200.Groups, "media")
	is.True(media != nil)
	is.Equal(media.HostCount, 2)

	empty := findGroupListItem(resp.JSON200.Groups, "empty-group")
	is.True(empty != nil)
	is.Equal(empty.HostCount, 0)
}

func TestHandler_ListHostGroups_Empty(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	resp, err := client.ListHostGroupsWithResponse(ctx)
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	is.Equal(len(resp.JSON200.Groups), 0)
}

func TestHandler_ListHostGroups_Unauthenticated(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAPIClient(t, srv)

	resp, err := client.ListHostGroupsWithResponse(ctx)
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusUnauthorized)
}

func findGroupListItem(groups []httpapi.GroupListItem, name string) *httpapi.GroupListItem {
	for i := range groups {
		if groups[i].Name == name {
			return &groups[i]
		}
	}
	return nil
}

// ── GetHostGroup ──────────────────────────────────────────────────────────────

// TestHandler_GetHostGroup_HappyPath covers all three member relationships in
// one call: hosts, a granted user, and a network policy assigned via the group.
func TestHandler_GetHostGroup_HappyPath(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	seed := testutils.NewSeeder(t).
		WithGroup(testutils.GroupFixture{Name: "backend"}).
		WithHost(testutils.HostFixture{FQDN: "api1.internal", Groups: []string{"backend"}}).
		WithHost(testutils.HostFixture{FQDN: "api2.internal", Groups: []string{"backend"}}).
		WithUser(testutils.UserFixture{Name: "grantee"}).
		SetUserAccess("grantee", false, "backend").
		WithPolicy(testutils.PolicyFixture{Name: "corp-vpn", CIDR: "10.0.0.0/16"}).
		AssignGroupsToPolicy("corp-vpn", "backend").
		Build(srv)

	resp, err := client.GetHostGroupWithResponse(ctx, seed.Group("backend").Int64())
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)

	detail := resp.JSON200
	is.Equal(detail.Name, "backend")
	is.Equal(len(detail.Hosts), 2)
	is.Equal(len(detail.Users), 1)
	is.Equal(detail.Users[0].Username, "grantee")
	is.Equal(len(detail.NetworkPolicies), 1)
	is.Equal(detail.NetworkPolicies[0].Name, "corp-vpn")
}

func TestHandler_GetHostGroup_EmptyGroup(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	seed := testutils.NewSeeder(t).
		WithGroup(testutils.GroupFixture{Name: "unused"}).
		Build(srv)

	resp, err := client.GetHostGroupWithResponse(ctx, seed.Group("unused").Int64())
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)

	detail := resp.JSON200
	is.Equal(len(detail.Hosts), 0)
	is.Equal(len(detail.Users), 0)
	is.Equal(len(detail.NetworkPolicies), 0)
}

// TestHandler_GetHostGroup_DeletedUserExcluded confirms a soft-deleted user
// granted to a group no longer appears in that group's user list.
func TestHandler_GetHostGroup_DeletedUserExcluded(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	seed := testutils.NewSeeder(t).
		WithGroup(testutils.GroupFixture{Name: "mixed"}).
		WithUser(testutils.UserFixture{Name: "active-user"}).
		WithUser(testutils.UserFixture{Name: "deleted-user"}).
		SetUserAccess("active-user", false, "mixed").
		SetUserAccess("deleted-user", false, "mixed").
		Build(srv)

	admin := testutils.AdminPrincipal(t, srv)
	is.NoErr(srv.AuthService.DeleteUser(ctx, admin, seed.User("deleted-user")))

	resp, err := client.GetHostGroupWithResponse(ctx, seed.Group("mixed").Int64())
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)

	is.Equal(len(resp.JSON200.Users), 1)
	is.Equal(resp.JSON200.Users[0].Username, "active-user")
}

func TestHandler_GetHostGroup_NotFound(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	resp, err := client.GetHostGroupWithResponse(ctx, 99999)
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNotFound)
}

func TestHandler_GetHostGroup_Unauthenticated(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAPIClient(t, srv)

	resp, err := client.GetHostGroupWithResponse(ctx, 1)
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusUnauthorized)
}
