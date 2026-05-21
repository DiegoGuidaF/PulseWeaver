//go:build test

package queries_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func findHost(hosts []httpapi.Host, fqdn string) *httpapi.Host {
	for i := range hosts {
		if hosts[i].Fqdn == fqdn {
			return &hosts[i]
		}
	}
	return nil
}

func findGroupWithUsers(groups []httpapi.GroupDetailWithUsers, name string) *httpapi.GroupDetailWithUsers {
	for i := range groups {
		if groups[i].Name == name {
			return &groups[i]
		}
	}
	return nil
}

func findUserRow(rows []httpapi.UserListItem, id int64) *httpapi.UserListItem {
	for i := range rows {
		if rows[i].Id == id {
			return &rows[i]
		}
	}
	return nil
}

func findNetworkPolicyRef(policies []httpapi.NetworkPolicyRef, name string) *httpapi.NetworkPolicyRef {
	for i := range policies {
		if policies[i].Name == name {
			return &policies[i]
		}
	}
	return nil
}

// ── ListHosts ─────────────────────────────────────────────────────────────────

func TestHandler_ListHosts_HappyPath(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	testutils.SeedFullWorld(t, srv).Build()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/access/hosts", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)
	var resp httpapi.HostListResponse
	is.NoErr(json.NewDecoder(w.Body).Decode(&resp))
	is.Equal(len(resp.Hosts), 4) // FixtureHostBackend1+2, FixtureHostFrontend1+2

	api1 := findHost(resp.Hosts, testutils.FixtureHostBackend1.FQDN)
	is.True(api1 != nil)
	is.Equal(len(api1.Groups), 1)
	is.Equal(api1.Groups[0].Name, testutils.FixtureGroupBackend.Name)

	api2 := findHost(resp.Hosts, testutils.FixtureHostBackend2.FQDN)
	is.True(api2 != nil)
	is.Equal(len(api2.Groups), 1)
	is.Equal(api2.Groups[0].Name, testutils.FixtureGroupBackend.Name)

	web1 := findHost(resp.Hosts, testutils.FixtureHostFrontend1.FQDN)
	is.True(web1 != nil)
	is.Equal(len(web1.Groups), 1)
	is.Equal(web1.Groups[0].Name, testutils.FixtureGroupFrontend.Name)

	web2 := findHost(resp.Hosts, testutils.FixtureHostFrontend2.FQDN)
	is.True(web2 != nil)
	is.Equal(len(web2.Groups), 1)
	is.Equal(web2.Groups[0].Name, testutils.FixtureGroupFrontend.Name)
}

func TestHandler_ListHosts_Empty(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/access/hosts", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)
	var resp httpapi.HostListResponse
	is.NoErr(json.NewDecoder(w.Body).Decode(&resp))
	is.Equal(len(resp.Hosts), 0)
}

func TestHandler_ListHosts_Unauthenticated(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/access/hosts", nil)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusUnauthorized)
}

// ── ListHostGroups ────────────────────────────────────────────────────────────

// TestHandler_ListHostGroups_HappyPath covers all valid group data paths in one
// call: group with hosts+users+network policy, group with different membership,
// and an empty group with none of the above.
func TestHandler_ListHostGroups_HappyPath(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	testutils.SeedFullWorld(t, srv).Build()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/access/host-groups", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)
	var resp httpapi.GroupListResponse
	is.NoErr(json.NewDecoder(w.Body).Decode(&resp))
	is.Equal(len(resp.Groups), 3) // FixtureGroupBackend, FixtureGroupFrontend, FixtureGroupEmpty

	// backend: 2 hosts, 2 users (alice+charlie), 1 network policy (corp-vpn)
	backend := findGroupWithUsers(resp.Groups, testutils.FixtureGroupBackend.Name)
	is.True(backend != nil)
	is.Equal(len(backend.Hosts), 2) // FixtureHostBackend1+2
	is.True(backend.Users != nil)
	is.Equal(len(*backend.Users), 2) // FixtureUserWithAccess + FixtureUserBypassAccess
	is.Equal(len(backend.NetworkPolicies), 1)
	backendPolicy := findNetworkPolicyRef(backend.NetworkPolicies, testutils.FixturePolicyWithGroups.Name)
	is.True(backendPolicy != nil)
	is.Equal(backendPolicy.Cidr, testutils.FixturePolicyWithGroups.CIDR)

	// frontend: 2 hosts, 1 user (alice only), 1 network policy (corp-vpn)
	frontend := findGroupWithUsers(resp.Groups, testutils.FixtureGroupFrontend.Name)
	is.True(frontend != nil)
	is.Equal(len(frontend.Hosts), 2) // FixtureHostFrontend1+2
	is.True(frontend.Users != nil)
	is.Equal(len(*frontend.Users), 1) // FixtureUserWithAccess only
	is.Equal(len(frontend.NetworkPolicies), 1)
	is.True(findNetworkPolicyRef(frontend.NetworkPolicies, testutils.FixturePolicyWithGroups.Name) != nil)

	// empty-group: no hosts, no users, no network policies
	emptyGroup := findGroupWithUsers(resp.Groups, testutils.FixtureGroupEmpty.Name)
	is.True(emptyGroup != nil)
	is.Equal(len(emptyGroup.Hosts), 0)
	is.True(emptyGroup.Users != nil)
	is.Equal(len(*emptyGroup.Users), 0)
	is.Equal(len(emptyGroup.NetworkPolicies), 0)
}

func TestHandler_ListHostGroups_Empty(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/access/host-groups", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)
	var resp httpapi.GroupListResponse
	is.NoErr(json.NewDecoder(w.Body).Decode(&resp))
	is.Equal(len(resp.Groups), 0)
}

func TestHandler_ListHostGroups_Unauthenticated(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/access/host-groups", nil)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusUnauthorized)
}

// ── ListHostSuggestions ───────────────────────────────────────────────────────

// TestHandler_ListHostSuggestions_HappyPath covers all three filter branches:
// unknown host (→ suggestion), known host (→ excluded), ignored host (→ ignored list).
func TestHandler_ListHostSuggestions_HappyPath(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	knownHost := "known-app.internal"
	suggestedHost := "new-app.internal"
	ignoredHost := "ignored-app.internal"

	testutils.NewSeeder(t, srv).
		WithHost(testutils.HostFixture{FQDN: knownHost}).
		WithAccessLogEntry(testutils.AccessLogEntryFixture{ClientIP: "9.9.9.9", Outcome: true, TargetHost: &knownHost}).
		WithAccessLogEntry(testutils.AccessLogEntryFixture{ClientIP: "9.9.9.9", Outcome: false, TargetHost: &suggestedHost}).
		WithAccessLogEntry(testutils.AccessLogEntryFixture{ClientIP: "9.9.9.9", Outcome: false, TargetHost: &ignoredHost}).
		Build()

	_, err := srv.HostsService.AddIgnoredSuggestion(t.Context(), ignoredHost)
	is.NoErr(err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/access/host-suggestions", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)
	var resp httpapi.HostSuggestionsPage
	is.NoErr(json.NewDecoder(w.Body).Decode(&resp))

	// only new-app.internal appears; known and ignored are filtered out
	is.Equal(len(resp.Suggestions), 1)
	is.Equal(resp.Suggestions[0].Fqdn, suggestedHost)
	is.Equal(resp.Suggestions[0].DeniedHits, 1)
	is.Equal(resp.Suggestions[0].AllowedHits, 0)

	is.Equal(len(resp.Ignored), 1)
	is.Equal(resp.Ignored[0].Fqdn, ignoredHost)
}

func TestHandler_ListHostSuggestions_Empty(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/access/host-suggestions", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)
	var resp httpapi.HostSuggestionsPage
	is.NoErr(json.NewDecoder(w.Body).Decode(&resp))
	is.Equal(len(resp.Suggestions), 0)
	is.Equal(len(resp.Ignored), 0)
}

func TestHandler_ListHostSuggestions_Unauthenticated(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/access/host-suggestions", nil)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusUnauthorized)
}

// ── ListUsersWithAccess ───────────────────────────────────────────────────────

// TestHandler_ListUsersWithAccess_HappyPath covers all user states in one response:
// user with group access (alice), user with bypass (charlie), and user with no access (bob).
func TestHandler_ListUsersWithAccess_HappyPath(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	seed := testutils.SeedFullWorld(t, srv).Build()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/access/users", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)
	var rows []httpapi.UserListItem
	is.NoErr(json.NewDecoder(w.Body).Decode(&rows))
	is.Equal(len(rows), 4) // admin + FixtureUserWithAccess + FixtureUserNoAccess + FixtureUserBypassAccess

	alice := findUserRow(rows, seed.User(testutils.FixtureUserWithAccess.Name).Int64())
	is.True(alice != nil)
	is.Equal(alice.BypassHostCheck, false)
	is.Equal(alice.HostCount, 4)        // FixtureGroupBackend(2) + FixtureGroupFrontend(2)
	is.Equal(alice.DeviceCount, 1)      // FixtureDeviceWithOwnerAccess
	is.Equal(alice.LiveAddressCount, 1) // FixtureAddressAlice
	is.Equal(len(alice.Groups), 2)      // FixtureGroupBackend + FixtureGroupFrontend

	bob := findUserRow(rows, seed.User(testutils.FixtureUserNoAccess.Name).Int64())
	is.True(bob != nil)
	is.Equal(bob.BypassHostCheck, false)
	is.Equal(bob.HostCount, 0)
	is.Equal(bob.DeviceCount, 1)      // FixtureDeviceWithoutOwnerAccess
	is.Equal(bob.LiveAddressCount, 1) // FixtureAddressBob
	is.Equal(len(bob.Groups), 0)

	charlie := findUserRow(rows, seed.User(testutils.FixtureUserBypassAccess.Name).Int64())
	is.True(charlie != nil)
	is.Equal(charlie.BypassHostCheck, true)
	is.Equal(charlie.HostCount, 4) // bypass = all 4 hosts
	is.Equal(charlie.DeviceCount, 1)
	is.Equal(len(charlie.Groups), 1) // FixtureGroupBackend only
}

func TestHandler_ListUsersWithAccess_OnlyAdminByDefault(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/access/users", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)
	var rows []httpapi.UserListItem
	is.NoErr(json.NewDecoder(w.Body).Decode(&rows))
	is.Equal(len(rows), 1)
	is.Equal(rows[0].BypassHostCheck, false)
	is.Equal(rows[0].HostCount, 0)
	is.Equal(len(rows[0].Groups), 0)
}

func TestHandler_ListUsersWithAccess_Unauthenticated(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/access/users", nil)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusUnauthorized)
}

// ── GetUserAccessDetail ───────────────────────────────────────────────────────

// TestHandler_GetUserAccessDetail_HappyPath covers all detail paths: granted groups
// with hosts and network policies, an ungranted group, and owned devices.
func TestHandler_GetUserAccessDetail_HappyPath(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	seed := testutils.SeedFullWorld(t, srv).Build()
	aliceID := seed.User(testutils.FixtureUserWithAccess.Name)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/admin/access/users/%d", aliceID), nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)
	var resp httpapi.UserAccessDetail
	is.NoErr(json.NewDecoder(w.Body).Decode(&resp))
	is.Equal(resp.Id, aliceID.Int64())
	is.Equal(resp.Username, testutils.FixtureUserWithAccess.Name)
	is.Equal(resp.BypassHostCheck, false)

	// all 3 groups are returned; only backend and frontend are granted
	is.Equal(len(resp.Groups), 3)

	backend := findGroup(resp.Groups, testutils.FixtureGroupBackend.Name)
	is.True(backend != nil)
	is.Equal(backend.Granted, true)
	is.Equal(len(backend.Hosts), 2) // FixtureHostBackend1+2
	is.Equal(len(backend.NetworkPolicies), 1)
	is.True(findNetworkPolicyRef(backend.NetworkPolicies, testutils.FixturePolicyWithGroups.Name) != nil)

	frontend := findGroup(resp.Groups, testutils.FixtureGroupFrontend.Name)
	is.True(frontend != nil)
	is.Equal(frontend.Granted, true)
	is.Equal(len(frontend.Hosts), 2) // FixtureHostFrontend1+2
	is.Equal(len(frontend.NetworkPolicies), 1)

	emptyGroup := findGroup(resp.Groups, testutils.FixtureGroupEmpty.Name)
	is.True(emptyGroup != nil)
	is.Equal(emptyGroup.Granted, false)
	is.Equal(len(emptyGroup.Hosts), 0)
	is.Equal(len(emptyGroup.NetworkPolicies), 0)

	// alice owns 1 device
	is.Equal(len(resp.Devices), 1)
	is.Equal(resp.Devices[0].Name, testutils.FixtureDeviceWithOwnerAccess.Name)
}

func TestHandler_GetUserAccessDetail_NotFound(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/access/users/99999", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusNotFound)
}

func TestHandler_GetUserAccessDetail_Unauthenticated(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/access/users/1", nil)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusUnauthorized)
}
