//go:build test

package queries_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/app"
	"github.com/DiegoGuidaF/PulseWeaver/internal/hosts"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/networkpolicies"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

// ── URL helpers ───────────────────────────────────────────────────────────────

func networkPoliciesURL() string { return "/api/v1/admin/access/network-policies" }
func networkPolicyURL(id any) string {
	return fmt.Sprintf("/api/v1/admin/access/network-policies/%v", id)
}

// ── seed helpers ──────────────────────────────────────────────────────────────

func seedNetworkPolicy(t *testing.T, srv *app.App, name, cidr string) networkpolicies.NetworkPolicy {
	t.Helper()
	p, err := srv.NetworkPoliciesService.CreatePolicy(t.Context(), name, cidr, nil)
	if err != nil {
		t.Fatalf("seedNetworkPolicy %q: %v", name, err)
	}
	return p
}

func seedHostGroup(t *testing.T, srv *app.App, name string) ids.HostGroupID {
	t.Helper()
	if err := srv.HostsService.ReconcileHostGroups(t.Context(), hosts.ReconcileHostGroupsInput{
		Groups: []hosts.DesiredHostGroup{{Name: name, Color: "#000000", Icon: "server"}},
	}); err != nil {
		t.Fatalf("seedHostGroup %q: %v", name, err)
	}
	groups, err := srv.HostsService.ListHostGroups(t.Context())
	if err != nil {
		t.Fatalf("seedHostGroup ListHostGroups: %v", err)
	}
	for _, g := range groups {
		if g.Name == name {
			return g.ID
		}
	}
	t.Fatalf("seedHostGroup: group %q not found after reconcile", name)
	return 0
}

// ── ListNetworkPolicies ───────────────────────────────────────────────────────

func TestHandler_ListNetworkPolicies_EmptyList(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	req := httptest.NewRequest(http.MethodGet, networkPoliciesURL(), nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)
	var resp []httpapi.NetworkPolicyListItem
	is.NoErr(json.NewDecoder(w.Body).Decode(&resp))
	is.Equal(len(resp), 0)
}

func TestHandler_ListNetworkPolicies_ReturnsPolicySummary(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	seedNetworkPolicy(t, srv, "home", "192.168.1.0/24")

	req := httptest.NewRequest(http.MethodGet, networkPoliciesURL(), nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)
	var resp []httpapi.NetworkPolicyListItem
	is.NoErr(json.NewDecoder(w.Body).Decode(&resp))
	is.Equal(len(resp), 1)
	is.Equal(resp[0].Name, "home")
	is.Equal(resp[0].Cidr, "192.168.1.0/24")
	is.True(resp[0].Enabled)
}

func TestHandler_ListNetworkPolicies_Unauthenticated(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)

	req := httptest.NewRequest(http.MethodGet, networkPoliciesURL(), nil)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusUnauthorized)
}

// ── GetNetworkPolicy ──────────────────────────────────────────────────────────

func TestHandler_GetNetworkPolicy_ReturnsDetail(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	p := seedNetworkPolicy(t, srv, "vpn", "10.8.0.0/16")

	req := httptest.NewRequest(http.MethodGet, networkPolicyURL(p.ID), nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)
	var resp httpapi.NetworkPolicyDetail
	is.NoErr(json.NewDecoder(w.Body).Decode(&resp))
	is.Equal(resp.Name, "vpn")
	is.Equal(resp.Cidr, "10.8.0.0/16")
	is.True(resp.Enabled)
	is.Equal(len(resp.Groups), 0)
}

func TestHandler_GetNetworkPolicy_WithAssignedGroup_GroupAppearsGranted(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	p := seedNetworkPolicy(t, srv, "edge", "203.0.113.0/24")
	groupID := seedHostGroup(t, srv, "public-servers")

	err := srv.NetworkPoliciesService.SetHostAccess(t.Context(), p.ID, false, []ids.HostGroupID{groupID})
	is.NoErr(err)

	req := httptest.NewRequest(http.MethodGet, networkPolicyURL(p.ID), nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)
	var resp httpapi.NetworkPolicyDetail
	is.NoErr(json.NewDecoder(w.Body).Decode(&resp))
	is.Equal(len(resp.Groups), 1)
	is.Equal(resp.Groups[0].Name, "public-servers")
	is.True(resp.Groups[0].Granted)
}

func TestHandler_GetNetworkPolicy_NotFound(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	req := httptest.NewRequest(http.MethodGet, networkPolicyURL(99999), nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusNotFound)
}
