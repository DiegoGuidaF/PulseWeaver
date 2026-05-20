//go:build test

package queries_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

// ── URL helpers ───────────────────────────────────────────────────────────────

func networkPoliciesURL() string { return "/api/v1/admin/access/network-policies" }
func networkPolicyURL(id any) string {
	return fmt.Sprintf("/api/v1/admin/access/network-policies/%v", id)
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

	testutils.SeedFullWorld(t, srv).Build()

	req := httptest.NewRequest(http.MethodGet, networkPoliciesURL(), nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)
	var resp []httpapi.NetworkPolicyListItem
	is.NoErr(json.NewDecoder(w.Body).Decode(&resp))
	is.Equal(len(resp), 2) // FixturePolicyWithGroups + FixturePolicyNoGroups

	corpVPN := findPolicy(resp, testutils.FixturePolicyWithGroups.Name)
	is.True(corpVPN != nil)
	is.Equal(corpVPN.Cidr, testutils.FixturePolicyWithGroups.CIDR)
	is.True(corpVPN.Enabled)
	is.Equal(corpVPN.BypassHostCheck, false)
	is.Equal(len(corpVPN.Groups), 2) // FixtureGroupBackend + FixtureGroupFrontend
	is.Equal(corpVPN.HostCount, 4)   // FixtureHostBackend1+2 + FixtureHostFrontend1+2
	is.True(!time.Time(corpVPN.CreatedAt).IsZero())

	// FixturePolicyNoGroups has no group assignments — verifies empty state in the same response
	isolated := findPolicy(resp, testutils.FixturePolicyNoGroups.Name)
	is.True(isolated != nil)
	is.Equal(len(isolated.Groups), 0)
	is.Equal(isolated.HostCount, 0)
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

	seed := testutils.SeedFullWorld(t, srv).Build()

	req := httptest.NewRequest(http.MethodGet, networkPolicyURL(seed.Policy(testutils.FixturePolicyWithGroups.Name)), nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)
	var resp httpapi.NetworkPolicyDetail
	is.NoErr(json.NewDecoder(w.Body).Decode(&resp))
	is.Equal(resp.Name, testutils.FixturePolicyWithGroups.Name)
	is.Equal(resp.Cidr, testutils.FixturePolicyWithGroups.CIDR)
	is.True(resp.Description != nil)
	is.Equal(*resp.Description, testutils.FixturePolicyWithGroups.Desc)
	is.True(resp.Enabled)
	is.Equal(resp.BypassHostCheck, false)
	is.True(!time.Time(resp.CreatedAt).IsZero())
	is.True(!time.Time(resp.UpdatedAt).IsZero())
	// All 3 groups returned (FixtureGroupEmpty + FixtureGroupBackend + FixtureGroupFrontend); 2 are granted
	is.Equal(len(resp.Groups), 3)
	backend := findGroup(resp.Groups, testutils.FixtureGroupBackend.Name)
	is.True(backend != nil)
	is.True(backend.Granted)
	is.Equal(len(backend.Hosts), 2) // FixtureHostBackend1 + FixtureHostBackend2
	frontend := findGroup(resp.Groups, testutils.FixtureGroupFrontend.Name)
	is.True(frontend != nil)
	is.True(frontend.Granted)
	is.Equal(len(frontend.Hosts), 2) // FixtureHostFrontend1 + FixtureHostFrontend2
	emptyGroup := findGroup(resp.Groups, testutils.FixtureGroupEmpty.Name)
	is.True(emptyGroup != nil)
	is.Equal(emptyGroup.Granted, false)
	is.Equal(len(emptyGroup.Hosts), 0)
}

func findGroup(groups []httpapi.SubjectGroupDetail, name string) *httpapi.SubjectGroupDetail {
	for i := range groups {
		if groups[i].Name == name {
			return &groups[i]
		}
	}
	return nil
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
