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
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func findUserRow(rows []httpapi.UserListItem, id int64) *httpapi.UserListItem {
	for i, r := range rows {
		if r.Id == id {
			return &rows[i]
		}
	}
	return nil
}

func findDetailsGroup(groups []httpapi.SubjectGroupDetail, id int64) *httpapi.SubjectGroupDetail {
	for i, g := range groups {
		if g.Id == id {
			return &groups[i]
		}
	}
	return nil
}

// ── ListUsersHostAccess ───────────────────────────────────────────────────────

func TestHandler_ListUsersHostAccess_AdminHasNoGrantsByDefault(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/access/users", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var rows []httpapi.UserListItem
	is.NoErr(json.NewDecoder(rec.Body).Decode(&rows))
	is.Equal(len(rows), 1)

	admin := rows[0]
	is.Equal(admin.BypassHostCheck, false)
	is.Equal(admin.HostCount, 0)
	is.Equal(len(admin.Groups), 0)
}

func TestHandler_ListUsersHostAccess_EffectiveCountCombinesDirectAndGroup(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)
	principal := testutils.AdminPrincipal(t, srv)

	createHostInput := hostaccess.ReconcileHostsInput{Hosts: []hostaccess.DesiredHost{
		{FQDN: "h1.example.com"}, // Should get ID 1
		{FQDN: "h2.example.com"}, // Should get ID 2
	}}

	// Create two hosts.
	err := srv.HostAccessService.ReconcileHosts(t.Context(), createHostInput)
	is.NoErr(err)

	// Create a group containing only h2.
	err = srv.HostAccessService.ReconcileHostGroups(t.Context(), hostaccess.ReconcileHostGroupsInput{Groups: []hostaccess.DesiredHostGroup{
		{Name: "G1", HostIDs: []ids.HostID{2}},
	}})
	is.NoErr(err)

	// Create a regular user.
	user, err := srv.AuthService.CreateUser(t.Context(), "alice", "Alice", "alice@example.com", principal)
	is.NoErr(err)

	// Grant alice group G1 (which contains h2).
	err = srv.HostAccessService.SetUserAccess(t.Context(), user.ID, false, []ids.HostGroupID{1})
	is.NoErr(err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/access/users", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var rows []httpapi.UserListItem
	is.NoErr(json.NewDecoder(rec.Body).Decode(&rows))

	alice := findUserRow(rows, user.ID.Int64())
	is.True(alice != nil)
	is.Equal(alice.BypassHostCheck, false)
	is.Equal(alice.HostCount, 1) // h2 via G1 = 1 effective
	is.Equal(len(alice.Groups), 1)
	is.Equal(alice.Groups[0].Id, int64(1))
	is.Equal(alice.Groups[0].Name, "G1")
}

func TestHandler_ListUsersHostAccess_BypassShowsAllHostsAsEffective(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)
	principal := testutils.AdminPrincipal(t, srv)

	// Create three hosts.
	createHostInput := hostaccess.ReconcileHostsInput{Hosts: []hostaccess.DesiredHost{
		{FQDN: "a.example.com"},
		{FQDN: "b.example.com"},
		{FQDN: "c.example.com"},
	}}
	err := srv.HostAccessService.ReconcileHosts(t.Context(), createHostInput)
	is.NoErr(err)

	// Create a bypass user with no explicit grants.
	user, err := srv.AuthService.CreateUser(t.Context(), "charlie", "Charlie", "charlie@example.com", principal)
	is.NoErr(err)

	err = srv.HostAccessService.SetUserAccess(t.Context(), user.ID, true, nil)
	is.NoErr(err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/access/users", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var rows []httpapi.UserListItem
	is.NoErr(json.NewDecoder(rec.Body).Decode(&rows))

	charlie := findUserRow(rows, user.ID.Int64())
	is.True(charlie != nil)
	is.Equal(charlie.BypassHostCheck, true)
	is.Equal(charlie.HostCount, 3) // all 3 hosts are effectively accessible
	is.Equal(len(charlie.Groups), 0)
}

func TestHandler_ListUsersHostAccess_Unauthenticated(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/access/users", nil)
	rec := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusUnauthorized)
}

// ── GetUserHostDetails ────────────────────────────────────────────────────────

func TestHandler_GetUserHostDetails_UserNotFound(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/access/users/99999", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusNotFound)
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
	err = srv.HostAccessService.SetUserAccess(t.Context(), user.ID, false, []ids.HostGroupID{1})
	is.NoErr(err)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/admin/access/users/%d", user.ID), nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var resp httpapi.UserAccessDetail
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

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/access/users/1", nil)
	rec := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusUnauthorized)
}

// ── ListHostGroups ────────────────────────────────────────────────────────────

func TestHandler_ListHostGroups_HappyPath(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)
	principal := testutils.AdminPrincipal(t, srv)

	err := srv.HostAccessService.ReconcileHosts(t.Context(), hostaccess.ReconcileHostsInput{
		Hosts: []hostaccess.DesiredHost{{FQDN: "app.example.com"}},
	})
	is.NoErr(err)

	err = srv.HostAccessService.ReconcileHostGroups(t.Context(), hostaccess.ReconcileHostGroupsInput{
		Groups: []hostaccess.DesiredHostGroup{{Name: "backend", HostIDs: []ids.HostID{1}}},
	})
	is.NoErr(err)

	user, err := srv.AuthService.CreateUser(t.Context(), "alice", "Alice", "alice@example.com", principal)
	is.NoErr(err)
	err = srv.HostAccessService.SetUserAccess(t.Context(), user.ID, false, []ids.HostGroupID{1})
	is.NoErr(err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/access/host-groups", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var resp httpapi.GroupListResponse
	is.NoErr(json.NewDecoder(rec.Body).Decode(&resp))
	is.Equal(len(resp.Groups), 1)
	g := resp.Groups[0]
	is.Equal(g.Name, "backend")
	is.Equal(len(g.Hosts), 1)
	is.True(g.Users != nil)
	is.Equal(len(*g.Users), 1)
	is.Equal((*g.Users)[0].Username, "alice")
}

func TestHandler_ListHostGroups_Unauthenticated(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/access/host-groups", nil)
	rec := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusUnauthorized)
}
