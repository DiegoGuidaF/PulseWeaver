//go:build test

package registration_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/registration"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

func createInvite(t *testing.T, server http.Handler, cookie *http.Cookie, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/registrations", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	return w
}

func defaultInviteBody() map[string]any {
	return map[string]any{
		"device_name":          "Dad's Phone",
		"heartbeat_server_url": "https://pulse.home.lan",
		"interval_seconds":     900,
		"expires_in_hours":     24,
	}
}

func TestHandler_CreateRegistration_AdminCreatesInvite(t *testing.T) {
	is := is.New(t)
	ts := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, ts.HTTPServer, "admin", testutils.TestAdminPassword)

	w := createInvite(t, ts.HTTPServer, cookie, defaultInviteBody())
	is.Equal(w.Code, http.StatusCreated)

	var resp httpapi.PendingRegistration
	is.NoErr(json.NewDecoder(w.Body).Decode(&resp))
	is.Equal(resp.DeviceName, "Dad's Phone")
	is.True(resp.RegistrationCode != nil && *resp.RegistrationCode != "")
	is.Equal(resp.Status, httpapi.PendingRegistrationStatusPending)
	// device_api_key must never be in the response
	raw := w.Body.String()
	// Re-read since Decode consumed it
	b, _ := json.Marshal(resp)
	raw = string(b)
	is.True(!containsKey(raw, "device_api_key"))
}

func TestHandler_CreateRegistration_RequiresAdmin(t *testing.T) {
	is := is.New(t)
	ts := testutils.SetupIntegrationServer(t)
	// No cookie at all
	b, _ := json.Marshal(defaultInviteBody())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/registrations", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.HTTPServer.ServeHTTP(w, req)
	is.True(w.Code == http.StatusUnauthorized || w.Code == http.StatusForbidden)
}

func TestHandler_ListRegistrations_DefaultPendingOnly(t *testing.T) {
	is := is.New(t)
	ts := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, ts.HTTPServer, "admin", testutils.TestAdminPassword)

	// Create two invites
	createInvite(t, ts.HTTPServer, cookie, defaultInviteBody())
	createInvite(t, ts.HTTPServer, cookie, map[string]any{
		"device_name":          "Mom's Phone",
		"heartbeat_server_url": "https://pulse.home.lan",
		"interval_seconds":     300,
		"expires_in_hours":     1,
	})

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

	w := createInvite(t, ts.HTTPServer, cookie, defaultInviteBody())
	is.Equal(w.Code, http.StatusCreated)

	var created httpapi.PendingRegistration
	is.NoErr(json.NewDecoder(w.Body).Decode(&created))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/registrations/"+created.Id, nil)
	req.AddCookie(cookie)
	w2 := httptest.NewRecorder()
	ts.HTTPServer.ServeHTTP(w2, req)
	is.Equal(w2.Code, http.StatusOK)

	var fetched httpapi.PendingRegistration
	is.NoErr(json.NewDecoder(w2.Body).Decode(&fetched))
	is.Equal(fetched.Id, created.Id)
	is.True(fetched.RegistrationCode != nil && *fetched.RegistrationCode == *created.RegistrationCode)
}

func TestHandler_DeleteRegistration_InvalidatesPendingInvite(t *testing.T) {
	is := is.New(t)
	ts := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, ts.HTTPServer, "admin", testutils.TestAdminPassword)

	w := createInvite(t, ts.HTTPServer, cookie, defaultInviteBody())
	is.Equal(w.Code, http.StatusCreated)

	var created httpapi.PendingRegistration
	is.NoErr(json.NewDecoder(w.Body).Decode(&created))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/registrations/"+created.Id, nil)
	req.AddCookie(cookie)
	w2 := httptest.NewRecorder()
	ts.HTTPServer.ServeHTTP(w2, req)
	is.Equal(w2.Code, http.StatusNoContent)

	// Verify it's gone
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/admin/registrations/"+created.Id, nil)
	req2.AddCookie(cookie)
	w3 := httptest.NewRecorder()
	ts.HTTPServer.ServeHTTP(w3, req2)
	is.Equal(w3.Code, http.StatusNotFound)
}

func TestHandler_ClaimRegistration_SuccessfulClaim(t *testing.T) {
	is := is.New(t)
	ts := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, ts.HTTPServer, "admin", testutils.TestAdminPassword)

	// Create invite
	w := createInvite(t, ts.HTTPServer, cookie, defaultInviteBody())
	is.Equal(w.Code, http.StatusCreated)
	var created httpapi.PendingRegistration
	is.NoErr(json.NewDecoder(w.Body).Decode(&created))

	// Claim it
	b, _ := json.Marshal(map[string]string{"code": *created.RegistrationCode})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	ts.HTTPServer.ServeHTTP(w2, req)
	is.Equal(w2.Code, http.StatusOK)

	var result httpapi.ClaimRegistrationResponse
	is.NoErr(json.NewDecoder(w2.Body).Decode(&result))
	is.Equal(result.ServerUrl, "https://pulse.home.lan")
	is.Equal(result.IntervalSeconds, 900)
	is.True(result.ApiKey != "")

	// Verify invite now shows as used
	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/admin/registrations/"+created.Id, nil)
	req3.AddCookie(cookie)
	w3 := httptest.NewRecorder()
	ts.HTTPServer.ServeHTTP(w3, req3)
	var updated httpapi.PendingRegistration
	is.NoErr(json.NewDecoder(w3.Body).Decode(&updated))
	is.Equal(updated.Status, httpapi.PendingRegistrationStatusUsed)
	is.True(updated.RegistrationCode == nil)
	is.True(updated.CreatedDeviceId != nil)
}

func TestHandler_ClaimRegistration_UnknownCodeReturns404(t *testing.T) {
	is := is.New(t)
	ts := testutils.SetupIntegrationServer(t)

	b, _ := json.Marshal(map[string]string{"code": "totallyinvalidcode"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.HTTPServer.ServeHTTP(w, req)
	is.Equal(w.Code, http.StatusNotFound)
}

func TestHandler_ClaimRegistration_CodeUsedTwiceReturns404(t *testing.T) {
	is := is.New(t)
	ts := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, ts.HTTPServer, "admin", testutils.TestAdminPassword)

	w := createInvite(t, ts.HTTPServer, cookie, defaultInviteBody())
	is.Equal(w.Code, http.StatusCreated)
	var created httpapi.PendingRegistration
	is.NoErr(json.NewDecoder(w.Body).Decode(&created))

	claim := func() *httptest.ResponseRecorder {
		b, _ := json.Marshal(map[string]string{"code": *created.RegistrationCode})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		ts.HTTPServer.ServeHTTP(w, req)
		return w
	}

	is.Equal(claim().Code, http.StatusOK)
	is.Equal(claim().Code, http.StatusNotFound)
}

func TestHandler_ClaimRegistration_InvalidatedCodeReturns404(t *testing.T) {
	is := is.New(t)
	ts := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, ts.HTTPServer, "admin", testutils.TestAdminPassword)

	w := createInvite(t, ts.HTTPServer, cookie, defaultInviteBody())
	var created httpapi.PendingRegistration
	json.NewDecoder(w.Body).Decode(&created) //nolint:errcheck

	// Invalidate
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/registrations/"+created.Id, nil)
	req.AddCookie(cookie)
	ts.HTTPServer.ServeHTTP(httptest.NewRecorder(), req)

	// Try to claim
	b, _ := json.Marshal(map[string]string{"code": *created.RegistrationCode})
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewReader(b))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	ts.HTTPServer.ServeHTTP(w2, req2)
	is.Equal(w2.Code, http.StatusNotFound)
}

func TestHandler_ClaimRegistration_CreatedDeviceHasCorrectAPIKeyPrefix(t *testing.T) {
	is := is.New(t)
	ts := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, ts.HTTPServer, "admin", testutils.TestAdminPassword)

	w := createInvite(t, ts.HTTPServer, cookie, defaultInviteBody())
	var created httpapi.PendingRegistration
	is.NoErr(json.NewDecoder(w.Body).Decode(&created))

	b, _ := json.Marshal(map[string]string{"code": *created.RegistrationCode})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	ts.HTTPServer.ServeHTTP(w2, req)
	var result httpapi.ClaimRegistrationResponse
	is.NoErr(json.NewDecoder(w2.Body).Decode(&result))

	// The returned API key should start with the expected prefix
	is.True(len(result.ApiKey) > len(registration.APIKeyPrefixForTest))
	is.Equal(result.ApiKey[:len(registration.APIKeyPrefixForTest)], registration.APIKeyPrefixForTest)
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
