//go:build test

package networkpolicies_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/app"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/networkpolicies"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

func seedPolicy(t *testing.T, srv *app.App, name, cidr string) networkpolicies.NetworkPolicy {
	t.Helper()
	p, err := srv.NetworkPoliciesService.CreatePolicy(t.Context(), name, cidr, nil)
	if err != nil {
		t.Fatalf("seedPolicy %q: %v", name, err)
	}
	return p
}

func policiesURL() string     { return "/api/v1/admin/access/network-policies" }
func policyURL(id any) string { return fmt.Sprintf("/api/v1/admin/access/network-policies/%v", id) }
func policyGrantsURL(id any) string {
	return fmt.Sprintf("/api/v1/admin/access/network-policies/%v/grants", id)
}

// ── CreateNetworkPolicy ───────────────────────────────────────────────────────

func TestHandler_CreateNetworkPolicy(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	body, _ := json.Marshal(map[string]any{"name": "home", "cidr": "192.168.1.5/24"})
	req := httptest.NewRequest(http.MethodPost, policiesURL(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusCreated)
	var resp httpapi.NetworkPolicyDetail
	is.NoErr(json.NewDecoder(w.Body).Decode(&resp))
	is.Equal(resp.Name, "home")
	is.Equal(resp.Cidr, "192.168.1.0/24") // host bits zeroed
	is.True(resp.Enabled)
}

func TestHandler_CreateNetworkPolicy_InvalidCIDR(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	body, _ := json.Marshal(map[string]any{"name": "bad", "cidr": "not-a-cidr"})
	req := httptest.NewRequest(http.MethodPost, policiesURL(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusBadRequest)
}

func TestHandler_CreateNetworkPolicy_DuplicateCIDR(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	seedPolicy(t, srv, "first", "10.0.0.0/8")

	body, _ := json.Marshal(map[string]any{"name": "second", "cidr": "10.0.0.0/8"})
	req := httptest.NewRequest(http.MethodPost, policiesURL(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusConflict)
}

func TestHandler_CreateNetworkPolicy_Unauthenticated(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)

	body, _ := json.Marshal(map[string]any{"name": "home", "cidr": "10.0.0.0/8"})
	req := httptest.NewRequest(http.MethodPost, policiesURL(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusUnauthorized)
}

// ── UpdateNetworkPolicy ───────────────────────────────────────────────────────

func TestHandler_UpdateNetworkPolicy(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	p := seedPolicy(t, srv, "original", "10.1.0.0/16")

	body, _ := json.Marshal(map[string]any{
		"name": "renamed", "cidr": "10.1.0.0/16", "enabled": false, "description": "",
	})
	req := httptest.NewRequest(http.MethodPut, policyURL(p.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusNoContent)
}

func TestHandler_UpdateNetworkPolicy_NotFound(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	body, _ := json.Marshal(map[string]any{
		"name": "ghost", "cidr": "10.2.0.0/16", "enabled": true, "description": "",
	})
	req := httptest.NewRequest(http.MethodPut, policyURL(99999), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusNotFound)
}

func TestHandler_UpdateNetworkPolicy_InvalidCIDR(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	p := seedPolicy(t, srv, "p", "10.3.0.0/16")

	body, _ := json.Marshal(map[string]any{
		"name": "p", "cidr": "bad-cidr", "enabled": true, "description": "",
	})
	req := httptest.NewRequest(http.MethodPut, policyURL(p.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusBadRequest)
}

func TestHandler_UpdateNetworkPolicy_DuplicateCIDR(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	seedPolicy(t, srv, "taken", "10.4.0.0/16")
	p := seedPolicy(t, srv, "target", "10.5.0.0/16")

	body, _ := json.Marshal(map[string]any{
		"name": "target", "cidr": "10.4.0.0/16", "enabled": true, "description": "",
	})
	req := httptest.NewRequest(http.MethodPut, policyURL(p.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusConflict)
}

// ── DeleteNetworkPolicy ───────────────────────────────────────────────────────

func TestHandler_DeleteNetworkPolicy(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	p := seedPolicy(t, srv, "to-delete", "10.6.0.0/16")

	req := httptest.NewRequest(http.MethodDelete, policyURL(p.ID), nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusNoContent)
}

func TestHandler_DeleteNetworkPolicy_NotFound(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	req := httptest.NewRequest(http.MethodDelete, policyURL(99999), nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusNotFound)
}

// ── UpdateNetworkPolicyAccess ─────────────────────────────────────────────────

func TestHandler_UpdateNetworkPolicyAccess(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	p := seedPolicy(t, srv, "access-target", "10.7.0.0/16")

	body, _ := json.Marshal(map[string]any{"bypass_host_check": true, "group_ids": []int{}})
	req := httptest.NewRequest(http.MethodPut, policyGrantsURL(p.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusNoContent)
}

func TestHandler_UpdateNetworkPolicyAccess_NotFound(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	body, _ := json.Marshal(map[string]any{"bypass_host_check": false, "group_ids": []int{}})
	req := httptest.NewRequest(http.MethodPut, policyGrantsURL(99999), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusNotFound)
}
