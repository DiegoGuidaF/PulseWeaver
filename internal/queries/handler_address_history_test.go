//go:build test

package queries_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

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

	// First event for the device — nothing to compare against.
	is.True(historyResp.Events[0].TimeGapSeconds == nil)
	is.True(!historyResp.Events[0].IpChanged)
	is.True(!historyResp.Events[0].IsRefresh)
	is.True(historyResp.Events[0].TtlSeconds == nil) // no lease rule configured
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

	for _, e := range page2.Events {
		is.True(e.Id < *page1.NextCursor)
	}
}
