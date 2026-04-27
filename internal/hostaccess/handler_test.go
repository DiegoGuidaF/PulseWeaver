//go:build test

package hostaccess_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

func TestHandler_ReconcileKnownHosts(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	body, _ := json.Marshal(map[string]any{
		"hosts": []map[string]any{{"fqdn": "router.example.com", "group_ids": []int{}}},
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/hosts/reconcile", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	res := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusNoContent)
}

func TestHandler_ReconcileHostGroups(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	body, _ := json.Marshal(map[string]any{
		"groups": []map[string]any{{"name": "infra"}},
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/host-groups/reconcile", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	res := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusNoContent)
}

func TestHandler_SetUserHostGrants(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	adminID := testutils.AdminPrincipal(t, srv).UserID
	url := fmt.Sprintf("/api/v1/admin/users/%d/host-grants", adminID)

	bypass := false
	body, _ := json.Marshal(map[string]any{"bypass": bypass})
	req := httptest.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	res := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusNoContent)
}

func TestHandler_IgnoreSuggestion(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	body, _ := json.Marshal(map[string]string{"fqdn": "ignored.example.com"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/host-suggestions/ignore", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	res := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusCreated)
	var resp httpapi.IgnoredHostSuggestion
	is.NoErr(json.NewDecoder(res.Body).Decode(&resp))
	is.Equal(resp.Fqdn, "ignored.example.com")
	is.True(resp.Id != 0)
}

func TestHandler_UnignoreSuggestion(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	_, err := srv.HostAccessService.AddIgnoredSuggestion(t.Context(), "ignored.example.com")
	is.NoErr(err)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/host-suggestions/ignore/ignored.example.com", nil)
	req.AddCookie(cookie)
	res := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusNoContent)
}
