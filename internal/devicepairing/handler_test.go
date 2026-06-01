//go:build test

package devicepairing_test

import (
	"context"
	"net/http"
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

func TestHandler_CreateDevicePairing_AdminCreatesPairing(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	ts := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, ts)
	devID := createSeedDevice(t, ts)

	resp, err := client.CreateDevicePairingWithResponse(ctx, devID.Int64(), httpapi.CreateDevicePairingJSONRequestBody{
		HeartbeatServerUrl: "https://pulse.home.lan",
		IntervalSeconds:    900,
		ExpiresInHours:     24,
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusCreated)
	is.Equal(resp.JSON201.DeviceId, devID.Int64())
	is.True(resp.JSON201.PairingCode != "")
	is.Equal(resp.JSON201.Status, httpapi.DevicePairingStatusPending)
}

func TestHandler_CreateDevicePairing_RequiresAdmin(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	ts := testutils.SetupIntegrationServer(t)
	devID := createSeedDevice(t, ts)

	resp, err := testutils.NewAPIClient(t, ts).CreateDevicePairingWithResponse(ctx, devID.Int64(), httpapi.CreateDevicePairingJSONRequestBody{
		HeartbeatServerUrl: "https://pulse.home.lan",
		IntervalSeconds:    900,
		ExpiresInHours:     24,
	})
	is.NoErr(err)
	is.True(resp.StatusCode() == http.StatusUnauthorized || resp.StatusCode() == http.StatusForbidden)
}

func TestHandler_ListDevicePairings_DefaultPendingOnly(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	ts := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, ts)
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

	resp, err := client.ListDevicePairingsWithResponse(ctx, devID.Int64(), nil)
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	is.Equal(len(*resp.JSON200), 1)
}

func TestHandler_GetDevicePairing_ReturnsWithCode(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	ts := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, ts)
	devID := createSeedDevice(t, ts)

	pairing, err := ts.DevicePairingService.CreatePairing(context.Background(), defaultCreatePairingRequest(devID))
	is.NoErr(err)

	resp, err := client.GetDevicePairingWithResponse(ctx, devID.Int64(), pairing.ID.Int64())
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	is.Equal(resp.JSON200.Id, pairing.ID.Int64())
	is.True(resp.JSON200.PairingCode == pairing.PairingCode)
}

func TestHandler_DeleteDevicePairing_InvalidatesPendingPairing(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	ts := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, ts)
	devID := createSeedDevice(t, ts)

	pairing, err := ts.DevicePairingService.CreatePairing(context.Background(), defaultCreatePairingRequest(devID))
	is.NoErr(err)

	resp, err := client.DeleteDevicePairingWithResponse(ctx, devID.Int64(), pairing.ID.Int64())
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNoContent)

	fetched, err := ts.DevicePairingService.GetPairing(context.Background(), pairing.ID)
	is.NoErr(err)
	is.Equal(fetched.Status, devicepairing.StatusInvalidated)
}

func TestHandler_ClaimPairing_SuccessfulClaim(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	ts := testutils.SetupIntegrationServer(t)
	devID := createSeedDevice(t, ts)

	pairing, err := ts.DevicePairingService.CreatePairing(context.Background(), defaultCreatePairingRequest(devID))
	is.NoErr(err)

	resp, err := testutils.NewAPIClient(t, ts).ClaimPairingWithResponse(ctx, httpapi.ClaimPairingJSONRequestBody{
		Code: pairing.PairingCode,
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	is.Equal(resp.JSON200.ServerUrl, "https://pulse.home.lan")
	is.Equal(resp.JSON200.IntervalSeconds, 900)
	is.True(resp.JSON200.ApiKey != "")

	fetched, err := ts.DevicePairingService.GetPairing(context.Background(), pairing.ID)
	is.NoErr(err)
	is.Equal(fetched.Status, devicepairing.StatusUsed)
}

func TestHandler_ClaimPairing_UnknownCodeReturns404(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	ts := testutils.SetupIntegrationServer(t)

	resp, err := testutils.NewAPIClient(t, ts).ClaimPairingWithResponse(ctx, httpapi.ClaimPairingJSONRequestBody{
		Code: "totallyinvalidcode",
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNotFound)
}

func TestHandler_ClaimPairing_CodeUsedTwiceReturns404(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	ts := testutils.SetupIntegrationServer(t)
	devID := createSeedDevice(t, ts)

	pairing, err := ts.DevicePairingService.CreatePairing(context.Background(), defaultCreatePairingRequest(devID))
	is.NoErr(err)
	_, err = ts.DevicePairingService.ClaimPairing(context.Background(), pairing.PairingCode)
	is.NoErr(err)

	resp, err := testutils.NewAPIClient(t, ts).ClaimPairingWithResponse(ctx, httpapi.ClaimPairingJSONRequestBody{
		Code: pairing.PairingCode,
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNotFound)
}

func TestHandler_ClaimPairing_InvalidatedCodeReturns404(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	ts := testutils.SetupIntegrationServer(t)
	devID := createSeedDevice(t, ts)

	pairing, err := ts.DevicePairingService.CreatePairing(context.Background(), defaultCreatePairingRequest(devID))
	is.NoErr(err)
	err = ts.DevicePairingService.InvalidatePairing(context.Background(), devID, pairing.ID)
	is.NoErr(err)

	resp, err := testutils.NewAPIClient(t, ts).ClaimPairingWithResponse(ctx, httpapi.ClaimPairingJSONRequestBody{
		Code: pairing.PairingCode,
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNotFound)
}

func TestHandler_ClaimPairing_RegeneratedKeyHasCorrectPrefix(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	ts := testutils.SetupIntegrationServer(t)
	devID := createSeedDevice(t, ts)

	pairing, err := ts.DevicePairingService.CreatePairing(context.Background(), defaultCreatePairingRequest(devID))
	is.NoErr(err)

	resp, err := testutils.NewAPIClient(t, ts).ClaimPairingWithResponse(ctx, httpapi.ClaimPairingJSONRequestBody{
		Code: pairing.PairingCode,
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	is.True(len(resp.JSON200.ApiKey) > len(device.APIKeyPrefix))
	is.Equal(resp.JSON200.ApiKey[:len(device.APIKeyPrefix)], device.APIKeyPrefix)
}
