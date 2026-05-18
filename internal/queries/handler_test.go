//go:build test

package queries_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

// SimulatePolicyAccess is implemented in the policy package; these integration
// tests live here because they rely on the same SetupIntegrationServer harness
// used by the rest of the queries handler tests.

func TestHandler_SimulatePolicyAccess_Unauthenticated(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/policy-simulate?ip=1.2.3.4&host=example.com", nil)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.True(rec.Code == http.StatusUnauthorized || rec.Code == http.StatusForbidden)
}

func TestHandler_SimulatePolicyAccess_IPNotRegistered(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", testutils.TestAdminPassword)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/policy-simulate?ip=9.9.9.9&host=example.com", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var result httpapi.PolicySimulateResult
	err := json.NewDecoder(rec.Body).Decode(&result)
	is.NoErr(err)
	is.True(!result.Allowed)
	is.True(result.DenyReason != nil)
	is.Equal(string(*result.DenyReason), "ip_not_registered")
}

func TestHandler_SimulatePolicyAccess_HostNotAllowed(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", testutils.TestAdminPassword)

	dev, err := testServer.DeviceService.CreateDevice(t.Context(), testutils.AdminPrincipal(t, testServer), "sim-device", nil)
	is.NoErr(err)
	_, _, err = testServer.DeviceService.RegisterAddressActivity(t.Context(), dev.ID, "5.6.7.9", device.EventSourceManual)
	is.NoErr(err)
	is.NoErr(testServer.PolicyService.Initialize(t.Context()))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/policy-simulate?ip=5.6.7.9&host=denied.example.com", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var result httpapi.PolicySimulateResult
	err = json.NewDecoder(rec.Body).Decode(&result)
	is.NoErr(err)
	is.True(!result.Allowed)
	is.True(result.DenyReason != nil)
	is.Equal(string(*result.DenyReason), "host_not_allowed")
}

func TestHandler_SimulatePolicyAccess_AllowedBypass(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", testutils.TestAdminPassword)

	principal := testutils.AdminPrincipal(t, testServer)
	is.NoErr(testServer.UserAccessService.SetUserAccess(t.Context(), principal.UserID, true, nil))

	dev, err := testServer.DeviceService.CreateDevice(t.Context(), principal, "sim-bypass-device", nil)
	is.NoErr(err)
	_, _, err = testServer.DeviceService.RegisterAddressActivity(t.Context(), dev.ID, "5.6.7.10", device.EventSourceManual)
	is.NoErr(err)
	is.NoErr(testServer.PolicyService.Initialize(t.Context()))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/policy-simulate?ip=5.6.7.10&host=anything.example.com", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var result httpapi.PolicySimulateResult
	err = json.NewDecoder(rec.Body).Decode(&result)
	is.NoErr(err)
	is.True(result.Allowed)
	is.True(result.DenyReason == nil)
}

func TestHandler_SimulatePolicyAccess_DoesNotWriteAccessLog(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", testutils.TestAdminPassword)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/policy-simulate?ip=9.9.9.9&host=example.com", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)
	is.Equal(rec.Code, http.StatusOK)

	// Access log must remain empty.
	logReq := httptest.NewRequest(http.MethodGet, "/api/v1/access-log", nil)
	logReq.AddCookie(adminCookie)
	logRec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(logRec, logReq)
	is.Equal(logRec.Code, http.StatusOK)

	var logResp httpapi.AccessLogResponse
	err := json.NewDecoder(logRec.Body).Decode(&logResp)
	is.NoErr(err)
	is.Equal(logResp.Total, 0)
}
