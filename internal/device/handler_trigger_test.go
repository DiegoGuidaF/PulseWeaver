//go:build test

package device_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

// newestAddressEvent reads a device's latest address event back through the
// address-history endpoint — the surface the trigger exists for — rather than
// querying address_events directly.
func newestAddressEvent(t *testing.T, ctx context.Context, client *httpapi.ClientWithResponses, deviceID ids.DeviceID) httpapi.AddressHistoryEvent {
	t.Helper()
	is := is.New(t)

	deviceIDFilter := []httpapi.ID{deviceID.Int64()}
	resp, err := client.GetAddressHistoryWithResponse(ctx, &httpapi.GetAddressHistoryParams{
		DeviceId: &deviceIDFilter,
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	is.True(len(resp.JSON200.Events) > 0)

	return resp.JSON200.Events[0] // newest first
}

// TestHandler_DeviceHeartbeatByApiKey_TriggerAttribution covers the full
// trigger matrix of the API-key heartbeat, including the two degrading cases:
// a value only the server may set and a value from no known vocabulary. Both
// must still return a success — the heartbeat is what keeps the device
// authorized, so it can never fail over its own annotation.
func TestHandler_DeviceHeartbeatByApiKey_TriggerAttribution(t *testing.T) {
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	cases := []struct {
		name  string
		param *string
		want  httpapi.AddressEventTrigger
	}{
		{"user pressed the app button", new("user"), httpapi.AddressEventTriggerUser},
		{"background scheduler fired", new("schedule"), httpapi.AddressEventTriggerSchedule},
		{"network changed", new("network_change"), httpapi.AddressEventTriggerNetworkChange},
		{"system is not claimable over the wire", new("system"), httpapi.AddressEventTriggerSchedule},
		{"unrecognised value degrades", new("not-a-trigger"), httpapi.AddressEventTriggerSchedule},
		{"omitted by an older client", nil, httpapi.AddressEventTriggerSchedule},
	}

	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)

			dev, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), fmt.Sprintf("trigger-device-%d", i), nil)
			is.NoErr(err)
			_, apiKey, err := testServer.DeviceService.RegenerateAPIKey(ctx, dev.ID)
			is.NoErr(err)

			apiClient := testutils.NewAPIClient(t, testServer,
				testutils.WithAPIKey(apiKey),
				testutils.WithRealIP(fmt.Sprintf("192.168.5.%d", i+1)),
			)
			resp, err := apiClient.DeviceHeartbeatByAPIKeyWithResponse(ctx, &httpapi.DeviceHeartbeatByAPIKeyParams{
				TriggerType: tc.param,
			})
			is.NoErr(err)
			is.Equal(resp.StatusCode(), http.StatusCreated)
			is.Equal(resp.JSON201.Source, httpapi.Heartbeat)

			event := newestAddressEvent(t, ctx, client, dev.ID)
			is.Equal(event.Source, httpapi.Heartbeat)
			is.Equal(event.TriggerType, tc.want)
		})
	}
}

// TestHandler_DeviceHeartbeat_IsWebUIUserAction pins what the path-based
// heartbeat is for: it is the web UI's "register my current IP" button, so it
// always records a web-UI user action.
func TestHandler_DeviceHeartbeat_IsWebUIUserAction(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "session-heartbeat", nil)
	is.NoErr(err)

	sessionClient := testutils.NewAdminAPIClient(t, testServer, testutils.WithRealIP("192.168.6.1"))
	resp, err := sessionClient.DeviceHeartbeatWithResponse(ctx, dev.ID.Int64())
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusCreated)
	is.Equal(resp.JSON201.Source, httpapi.WebUi)

	event := newestAddressEvent(t, ctx, client, dev.ID)
	is.Equal(event.Source, httpapi.WebUi)
	is.Equal(event.TriggerType, httpapi.AddressEventTriggerUser)
}

// TestHandler_DeviceHeartbeat_RejectsDeviceAPIKey guards the assumption the
// handler is built on. It declares no apiKeyAuth, so the request validator
// turns a device key away before the handler runs — which is why the handler
// can attribute unconditionally instead of branching on the principal. If this
// route ever gains apiKeyAuth, that attribution goes silently wrong.
func TestHandler_DeviceHeartbeat_RejectsDeviceAPIKey(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)

	dev, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "apikey-path-heartbeat", nil)
	is.NoErr(err)
	_, apiKey, err := testServer.DeviceService.RegenerateAPIKey(ctx, dev.ID)
	is.NoErr(err)

	apiClient := testutils.NewAPIClient(t, testServer,
		testutils.WithAPIKey(apiKey),
		testutils.WithRealIP("192.168.6.2"),
	)
	resp, err := apiClient.DeviceHeartbeatWithResponse(ctx, dev.ID.Int64())
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusUnauthorized)
}

// TestHandler_AdminActionsAreWebUIUserActions covers the write paths an admin
// drives from the browser: they share one provenance pair, whichever endpoint
// or bulk path records them.
func TestHandler_AdminActionsAreWebUIUserActions(t *testing.T) {
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	cases := []struct {
		name string
		act  func(t *testing.T, deviceID ids.DeviceID, addressID httpapi.ID)
	}{
		{"address added by hand", func(_ *testing.T, _ ids.DeviceID, _ httpapi.ID) {}},
		{"address toggled off", func(t *testing.T, deviceID ids.DeviceID, addressID httpapi.ID) {
			is := is.New(t)
			resp, err := client.DisableAddressWithResponse(ctx, deviceID.Int64(), addressID)
			is.NoErr(err)
			is.Equal(resp.StatusCode(), http.StatusOK)
		}},
		{"device disabled", func(t *testing.T, deviceID ids.DeviceID, _ httpapi.ID) {
			is := is.New(t)
			resp, err := client.DisableDeviceWithResponse(ctx, deviceID.Int64())
			is.NoErr(err)
			is.Equal(resp.StatusCode(), http.StatusOK)
		}},
	}

	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)

			dev, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), fmt.Sprintf("admin-action-%d", i), nil)
			is.NoErr(err)

			addResp, err := client.AddAddressWithResponse(ctx, dev.ID.Int64(), httpapi.AddAddressJSONRequestBody{
				Ip: fmt.Sprintf("192.168.7.%d", i+1),
			})
			is.NoErr(err)
			is.Equal(addResp.StatusCode(), http.StatusCreated)

			tc.act(t, dev.ID, addResp.JSON201.Id)

			event := newestAddressEvent(t, ctx, client, dev.ID)
			is.Equal(event.Source, httpapi.WebUi)
			is.Equal(event.TriggerType, httpapi.AddressEventTriggerUser)
		})
	}
}

// TestHandler_GetAddressHistory_TriggerFilter exercises the new filter column
// in both directions, since in and not_in are separate registry operators.
func TestHandler_GetAddressHistory_TriggerFilter(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "trigger-filter", nil)
	is.NoErr(err)

	// One address per trigger, so a filter's expected count is unambiguous.
	for ip, trigger := range map[string]device.EventTrigger{
		"10.7.0.1": device.EventTriggerSchedule,
		"10.7.0.2": device.EventTriggerNetworkChange,
		"10.7.0.3": device.EventTriggerUser,
	} {
		_, _, err = testServer.DeviceService.RegisterAddressActivity(ctx, dev.ID, ip, device.EventSourceHeartbeat, trigger)
		is.NoErr(err)
	}

	deviceIDFilter := []httpapi.ID{dev.ID.Int64()}
	userOnly := []httpapi.AddressEventTrigger{httpapi.AddressEventTriggerUser}

	inResp, err := client.GetAddressHistoryWithResponse(ctx, &httpapi.GetAddressHistoryParams{
		DeviceId: &deviceIDFilter, TriggerType: &userOnly,
	})
	is.NoErr(err)
	is.Equal(inResp.StatusCode(), http.StatusOK)
	is.Equal(inResp.JSON200.Total, 1)
	is.Equal(inResp.JSON200.Events[0].TriggerType, httpapi.AddressEventTriggerUser)

	notIn := httpapi.AddressHistoryFilterOperatorNotIn
	outResp, err := client.GetAddressHistoryWithResponse(ctx, &httpapi.GetAddressHistoryParams{
		DeviceId: &deviceIDFilter, TriggerType: &userOnly, TriggerTypeOp: &notIn,
	})
	is.NoErr(err)
	is.Equal(outResp.StatusCode(), http.StatusOK)
	is.Equal(outResp.JSON200.Total, 2)
	for _, e := range outResp.JSON200.Events {
		is.True(e.TriggerType != httpapi.AddressEventTriggerUser)
	}
}
