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

func TestHandler_GetPolicyUserMap_Unauthenticated(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/policy-map", nil)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.True(rec.Code == http.StatusUnauthorized || rec.Code == http.StatusForbidden)
}

func TestHandler_GetPolicyUserMap_EmptyCache(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", testutils.TestAdminPassword)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/policy-map", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var response httpapi.PolicyUserMapAudit
	err := json.NewDecoder(rec.Body).Decode(&response)
	is.NoErr(err)
	// Empty cache: admin user should appear as a no-access user.
	is.True(len(response.Users) >= 1)
	is.True(response.RefreshDurationMs >= 0)
	// All users should have empty ips (no cache entries).
	for _, u := range response.Users {
		is.Equal(len(u.Ips), 0)
		is.True(u.LastSeenAt == nil)
	}
}

func TestHandler_GetPolicyUserMap_WithEntry(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", testutils.TestAdminPassword)

	dev, err := testServer.DeviceService.CreateDevice(t.Context(), testutils.AdminPrincipal(t, testServer), "policyusermap-device", nil)
	is.NoErr(err)
	_, _, err = testServer.DeviceService.RegisterAddressActivity(t.Context(), dev.ID, "5.6.7.8", device.EventSourceManual)
	is.NoErr(err)
	is.NoErr(testServer.PolicyService.Initialize(t.Context()))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/policy-map", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var response httpapi.PolicyUserMapAudit
	err = json.NewDecoder(rec.Body).Decode(&response)
	is.NoErr(err)
	is.True(len(response.Users) >= 1)

	// Find the admin user entry (owner of the device).
	var adminEntry *httpapi.PolicyUserEntry
	adminID := testutils.AdminPrincipal(t, testServer).UserID
	for i := range response.Users {
		if response.Users[i].UserId == int64(adminID) {
			adminEntry = &response.Users[i]
			break
		}
	}
	is.True(adminEntry != nil)
	is.Equal(len(adminEntry.Ips), 1)
	is.Equal(adminEntry.Ips[0].Ip, "5.6.7.8")
	is.Equal(len(adminEntry.Ips[0].Addresses), 1)
	is.Equal(adminEntry.DeviceCount, 1)
	is.Equal(adminEntry.IpCount, 1)
}
