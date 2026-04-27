//go:build test

package queries_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/hostaccess"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func findUserRow(rows []httpapi.UserHostAccessSummary, id int64) *httpapi.UserHostAccessSummary {
	for i, r := range rows {
		if r.Id == id {
			return &rows[i]
		}
	}
	return nil
}

func findDetailsGroup(groups []httpapi.UserHostDetailsGroup, id int64) *httpapi.UserHostDetailsGroup {
	for i, g := range groups {
		if g.Id == id {
			return &groups[i]
		}
	}
	return nil
}

func findDetailsHost(hosts []httpapi.UserHostDetailsHost, fqdn string) *httpapi.UserHostDetailsHost {
	for i, h := range hosts {
		if h.Fqdn == fqdn {
			return &hosts[i]
		}
	}
	return nil
}

// ── ListUsersHostAccess ───────────────────────────────────────────────────────

func TestHandler_ListUsersHostAccess_AdminHasNoGrantsByDefault(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/host-access/users", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var rows []httpapi.UserHostAccessSummary
	is.NoErr(json.NewDecoder(rec.Body).Decode(&rows))
	is.Equal(len(rows), 1)

	admin := rows[0]
	is.Equal(admin.Bypass, false)
	is.Equal(admin.DirectHostCount, 0)
	is.Equal(len(admin.Groups), 0)
}

func TestHandler_ListUsersHostAccess_EffectiveCountCombinesDirectAndGroup(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)
	principal := testutils.AdminPrincipal(t, srv)

	createHostInput := hostaccess.ReconcileKnownHostsInput{Hosts: []hostaccess.DesiredKnownHost{
		{FQDN: "h1.example.com"}, // Should get ID 1
		{FQDN: "h2.example.com"}, // Should get ID 2
	}}

	// Create two known hosts.
	err := srv.HostAccessService.ReconcileKnownHosts(t.Context(), createHostInput)
	is.NoErr(err)

	// Create a group containing only h2.
	err = srv.HostAccessService.ReconcileHostGroups(t.Context(), hostaccess.ReconcileHostGroupsInput{Groups: []hostaccess.DesiredHostGroup{
		{Name: "G1", HostIDs: []hostaccess.KnownHostID{2}},
	}})
	is.NoErr(err)

	// Create a regular user.
	user, err := srv.AuthService.CreateUser(t.Context(), "alice", "Alice", "alice@example.com", principal)
	is.NoErr(err)

	// Grant alice: h1 directly, G1 via group.
	err = srv.HostAccessService.SetFullUserGrants(t.Context(), user.ID, nil, []hostaccess.KnownHostID{1}, []hostaccess.HostGroupID{1})
	is.NoErr(err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/host-access/users", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var rows []httpapi.UserHostAccessSummary
	is.NoErr(json.NewDecoder(rec.Body).Decode(&rows))

	alice := findUserRow(rows, user.ID.Int64())
	is.True(alice != nil)
	is.Equal(alice.Bypass, false)
	is.Equal(alice.DirectHostCount, 2) // h1 direct + h2 via group = 2 effective
	is.Equal(len(alice.Groups), 1)
	is.Equal(alice.Groups[0].Id, int64(1))
	is.Equal(alice.Groups[0].Name, "G1")
}

func TestHandler_ListUsersHostAccess_BypassShowsAllHostsAsEffective(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)
	principal := testutils.AdminPrincipal(t, srv)

	// Create three known hosts.
	createHostInput := hostaccess.ReconcileKnownHostsInput{Hosts: []hostaccess.DesiredKnownHost{
		{FQDN: "a.example.com"},
		{FQDN: "b.example.com"},
		{FQDN: "c.example.com"},
	}}
	err := srv.HostAccessService.ReconcileKnownHosts(t.Context(), createHostInput)
	is.NoErr(err)

	// Create a bypass user with no explicit grants.
	user, err := srv.AuthService.CreateUser(t.Context(), "charlie", "Charlie", "charlie@example.com", principal)
	is.NoErr(err)

	err = srv.HostAccessService.SetFullUserGrants(t.Context(), user.ID, new(true), nil, nil)
	is.NoErr(err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/host-access/users", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var rows []httpapi.UserHostAccessSummary
	is.NoErr(json.NewDecoder(rec.Body).Decode(&rows))

	charlie := findUserRow(rows, user.ID.Int64())
	is.True(charlie != nil)
	is.Equal(charlie.Bypass, true)
	is.Equal(charlie.DirectHostCount, 3) // all 3 known hosts are effectively accessible
	is.Equal(len(charlie.Groups), 0)
}

func TestHandler_ListUsersHostAccess_Unauthenticated(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/host-access/users", nil)
	rec := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusUnauthorized)
}

// ── GetUserHostDetails ────────────────────────────────────────────────────────

func TestHandler_GetUserHostDetails_UserNotFound(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/host-access/users/99999", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusNotFound)
}

func TestHandler_GetUserHostDetails_DirectAndGroupGrant(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)
	principal := testutils.AdminPrincipal(t, srv)

	// Create two known hosts.
	createHostInput := hostaccess.ReconcileKnownHostsInput{Hosts: []hostaccess.DesiredKnownHost{
		{FQDN: "h1.example.com"}, // Should get ID 1
		{FQDN: "h2.example.com"}, // Should get ID 2
	}}
	err := srv.HostAccessService.ReconcileKnownHosts(t.Context(), createHostInput)
	is.NoErr(err)

	// Create a group containing only h2.
	err = srv.HostAccessService.ReconcileHostGroups(t.Context(), hostaccess.ReconcileHostGroupsInput{Groups: []hostaccess.DesiredHostGroup{
		{Name: "G1", HostIDs: []hostaccess.KnownHostID{2}},
	}})
	is.NoErr(err)

	// Create a user with h1 direct + G1 via group.
	user, err := srv.AuthService.CreateUser(t.Context(), "bob", "Bob", "bob@example.com", principal)
	is.NoErr(err)
	err = srv.HostAccessService.SetFullUserGrants(t.Context(), user.ID, nil, []hostaccess.KnownHostID{1}, []hostaccess.HostGroupID{1})
	is.NoErr(err)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/admin/host-access/users/%d", user.ID), nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var resp httpapi.UserHostDetails
	is.NoErr(json.NewDecoder(rec.Body).Decode(&resp))

	// Groups: G1 should be granted; any other groups ungranted.
	g1 := findDetailsGroup(resp.Groups, 1)
	is.True(g1 != nil)
	is.Equal(g1.Granted, true)
	is.Equal(len(g1.Hosts), 1)
	is.Equal(g1.Hosts[0].Fqdn, "h2.example.com")

	// h1: directly granted, not via group.
	h1 := findDetailsHost(resp.Hosts, "h1.example.com")
	is.True(h1 != nil)
	is.Equal(h1.DirectlyGranted, true)
	is.True(h1.ViaGroup == nil)

	// h2: not directly granted, but covered by G1.
	h2 := findDetailsHost(resp.Hosts, "h2.example.com")
	is.True(h2 != nil)
	is.Equal(h2.DirectlyGranted, false)
	is.True(h2.ViaGroup != nil)
	is.Equal(h2.ViaGroup.Id, int64(1))
	is.Equal(h2.ViaGroup.Name, "G1")
}

func TestHandler_GetUserHostDetails_AllGroupsReturnedRegardlessOfGrant(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)
	principal := testutils.AdminPrincipal(t, srv)

	// Create two groups; only grant one.
	err := srv.HostAccessService.ReconcileHostGroups(t.Context(), hostaccess.ReconcileHostGroupsInput{Groups: []hostaccess.DesiredHostGroup{
		{Name: "Granted"},
		{Name: "Ungranted"},
	}})
	is.NoErr(err)

	user, err := srv.AuthService.CreateUser(t.Context(), "dana", "Dana", "dana@example.com", principal)
	is.NoErr(err)
	err = srv.HostAccessService.SetFullUserGrants(t.Context(), user.ID, nil, nil, []hostaccess.HostGroupID{1})
	is.NoErr(err)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/admin/host-access/users/%d", user.ID), nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var resp httpapi.UserHostDetails
	is.NoErr(json.NewDecoder(rec.Body).Decode(&resp))

	// Both groups are present; only the granted one has granted=true.
	granted := findDetailsGroup(resp.Groups, int64(1))
	is.True(granted != nil)
	is.Equal(granted.Granted, true)

	ungranted := findDetailsGroup(resp.Groups, int64(2))
	is.True(ungranted != nil)
	is.Equal(ungranted.Granted, false)
}

func TestHandler_GetUserHostDetails_Unauthenticated(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/host-access/users/1", nil)
	rec := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusUnauthorized)
}
