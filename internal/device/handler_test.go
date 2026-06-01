//go:build test

package device_test

import (
	"context"
	"fmt"
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
	created := *resp.JSON201
	is.Equal(created.Name, "sensor-1")
	is.True(created.Id != 0)
	// No API key returned on device creation — must be generated separately.
	is.True(created.ApiKeyPrefix == nil)
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

	// Device no longer in list
	listResp, err := client.GetDevicesWithResponse(ctx)
	is.NoErr(err)
	is.Equal(listResp.StatusCode(), http.StatusOK)
	groups := *listResp.JSON200
	is.Equal(len(groups), 0)
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

func TestHandler_GetAddressHistory(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "history-device", nil)
	is.NoErr(err)

	// Register an address via service (creates an enable event)
	_, _, err = testServer.DeviceService.RegisterAddressActivity(ctx, dev.ID, "10.0.0.1", "heartbeat")
	is.NoErr(err)

	deviceID := dev.ID.Int64()
	resp, err := client.GetAddressHistoryWithResponse(ctx, &httpapi.GetAddressHistoryParams{
		DeviceId: &[]httpapi.ID{deviceID},
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	historyResp := *resp.JSON200

	is.True(len(historyResp.Buckets) >= 1)
	is.True(len(historyResp.Events) >= 1)
	is.Equal(historyResp.Events[0].Ip, "10.0.0.1")
	is.True(historyResp.Events[0].IsEnabled)
	is.Equal(historyResp.Events[0].DeviceName, "history-device")
	is.True(historyResp.TotalEvents >= 1)
}

func TestHandler_GetAddressHistory_AllDevices(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev1, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "dev-a", nil)
	is.NoErr(err)
	dev2, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "dev-b", nil)
	is.NoErr(err)

	_, _, err = testServer.DeviceService.RegisterAddressActivity(ctx, dev1.ID, "10.0.0.1", "heartbeat")
	is.NoErr(err)
	_, _, err = testServer.DeviceService.RegisterAddressActivity(ctx, dev2.ID, "10.0.0.2", "manual")
	is.NoErr(err)

	// No device_id filter → all devices
	resp, err := client.GetAddressHistoryWithResponse(ctx, &httpapi.GetAddressHistoryParams{})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	historyResp := *resp.JSON200

	is.True(historyResp.TotalEvents >= 2)
	is.True(len(historyResp.Events) >= 2)
}

func TestHandler_GetAddressHistory_InvalidGranularity(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)

	// The typed params only allow "day"/"hour"; send a raw invalid value via
	// the query string to exercise the handler's 400 path. A request editor
	// rewrites the query because the generated enum type rejects non-enum
	// values at compile time.
	client := testutils.NewAdminAPIClient(t, testServer,
		httpapi.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
			q := req.URL.Query()
			q.Set("granularity", "invalid")
			req.URL.RawQuery = q.Encode()
			return nil
		}),
	)
	resp, err := client.GetAddressHistoryWithResponse(ctx, &httpapi.GetAddressHistoryParams{})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusBadRequest)
}

func TestHandler_GetAddressHistory_Pagination(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "pagination-dev", nil)
	is.NoErr(err)

	// Create several events
	for i := range 5 {
		_, _, err = testServer.DeviceService.RegisterAddressActivity(ctx, dev.ID, fmt.Sprintf("10.0.0.%d", i+1), "heartbeat")
		is.NoErr(err)
	}

	// Page 1: limit 2
	limit := 2
	page1Resp, err := client.GetAddressHistoryWithResponse(ctx, &httpapi.GetAddressHistoryParams{
		Limit: &limit,
	})
	is.NoErr(err)
	is.Equal(page1Resp.StatusCode(), http.StatusOK)
	page1 := *page1Resp.JSON200
	is.Equal(len(page1.Events), 2)
	is.True(page1.NextCursor != nil) // more pages
	is.True(page1.TotalEvents >= 5)

	// Page 2: use cursor
	page2Resp, err := client.GetAddressHistoryWithResponse(ctx, &httpapi.GetAddressHistoryParams{
		Limit:    &limit,
		BeforeId: page1.NextCursor,
	})
	is.NoErr(err)
	is.Equal(page2Resp.StatusCode(), http.StatusOK)
	page2 := *page2Resp.JSON200
	is.Equal(len(page2.Events), 2)
}

func TestHandler_UpdateDevice_RenameAndSetType(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "sensor", nil)
	is.NoErr(err)

	deviceType := httpapi.UpdateDeviceRequestDeviceTypeMobile
	resp, err := client.UpdateDeviceWithResponse(ctx, dev.ID.Int64(), httpapi.UpdateDeviceJSONRequestBody{
		Name:       new("sensor-renamed"),
		DeviceType: &deviceType,
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	updated := *resp.JSON200
	is.Equal(updated.Name, "sensor-renamed")
	is.Equal(string(updated.DeviceType), "mobile")
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

func TestHandler_UpdateDevice_InvalidType_Returns400(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "type-test", nil)
	is.NoErr(err)

	// "robot" is not a valid UpdateDeviceRequestDeviceType enum value.
	// The typed struct cannot express invalid enum values, so we inject the
	// bad value via a request editor that rewrites the query string — but for
	// a body field we fall back to the body manipulation approach via a raw
	// request editor on the body.  The simplest alternative: use
	// UpdateDeviceWithBodyWithResponse with raw JSON.
	robotType := httpapi.UpdateDeviceRequestDeviceType("robot")
	resp, err := client.UpdateDeviceWithResponse(ctx, dev.ID.Int64(), httpapi.UpdateDeviceJSONRequestBody{
		DeviceType: &robotType,
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusBadRequest)
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

func TestHandler_ListDeviceTypes(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	resp, err := client.ListDeviceTypesWithResponse(ctx)
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	types := *resp.JSON200
	is.Equal(len(types), 2)
	is.Equal(types[0].Value, "static")
	is.Equal(types[0].Label, "Static")
	is.Equal(types[1].Value, "mobile")
}
