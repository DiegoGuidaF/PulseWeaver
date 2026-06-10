//go:build test

package device_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

func TestHandler_AddressLifecycle(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "router", nil)
	is.NoErr(err)

	// Create — 201, all fields populated
	addResp, err := client.AddAddressWithResponse(ctx, dev.ID.Int64(), httpapi.AddAddressJSONRequestBody{
		Ip: "192.168.1.100",
	})
	is.NoErr(err)
	is.Equal(addResp.StatusCode(), http.StatusCreated)
	created := *addResp.JSON201
	is.True(created.Id != 0)
	is.Equal(created.DeviceId, dev.ID.Int64())
	is.Equal(created.Ip, "192.168.1.100")
	is.True(created.IsEnabled)
	is.Equal(string(created.Source), "manual")
	is.True(!time.Time(created.CreatedAt).IsZero())
	is.True(!time.Time(created.UpdatedAt).IsZero())
	is.True(created.ExpiresAt == nil)

	// List — address appears with same fields
	listResp, err := client.GetDeviceAddressesWithResponse(ctx, dev.ID.Int64())
	is.NoErr(err)
	is.Equal(listResp.StatusCode(), http.StatusOK)
	addresses := *listResp.JSON200
	is.Equal(len(addresses), 1)
	is.Equal(addresses[0].Id, created.Id)
	is.Equal(addresses[0].Ip, "192.168.1.100")
	is.True(addresses[0].IsEnabled)
	is.Equal(string(addresses[0].Source), "manual")

	// Disable — 200, same id, is_enabled=false, source=manual
	disableResp, err := client.DisableAddressWithResponse(ctx, dev.ID.Int64(), created.Id)
	is.NoErr(err)
	is.Equal(disableResp.StatusCode(), http.StatusOK)
	disabled := *disableResp.JSON200
	is.Equal(disabled.Id, created.Id)
	is.True(!disabled.IsEnabled)
	is.Equal(string(disabled.Source), "manual")

	// Re-enable same IP — 200 (update, not create), same id, is_enabled=true, source=manual
	addResp2, err := client.AddAddressWithResponse(ctx, dev.ID.Int64(), httpapi.AddAddressJSONRequestBody{
		Ip: "192.168.1.100",
	})
	is.NoErr(err)
	is.Equal(addResp2.StatusCode(), http.StatusOK)
	reenabled := *addResp2.JSON200
	is.Equal(reenabled.Id, created.Id)
	is.True(reenabled.IsEnabled)
	is.Equal(string(reenabled.Source), "manual")
}

func TestHandler_DeviceHeartbeat(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)

	dev, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "checkin-device", nil)
	is.NoErr(err)

	// First heartbeat — creates address, 201, source=heartbeat
	firstClient := testutils.NewAdminAPIClient(t, testServer, testutils.WithRealIP("192.168.1.50"))
	firstResp, err := firstClient.DeviceHeartbeatWithResponse(ctx, dev.ID.Int64())
	is.NoErr(err)
	is.Equal(firstResp.StatusCode(), http.StatusCreated)
	created := *firstResp.JSON201
	is.True(created.Id != 0)
	is.Equal(created.DeviceId, dev.ID.Int64())
	is.Equal(created.Ip, "192.168.1.50")
	is.True(created.IsEnabled)
	is.Equal(string(created.Source), "heartbeat")
	is.True(!time.Time(created.CreatedAt).IsZero())
	is.True(!time.Time(created.UpdatedAt).IsZero())

	// Second heartbeat — refreshes same address, 200, source=heartbeat
	secondClient := testutils.NewAdminAPIClient(t, testServer, testutils.WithRealIP("192.168.1.50"))
	secondResp, err := secondClient.DeviceHeartbeatWithResponse(ctx, dev.ID.Int64())
	is.NoErr(err)
	is.Equal(secondResp.StatusCode(), http.StatusOK)
	refreshed := *secondResp.JSON200
	is.Equal(refreshed.Id, created.Id)
	is.True(refreshed.IsEnabled)
	is.Equal(string(refreshed.Source), "heartbeat")
}

func TestHandler_CreateDevice(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	resp, err := client.CreateDeviceWithResponse(ctx, httpapi.CreateDeviceJSONRequestBody{
		Name: "sensor-1",
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusCreated)
	created := resp.JSON201.Device
	is.Equal(created.Name, "sensor-1")
	is.True(created.Id != 0)
	// No credential requested — no key minted, none returned.
	is.True(created.ApiKeyPrefix == nil)
	is.True(resp.JSON201.ApiKey == nil)
}

func TestHandler_CreateDevice_WithApiKey(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	genKey := true
	desc := "Juan's work phone"
	resp, err := client.CreateDeviceWithResponse(ctx, httpapi.CreateDeviceJSONRequestBody{
		Name:           "bob-phone",
		GenerateApiKey: &genKey,
		Description:    httpapi.NullableString{Set: true, Value: &desc},
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusCreated)
	// Key minted atomically and returned once.
	is.True(resp.JSON201.ApiKey != nil)
	is.True(*resp.JSON201.ApiKey != "")
	dev := resp.JSON201.Device
	is.True(dev.ApiKeyPrefix != nil) // key reflected on the device
	is.True(dev.Description != nil && *dev.Description == desc)
}

func TestHandler_DeviceHeartbeatByApiKey_NoBody(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)

	dev, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "apikey-heartbeat-device", nil)
	is.NoErr(err)
	_, apiKey, err := testServer.DeviceService.RegenerateAPIKey(ctx, dev.ID)
	is.NoErr(err)

	// POST /heartbeat with X-API-Key, no body — IP comes from X-Real-IP (RemoteAddr is the trusted proxy)
	apiClient := testutils.NewAPIClient(t, testServer, testutils.WithAPIKey(apiKey), testutils.WithRealIP("192.168.1.99"))
	resp, err := apiClient.DeviceHeartbeatByAPIKeyWithResponse(ctx)
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusCreated)
	addr := *resp.JSON201
	is.True(addr.Id != 0)
	is.Equal(addr.DeviceId, dev.ID.Int64())
	is.Equal(addr.Ip, "192.168.1.99")
	is.True(addr.IsEnabled)
	is.Equal(string(addr.Source), "heartbeat")
	is.True(!time.Time(addr.CreatedAt).IsZero())
	is.True(!time.Time(addr.UpdatedAt).IsZero())
}

func TestHandler_DeviceHeartbeatByApiKey_401_NoKey(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)

	// No X-API-Key header — auth is checked before IP validation
	resp, err := testutils.NewAPIClient(t, testServer).DeviceHeartbeatByAPIKeyWithResponse(ctx)
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusUnauthorized)
}

func TestHandler_DeviceHeartbeatByApiKey_401_InvalidKey(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)

	resp, err := testutils.NewAPIClient(t, testServer, testutils.WithAPIKey("wdk_invalid_key_that_does_not_exist_in_db")).DeviceHeartbeatByAPIKeyWithResponse(ctx)
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusUnauthorized)
}

func TestHandler_DeviceHeartbeat_404_DeviceNotFound(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)

	// Heartbeat for non-existent device_id (session auth).
	// X-Real-IP must be a routable address — the transport pins RemoteAddr to
	// 127.0.0.1 (trusted proxy), so without X-Real-IP the handler sees 127.0.0.1
	// (loopback) and returns 400 before it can reach the 404 device-not-found path.
	client := testutils.NewAdminAPIClient(t, testServer, testutils.WithRealIP("192.168.1.1"))
	resp, err := client.DeviceHeartbeatWithResponse(ctx, int64(99999))
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNotFound)
}

func TestHandler_DeleteDevice_204(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	device, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "to-delete", nil)
	is.NoErr(err)

	deleteResp, err := client.DeleteDeviceWithResponse(ctx, device.ID.Int64())
	is.NoErr(err)
	is.Equal(deleteResp.StatusCode(), http.StatusNoContent)

	// Device no longer in list; admin owner remains with zero devices.
	listResp, err := client.GetDevicesWithResponse(ctx)
	is.NoErr(err)
	is.Equal(listResp.StatusCode(), http.StatusOK)
	groups := *listResp.JSON200
	is.Equal(len(groups), 1)
	is.Equal(len(groups[0].Devices), 0)
}

func TestHandler_DeleteDevice_404(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	resp, err := client.DeleteDeviceWithResponse(ctx, int64(99999))
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNotFound)
}

func TestHandler_RegenerateDeviceApiKey_200(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	device, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "regen-device", nil)
	is.NoErr(err)

	resp, err := client.RegenerateDeviceAPIKeyWithResponse(ctx, device.ID.Int64())
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	body := *resp.JSON200
	is.Equal(body.Device.Id, int64(device.ID))
	is.Equal(body.Device.Name, device.Name)
	is.True(body.ApiKey != "")
}

func TestHandler_RegenerateDeviceApiKey_404(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	resp, err := client.RegenerateDeviceAPIKeyWithResponse(ctx, int64(99999))
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNotFound)
}

func TestHandler_DisableDevice_200(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "lost-phone", nil)
	is.NoErr(err)
	_, _, err = testServer.DeviceService.RegenerateAPIKey(ctx, dev.ID)
	is.NoErr(err)

	resp, err := client.DisableDeviceWithResponse(ctx, dev.ID.Int64())
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	body := *resp.JSON200
	is.True(body.DisabledAt != nil)   // flag stamped
	is.True(body.ApiKeyPrefix != nil) // API key is kept — disable is a freeze, not a de-credentialing

	// Device shows as disabled in the list.
	listResp, err := client.GetDevicesWithResponse(ctx)
	is.NoErr(err)
	groups := *listResp.JSON200
	is.Equal(groups[0].Devices[0].State, httpapi.Disabled)
}

func TestHandler_DisableDevice_404(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	resp, err := client.DisableDeviceWithResponse(ctx, int64(99999))
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNotFound)
}

func TestHandler_EnableDevice_200(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "recovered-phone", nil)
	is.NoErr(err)
	_, _, err = testServer.DeviceService.RegenerateAPIKey(ctx, dev.ID)
	is.NoErr(err)

	_, err = client.DisableDeviceWithResponse(ctx, dev.ID.Int64())
	is.NoErr(err)

	// Address enable/refresh is blocked while disabled.
	addResp, err := client.AddAddressWithResponse(ctx, dev.ID.Int64(), httpapi.AddAddressJSONRequestBody{Ip: "192.168.1.50"})
	is.NoErr(err)
	is.Equal(addResp.StatusCode(), http.StatusConflict)

	resp, err := client.EnableDeviceWithResponse(ctx, dev.ID.Int64())
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	is.True(resp.JSON200.DisabledAt == nil)   // disabled flag cleared
	is.True(resp.JSON200.ApiKeyPrefix != nil) // key preserved across the freeze

	// After re-enabling, address registration works again.
	addResp2, err := client.AddAddressWithResponse(ctx, dev.ID.Int64(), httpapi.AddAddressJSONRequestBody{Ip: "192.168.1.50"})
	is.NoErr(err)
	is.Equal(addResp2.StatusCode(), http.StatusCreated)
}

func TestHandler_EnableDevice_404(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	resp, err := client.EnableDeviceWithResponse(ctx, int64(99999))
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNotFound)
}

func TestHandler_DeleteDeviceApiKey_204(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "delete-key-device", nil)
	is.NoErr(err)
	// Generate a key first so there is something to delete
	_, _, err = testServer.DeviceService.RegenerateAPIKey(ctx, dev.ID)
	is.NoErr(err)

	resp, err := client.DeleteDeviceAPIKeyWithResponse(ctx, dev.ID.Int64())
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNoContent)
}

func TestHandler_DeleteDeviceApiKey_404_NoKey(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "no-key-device", nil)
	is.NoErr(err)

	resp, err := client.DeleteDeviceAPIKeyWithResponse(ctx, dev.ID.Int64())
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNotFound)
}

func TestHandler_DeleteDeviceApiKey_404_DeviceNotFound(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	resp, err := client.DeleteDeviceAPIKeyWithResponse(ctx, int64(99999))
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNotFound)
}

func TestHandler_CreateDevice_409_DuplicateName(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	// Create first device via service so name is taken
	_, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "dup-name", nil)
	is.NoErr(err)

	// POST create with same name via HTTP (tests handler 409)
	resp, err := client.CreateDeviceWithResponse(ctx, httpapi.CreateDeviceJSONRequestBody{
		Name: "dup-name",
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusConflict)
}

func TestHandler_UpdateDevice_Rename(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "sensor", nil)
	is.NoErr(err)

	resp, err := client.UpdateDeviceWithResponse(ctx, dev.ID.Int64(), httpapi.UpdateDeviceJSONRequestBody{
		Name: new("sensor-renamed"),
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	updated := *resp.JSON200
	is.Equal(updated.Name, "sensor-renamed")
	is.True(updated.Description == nil)
}

func TestHandler_UpdateDevice_SetAndClearDescription(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "noted-device", nil)
	is.NoErr(err)

	// Set description
	note := "my note"
	setResp, err := client.UpdateDeviceWithResponse(ctx, dev.ID.Int64(), httpapi.UpdateDeviceJSONRequestBody{
		Description: httpapi.NullableString{Set: true, Value: &note},
	})
	is.NoErr(err)
	is.Equal(setResp.StatusCode(), http.StatusOK)
	withDesc := *setResp.JSON200
	is.True(withDesc.Description != nil)
	is.Equal(*withDesc.Description, "my note")

	// Clear description via explicit null
	clearResp, err := client.UpdateDeviceWithResponse(ctx, dev.ID.Int64(), httpapi.UpdateDeviceJSONRequestBody{
		Description: httpapi.NullableString{Set: true, Value: nil},
	})
	is.NoErr(err)
	is.Equal(clearResp.StatusCode(), http.StatusOK)
	cleared := *clearResp.JSON200
	is.True(cleared.Description == nil)
}

func TestHandler_UpdateDevice_DuplicateName_Returns409(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	_, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "taken", nil)
	is.NoErr(err)
	dev2, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "to-rename", nil)
	is.NoErr(err)

	resp, err := client.UpdateDeviceWithResponse(ctx, dev2.ID.Int64(), httpapi.UpdateDeviceJSONRequestBody{
		Name: new("taken"),
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusConflict)
}

func TestHandler_UpdateDevice_NotFound_Returns404(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	resp, err := client.UpdateDeviceWithResponse(ctx, int64(9999), httpapi.UpdateDeviceJSONRequestBody{
		Name: new("ghost"),
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNotFound)
}
