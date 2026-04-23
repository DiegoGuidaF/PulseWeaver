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

	// Create two known hosts.
	hosts, err := srv.HostAccessService.BulkCreateKnownHosts(t.Context(), []string{"h1.example.com", "h2.example.com"})
	is.NoErr(err)
	hostByFQDN := make(map[string]hostaccess.KnownHostID, len(hosts))
	for _, h := range hosts {
		hostByFQDN[h.FQDN] = h.ID
	}

	// Create a group containing only h2.
	groupID, err := srv.HostAccessService.CreateHostGroup(t.Context(), "G1", nil, nil, []hostaccess.KnownHostID{hostByFQDN["h2.example.com"]})
	is.NoErr(err)

	// Create a regular user.
	user, err := srv.AuthService.CreateUser(t.Context(), "alice", "Alice", "alice@example.com", principal)
	is.NoErr(err)

	// Grant alice: h1 directly, G1 via group.
	err = srv.HostAccessService.SetFullUserGrants(t.Context(), user.ID, nil, []hostaccess.KnownHostID{hostByFQDN["h1.example.com"]}, []hostaccess.HostGroupID{groupID})
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
	is.Equal(alice.Groups[0].Id, groupID.Int64())
	is.Equal(alice.Groups[0].Name, "G1")
}

func TestHandler_ListUsersHostAccess_BypassShowsAllHostsAsEffective(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)
	principal := testutils.AdminPrincipal(t, srv)

	// Create three known hosts.
	_, err := srv.HostAccessService.BulkCreateKnownHosts(t.Context(), []string{"a.example.com", "b.example.com", "c.example.com"})
	is.NoErr(err)

	// Create a bypass user with no explicit grants.
	user, err := srv.AuthService.CreateUser(t.Context(), "charlie", "Charlie", "charlie@example.com", principal)
	is.NoErr(err)
	bypass := true
	err = srv.HostAccessService.SetFullUserGrants(t.Context(), user.ID, &bypass, nil, nil)
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
	hosts, err := srv.HostAccessService.BulkCreateKnownHosts(t.Context(), []string{"h1.example.com", "h2.example.com"})
	is.NoErr(err)
	hostByFQDN := make(map[string]hostaccess.KnownHostID, len(hosts))
	for _, h := range hosts {
		hostByFQDN[h.FQDN] = h.ID
	}

	// Create a group containing only h2.
	groupID, err := srv.HostAccessService.CreateHostGroup(t.Context(), "G1", nil, nil, []hostaccess.KnownHostID{hostByFQDN["h2.example.com"]})
	is.NoErr(err)

	// Create a user with h1 direct + G1 via group.
	user, err := srv.AuthService.CreateUser(t.Context(), "bob", "Bob", "bob@example.com", principal)
	is.NoErr(err)
	err = srv.HostAccessService.SetFullUserGrants(t.Context(), user.ID, nil, []hostaccess.KnownHostID{hostByFQDN["h1.example.com"]}, []hostaccess.HostGroupID{groupID})
	is.NoErr(err)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/admin/host-access/users/%d", user.ID), nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var resp httpapi.UserHostDetails
	is.NoErr(json.NewDecoder(rec.Body).Decode(&resp))

	// Groups: G1 should be granted; any other groups ungranted.
	g1 := findDetailsGroup(resp.Groups, groupID.Int64())
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
	is.Equal(h2.ViaGroup.Id, groupID.Int64())
	is.Equal(h2.ViaGroup.Name, "G1")
}

func TestHandler_GetUserHostDetails_AllGroupsReturnedRegardlessOfGrant(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)
	principal := testutils.AdminPrincipal(t, srv)

	// Create two groups; only grant one.
	groupGranted, err := srv.HostAccessService.CreateHostGroup(t.Context(), "Granted", nil, nil, nil)
	is.NoErr(err)
	groupUngranted, err := srv.HostAccessService.CreateHostGroup(t.Context(), "Ungranted", nil, nil, nil)
	is.NoErr(err)

	user, err := srv.AuthService.CreateUser(t.Context(), "dana", "Dana", "dana@example.com", principal)
	is.NoErr(err)
	err = srv.HostAccessService.SetFullUserGrants(t.Context(), user.ID, nil, nil, []hostaccess.HostGroupID{groupGranted})
	is.NoErr(err)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/admin/host-access/users/%d", user.ID), nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var resp httpapi.UserHostDetails
	is.NoErr(json.NewDecoder(rec.Body).Decode(&resp))

	// Both groups are present; only the granted one has granted=true.
	granted := findDetailsGroup(resp.Groups, groupGranted.Int64())
	is.True(granted != nil)
	is.Equal(granted.Granted, true)

	ungranted := findDetailsGroup(resp.Groups, groupUngranted.Int64())
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
