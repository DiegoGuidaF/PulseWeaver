//go:build test

package registration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/registration"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

func defaultCreateInviteRequest() registration.CreateInviteRequest {
	return registration.CreateInviteRequest{
		DeviceName:         "Dad's Phone",
		OwnerID:            auth.UserID(1),
		HeartbeatServerURL: "https://pulse.home.lan",
		IntervalSeconds:    900,
		ExpiresInHours:     24,
	}
}

func claimInvite(t *testing.T, server http.Handler, code string) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(map[string]string{"code": code})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	return w
}

func TestHandler_CreateRegistration_AdminCreatesInvite(t *testing.T) {
	is := is.New(t)
	ts := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, ts.HTTPServer, "admin", testutils.TestAdminPassword)

	body, _ := json.Marshal(map[string]any{
		"device_name":          "Dad's Phone",
		"heartbeat_server_url": "https://pulse.home.lan",
		"interval_seconds":     900,
		"expires_in_hours":     24,
		"owner_id":             1,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/registrations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	ts.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusCreated)
	var resp httpapi.PendingRegistration
	is.NoErr(json.NewDecoder(w.Body).Decode(&resp))
	is.Equal(resp.DeviceName, "Dad's Phone")
	is.True(resp.RegistrationCode != nil && *resp.RegistrationCode != "")
	is.Equal(resp.Status, httpapi.PendingRegistrationStatusPending)
	// device_api_key must never be in the response
	raw, _ := json.Marshal(resp)
	is.True(!containsKey(string(raw), "device_api_key"))
}

func TestHandler_CreateRegistration_RequiresAdmin(t *testing.T) {
	is := is.New(t)
	ts := testutils.SetupIntegrationServer(t)

	body, _ := json.Marshal(map[string]any{
		"device_name":          "Dad's Phone",
		"heartbeat_server_url": "https://pulse.home.lan",
		"interval_seconds":     900,
		"expires_in_hours":     24,
		"owner_id":             1,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/registrations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.HTTPServer.ServeHTTP(w, req)
	is.True(w.Code == http.StatusUnauthorized || w.Code == http.StatusForbidden)
}

func TestHandler_ListRegistrations_DefaultPendingOnly(t *testing.T) {
	is := is.New(t)
	ts := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, ts.HTTPServer, "admin", testutils.TestAdminPassword)

	// Given — two pending invites seeded via service
	_, err := ts.RegistrationService.CreateInvite(context.Background(), defaultCreateInviteRequest())
	is.NoErr(err)
	_, err = ts.RegistrationService.CreateInvite(context.Background(), registration.CreateInviteRequest{
		DeviceName:         "Mom's Phone",
		OwnerID:            auth.UserID(1),
		HeartbeatServerURL: "https://pulse.home.lan",
		IntervalSeconds:    300,
		ExpiresInHours:     1,
	})
	is.NoErr(err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/registrations", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	ts.HTTPServer.ServeHTTP(w, req)
	is.Equal(w.Code, http.StatusOK)

	var resp []httpapi.PendingRegistration
	is.NoErr(json.NewDecoder(w.Body).Decode(&resp))
	is.Equal(len(resp), 2)
}

func TestHandler_GetRegistration_ReturnsInviteWithCode(t *testing.T) {
	is := is.New(t)
	ts := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, ts.HTTPServer, "admin", testutils.TestAdminPassword)

	// Given
	invite, err := ts.RegistrationService.CreateInvite(context.Background(), defaultCreateInviteRequest())
	is.NoErr(err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/registrations/"+strconv.FormatInt(invite.ID.Int64(), 10), nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	ts.HTTPServer.ServeHTTP(w, req)
	is.Equal(w.Code, http.StatusOK)

	var fetched httpapi.PendingRegistration
	is.NoErr(json.NewDecoder(w.Body).Decode(&fetched))
	is.Equal(fetched.Id, invite.ID.Int64())
	is.True(fetched.RegistrationCode != nil && *fetched.RegistrationCode == *invite.RegistrationCode)
}

func TestHandler_DeleteRegistration_InvalidatesPendingInvite(t *testing.T) {
	is := is.New(t)
	ts := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, ts.HTTPServer, "admin", testutils.TestAdminPassword)

	// Given
	invite, err := ts.RegistrationService.CreateInvite(context.Background(), defaultCreateInviteRequest())
	is.NoErr(err)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/registrations/"+strconv.FormatInt(invite.ID.Int64(), 10), nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	ts.HTTPServer.ServeHTTP(w, req)
	is.Equal(w.Code, http.StatusNoContent)

	// Soft delete: the invite still exists but its status is now "invalidated"
	fetched, err := ts.RegistrationService.GetInvite(context.Background(), invite.ID)
	is.NoErr(err)
	is.Equal(fetched.Status(), registration.StatusInvalidated)
}

func TestHandler_ClaimRegistration_SuccessfulClaim(t *testing.T) {
	is := is.New(t)
	ts := testutils.SetupIntegrationServer(t)

	// Given
	invite, err := ts.RegistrationService.CreateInvite(context.Background(), defaultCreateInviteRequest())
	is.NoErr(err)

	// When
	w := claimInvite(t, ts.HTTPServer, *invite.RegistrationCode)
	is.Equal(w.Code, http.StatusOK)

	var result httpapi.ClaimRegistrationResponse
	is.NoErr(json.NewDecoder(w.Body).Decode(&result))
	is.Equal(result.ServerUrl, "https://pulse.home.lan")
	is.Equal(result.IntervalSeconds, 900)
	is.True(result.ApiKey != "")

	// Then — verify side effects not visible in the claim response
	fetched, err := ts.RegistrationService.GetInvite(context.Background(), invite.ID)
	is.NoErr(err)
	is.Equal(fetched.Status(), registration.StatusUsed)
	is.True(fetched.RegistrationCode == nil)
	is.True(fetched.CreatedDeviceID != nil)
}

func TestHandler_ClaimRegistration_UnknownCodeReturns404(t *testing.T) {
	is := is.New(t)
	ts := testutils.SetupIntegrationServer(t)

	w := claimInvite(t, ts.HTTPServer, "totallyinvalidcode")
	is.Equal(w.Code, http.StatusNotFound)
}

func TestHandler_ClaimRegistration_CodeUsedTwiceReturns404(t *testing.T) {
	is := is.New(t)
	ts := testutils.SetupIntegrationServer(t)

	// Given — invite already claimed
	invite, err := ts.RegistrationService.CreateInvite(context.Background(), defaultCreateInviteRequest())
	is.NoErr(err)
	_, err = ts.RegistrationService.ClaimInvite(context.Background(), *invite.RegistrationCode)
	is.NoErr(err)

	// When — try to claim the same code again
	w := claimInvite(t, ts.HTTPServer, *invite.RegistrationCode)
	is.Equal(w.Code, http.StatusNotFound)
}

func TestHandler_ClaimRegistration_InvalidatedCodeReturns404(t *testing.T) {
	is := is.New(t)
	ts := testutils.SetupIntegrationServer(t)

	// Given — invite invalidated by admin
	invite, err := ts.RegistrationService.CreateInvite(context.Background(), defaultCreateInviteRequest())
	is.NoErr(err)
	err = ts.RegistrationService.InvalidateInvite(context.Background(), invite.ID)
	is.NoErr(err)

	// When — try to claim an invalidated code
	w := claimInvite(t, ts.HTTPServer, *invite.RegistrationCode)
	is.Equal(w.Code, http.StatusNotFound)
}

func TestHandler_ClaimRegistration_CreatedDeviceHasCorrectAPIKeyPrefix(t *testing.T) {
	is := is.New(t)
	ts := testutils.SetupIntegrationServer(t)

	// Given
	invite, err := ts.RegistrationService.CreateInvite(context.Background(), defaultCreateInviteRequest())
	is.NoErr(err)

	// When
	w := claimInvite(t, ts.HTTPServer, *invite.RegistrationCode)
	is.Equal(w.Code, http.StatusOK)

	var result httpapi.ClaimRegistrationResponse
	is.NoErr(json.NewDecoder(w.Body).Decode(&result))
	is.True(len(result.ApiKey) > len(device.APIKeyPrefix))
	is.Equal(result.ApiKey[:len(device.APIKeyPrefix)], device.APIKeyPrefix)
}

// containsKey checks whether a JSON string contains a top-level key.
func containsKey(jsonStr, key string) bool {
	var m map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		return false
	}
	_, ok := m[key]
	return ok
}
