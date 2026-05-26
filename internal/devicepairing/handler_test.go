//go:build test

package devicepairing_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/app"
	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/devicepairing"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

// createSeedDevice creates a device for use in pairing tests and returns its ID.
func createSeedDevice(t *testing.T, ts *app.App) ids.DeviceID {
	t.Helper()
	dev, err := ts.DeviceService.CreateDevice(context.Background(), &auth.Principal{UserID: ids.UserID(1)}, "test-device", nil)
	if err != nil {
		t.Fatalf("createSeedDevice: %v", err)
	}
	return dev.ID
}

func defaultCreatePairingRequest(deviceID ids.DeviceID) devicepairing.CreatePairingRequest {
	return devicepairing.CreatePairingRequest{
		DeviceID:           deviceID,
		HeartbeatServerURL: "https://pulse.home.lan",
		IntervalSeconds:    900,
		ExpiresInHours:     24,
	}
}

func pairingsURL(deviceID ids.DeviceID) string {
	return fmt.Sprintf("/api/v1/devices/%d/pairings", deviceID)
}

func pairingByIDURL(deviceID ids.DeviceID, pairingID int64) string {
	return fmt.Sprintf("/api/v1/devices/%d/pairings/%d", deviceID, pairingID)
}

func claimPairing(t *testing.T, server http.Handler, code string) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(map[string]string{"code": code})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/device-pair", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	return w
}

func TestHandler_CreateDevicePairing_AdminCreatesPairing(t *testing.T) {
	is := is.New(t)
	ts := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, ts.HTTPServer, "admin", testutils.TestAdminPassword)
	devID := createSeedDevice(t, ts)

	body, _ := json.Marshal(map[string]any{
		"heartbeat_server_url": "https://pulse.home.lan",
		"interval_seconds":     900,
		"expires_in_hours":     24,
	})
	req := httptest.NewRequest(http.MethodPost, pairingsURL(devID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	ts.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusCreated)
	var resp httpapi.DevicePairing
	is.NoErr(json.NewDecoder(w.Body).Decode(&resp))
	is.Equal(resp.DeviceId, devID.Int64())
	is.True(resp.PairingCode != "")
	is.Equal(resp.Status, httpapi.DevicePairingStatusPending)
}

func TestHandler_CreateDevicePairing_RequiresAdmin(t *testing.T) {
	is := is.New(t)
	ts := testutils.SetupIntegrationServer(t)
	devID := createSeedDevice(t, ts)

	body, _ := json.Marshal(map[string]any{
		"heartbeat_server_url": "https://pulse.home.lan",
		"interval_seconds":     900,
		"expires_in_hours":     24,
	})
	req := httptest.NewRequest(http.MethodPost, pairingsURL(devID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.HTTPServer.ServeHTTP(w, req)
	is.True(w.Code == http.StatusUnauthorized || w.Code == http.StatusForbidden)
}

func TestHandler_ListDevicePairings_DefaultPendingOnly(t *testing.T) {
	is := is.New(t)
	ts := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, ts.HTTPServer, "admin", testutils.TestAdminPassword)
	devID := createSeedDevice(t, ts)

	// Second CreatePairing replaces the first, so only 1 pending remains.
	_, err := ts.DevicePairingService.CreatePairing(context.Background(), defaultCreatePairingRequest(devID))
	is.NoErr(err)
	_, err = ts.DevicePairingService.CreatePairing(context.Background(), devicepairing.CreatePairingRequest{
		DeviceID:           devID,
		HeartbeatServerURL: "https://pulse.home.lan",
		IntervalSeconds:    300,
		ExpiresInHours:     1,
	})
	is.NoErr(err)

	req := httptest.NewRequest(http.MethodGet, pairingsURL(devID), nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	ts.HTTPServer.ServeHTTP(w, req)
	is.Equal(w.Code, http.StatusOK)

	var resp []httpapi.DevicePairing
	is.NoErr(json.NewDecoder(w.Body).Decode(&resp))
	is.Equal(len(resp), 1)
}

func TestHandler_GetDevicePairing_ReturnsWithCode(t *testing.T) {
	is := is.New(t)
	ts := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, ts.HTTPServer, "admin", testutils.TestAdminPassword)
	devID := createSeedDevice(t, ts)

	pairing, err := ts.DevicePairingService.CreatePairing(context.Background(), defaultCreatePairingRequest(devID))
	is.NoErr(err)

	req := httptest.NewRequest(http.MethodGet, pairingByIDURL(devID, pairing.ID.Int64()), nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	ts.HTTPServer.ServeHTTP(w, req)
	is.Equal(w.Code, http.StatusOK)

	var fetched httpapi.DevicePairing
	is.NoErr(json.NewDecoder(w.Body).Decode(&fetched))
	is.Equal(fetched.Id, pairing.ID.Int64())
	is.True(fetched.PairingCode == pairing.PairingCode)
}

func TestHandler_DeleteDevicePairing_InvalidatesPendingPairing(t *testing.T) {
	is := is.New(t)
	ts := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, ts.HTTPServer, "admin", testutils.TestAdminPassword)
	devID := createSeedDevice(t, ts)

	pairing, err := ts.DevicePairingService.CreatePairing(context.Background(), defaultCreatePairingRequest(devID))
	is.NoErr(err)

	req := httptest.NewRequest(http.MethodDelete, pairingByIDURL(devID, pairing.ID.Int64()), nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	ts.HTTPServer.ServeHTTP(w, req)
	is.Equal(w.Code, http.StatusNoContent)

	fetched, err := ts.DevicePairingService.GetPairing(context.Background(), pairing.ID)
	is.NoErr(err)
	is.Equal(fetched.Status, devicepairing.StatusInvalidated)
}

func TestHandler_ClaimPairing_SuccessfulClaim(t *testing.T) {
	is := is.New(t)
	ts := testutils.SetupIntegrationServer(t)
	devID := createSeedDevice(t, ts)

	pairing, err := ts.DevicePairingService.CreatePairing(context.Background(), defaultCreatePairingRequest(devID))
	is.NoErr(err)

	w := claimPairing(t, ts.HTTPServer, pairing.PairingCode)
	is.Equal(w.Code, http.StatusOK)

	var result httpapi.ClaimPairingResponse
	is.NoErr(json.NewDecoder(w.Body).Decode(&result))
	is.Equal(result.ServerUrl, "https://pulse.home.lan")
	is.Equal(result.IntervalSeconds, 900)
	is.True(result.ApiKey != "")

	fetched, err := ts.DevicePairingService.GetPairing(context.Background(), pairing.ID)
	is.NoErr(err)
	is.Equal(fetched.Status, devicepairing.StatusUsed)
}

func TestHandler_ClaimPairing_UnknownCodeReturns404(t *testing.T) {
	is := is.New(t)
	ts := testutils.SetupIntegrationServer(t)

	w := claimPairing(t, ts.HTTPServer, "totallyinvalidcode")
	is.Equal(w.Code, http.StatusNotFound)
}

func TestHandler_ClaimPairing_CodeUsedTwiceReturns404(t *testing.T) {
	is := is.New(t)
	ts := testutils.SetupIntegrationServer(t)
	devID := createSeedDevice(t, ts)

	pairing, err := ts.DevicePairingService.CreatePairing(context.Background(), defaultCreatePairingRequest(devID))
	is.NoErr(err)
	_, err = ts.DevicePairingService.ClaimPairing(context.Background(), pairing.PairingCode)
	is.NoErr(err)

	w := claimPairing(t, ts.HTTPServer, pairing.PairingCode)
	is.Equal(w.Code, http.StatusNotFound)
}

func TestHandler_ClaimPairing_InvalidatedCodeReturns404(t *testing.T) {
	is := is.New(t)
	ts := testutils.SetupIntegrationServer(t)
	devID := createSeedDevice(t, ts)

	pairing, err := ts.DevicePairingService.CreatePairing(context.Background(), defaultCreatePairingRequest(devID))
	is.NoErr(err)
	err = ts.DevicePairingService.InvalidatePairing(context.Background(), devID, pairing.ID)
	is.NoErr(err)

	w := claimPairing(t, ts.HTTPServer, pairing.PairingCode)
	is.Equal(w.Code, http.StatusNotFound)
}

func TestHandler_ClaimPairing_RegeneratedKeyHasCorrectPrefix(t *testing.T) {
	is := is.New(t)
	ts := testutils.SetupIntegrationServer(t)
	devID := createSeedDevice(t, ts)

	pairing, err := ts.DevicePairingService.CreatePairing(context.Background(), defaultCreatePairingRequest(devID))
	is.NoErr(err)

	w := claimPairing(t, ts.HTTPServer, pairing.PairingCode)
	is.Equal(w.Code, http.StatusOK)

	var result httpapi.ClaimPairingResponse
	is.NoErr(json.NewDecoder(w.Body).Decode(&result))
	is.True(len(result.ApiKey) > len(device.APIKeyPrefix))
	is.Equal(result.ApiKey[:len(device.APIKeyPrefix)], device.APIKeyPrefix)
}
