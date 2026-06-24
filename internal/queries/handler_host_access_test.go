//go:build test

package queries_test

import (
	"encoding/json"
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

	testutils.SeedFullWorld(t).Build(srv)

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
	is.Equal(api1.Groups[0].Name, testutils.GroupMedia.Name)

	api2 := findHost(resp.Hosts, testutils.FixtureHostBackend2.FQDN)
	is.True(api2 != nil)
	is.Equal(len(api2.Groups), 1)
	is.Equal(api2.Groups[0].Name, testutils.GroupMedia.Name)

	web1 := findHost(resp.Hosts, testutils.FixtureHostFrontend1.FQDN)
	is.True(web1 != nil)
	is.Equal(len(web1.Groups), 1)
	is.Equal(web1.Groups[0].Name, testutils.GroupProductivity.Name)

	web2 := findHost(resp.Hosts, testutils.FixtureHostFrontend2.FQDN)
	is.True(web2 != nil)
	is.Equal(len(web2.Groups), 1)
	is.Equal(web2.Groups[0].Name, testutils.GroupProductivity.Name)
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

	testutils.SeedFullWorld(t).Build(srv)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/access/host-groups", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)
	var resp httpapi.GroupListResponse
	is.NoErr(json.NewDecoder(w.Body).Decode(&resp))
	is.Equal(len(resp.Groups), 4) // GroupMedia, GroupProductivity, GroupInfrastructure, FixtureGroupAdversarial

	// backend: 2 hosts, 2 users (james+maria), 1 network policy (corp-vpn)
	backend := findGroupWithUsers(resp.Groups, testutils.GroupMedia.Name)
	is.True(backend != nil)
	is.Equal(len(backend.Hosts), 2) // FixtureHostBackend1+2
	is.True(backend.Users != nil)
	is.Equal(len(*backend.Users), 2) // FixtureUserWithAccess + FixtureUserBypassAccess
	is.Equal(len(backend.NetworkPolicies), 1)
	backendPolicy := findNetworkPolicyRef(backend.NetworkPolicies, testutils.FixturePolicyWithGroups.Name)
	is.True(backendPolicy != nil)
	is.Equal(backendPolicy.Cidr, testutils.FixturePolicyWithGroups.CIDR)

	// frontend: 2 hosts, 1 user (james only), 1 network policy (corp-vpn)
	frontend := findGroupWithUsers(resp.Groups, testutils.GroupProductivity.Name)
	is.True(frontend != nil)
	is.Equal(len(frontend.Hosts), 2) // FixtureHostFrontend1+2
	is.True(frontend.Users != nil)
	is.Equal(len(*frontend.Users), 1) // FixtureUserWithAccess only
	is.Equal(len(frontend.NetworkPolicies), 1)
	is.True(findNetworkPolicyRef(frontend.NetworkPolicies, testutils.FixturePolicyWithGroups.Name) != nil)

	// empty-group: no hosts, no users, no network policies
	emptyGroup := findGroupWithUsers(resp.Groups, testutils.GroupInfrastructure.Name)
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

// TestHandler_ListHostGroups_DeletedUserExcluded confirms a soft-deleted user
// granted to a group no longer appears in that group's user list.
func TestHandler_ListHostGroups_DeletedUserExcluded(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	groupName := "mixed"
	seed := testutils.NewSeeder(t).
		WithGroup(testutils.GroupFixture{Name: groupName}).
		WithUser(testutils.UserFixture{Name: "active-user"}).
		WithUser(testutils.UserFixture{Name: "deleted-user"}).
		SetUserAccess("active-user", false, groupName).
		SetUserAccess("deleted-user", false, groupName).
		Build(srv)

	admin := testutils.AdminPrincipal(t, srv)
	is.NoErr(srv.AuthService.DeleteUser(t.Context(), admin, seed.User("deleted-user")))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/access/host-groups", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)
	var resp httpapi.GroupListResponse
	is.NoErr(json.NewDecoder(w.Body).Decode(&resp))

	mixed := findGroupWithUsers(resp.Groups, groupName)
	is.True(mixed != nil)
	is.True(mixed.Users != nil)
	is.Equal(len(*mixed.Users), 1)
	is.Equal((*mixed.Users)[0].Username, "active-user")
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

	testutils.NewSeeder(t).
		WithHost(testutils.HostFixture{FQDN: knownHost}).
		WithAccessLogEntry(testutils.AccessLogEntryFixture{ClientIP: "9.9.9.9", Outcome: true, TargetHost: &knownHost}).
		WithAccessLogEntry(testutils.AccessLogEntryFixture{ClientIP: "9.9.9.9", Outcome: false, TargetHost: &suggestedHost}).
		WithAccessLogEntry(testutils.AccessLogEntryFixture{ClientIP: "9.9.9.9", Outcome: false, TargetHost: &ignoredHost}).
		Build(srv)

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

// TestHandler_ListHostSuggestions_PortNormalised proves that a host observed with
// a port suffix (the shape produced when a proxy is fronted on a non-default port)
// surfaces as a suggestion for its bare FQDN, aggregates with bare-host hits, and
// is excluded once the bare host is granted.
func TestHandler_ListHostSuggestions_PortNormalised(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	knownHost := "known-app.internal"
	knownHostPort := "known-app.internal:8443"
	suggestedHost := "new-app.internal"
	suggestedHostPort := "new-app.internal:8443"

	testutils.NewSeeder(t).
		WithHost(testutils.HostFixture{FQDN: knownHost}).
		// granted host seen with a port suffix must still be excluded from suggestions
		WithAccessLogEntry(testutils.AccessLogEntryFixture{ClientIP: "9.9.9.9", Outcome: true, TargetHost: &knownHostPort}).
		// same unknown host seen bare and with a port → one suggestion, hits summed
		WithAccessLogEntry(testutils.AccessLogEntryFixture{ClientIP: "9.9.9.9", Outcome: false, TargetHost: &suggestedHost}).
		WithAccessLogEntry(testutils.AccessLogEntryFixture{ClientIP: "9.9.9.9", Outcome: false, TargetHost: &suggestedHostPort}).
		Build(srv)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/access/host-suggestions", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)
	var resp httpapi.HostSuggestionsPage
	is.NoErr(json.NewDecoder(w.Body).Decode(&resp))

	is.Equal(len(resp.Suggestions), 1)
	is.Equal(resp.Suggestions[0].Fqdn, suggestedHost) // bare FQDN, port stripped
	is.Equal(resp.Suggestions[0].DeniedHits, 2)       // bare + port-suffixed hits merged
	is.Equal(resp.Suggestions[0].AllowedHits, 0)
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
