//go:build test

package rule_test

import (
	"net/http"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/app"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/rule"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

func createTestDevice(t *testing.T, testServer *app.App, name string) *device.Device {
	t.Helper()

	dev, err := testServer.DeviceService.CreateDevice(t.Context(), testutils.AdminPrincipal(t, testServer), name, nil)
	if err != nil {
		t.Fatalf("create device %q: %v", name, err)
	}
	return dev
}

func createDeviceAddressLeaseRule(t *testing.T, testServer *app.App, deviceID ids.DeviceID, ttlSeconds int) *rule.DeviceAddressLeaseRule {
	t.Helper()

	r, err := testServer.RuleService.EnableDeviceAddressLeaseRule(t.Context(), deviceID, ttlSeconds)
	if err != nil {
		t.Fatalf("enable lease rule for device %d: %v", deviceID, err)
	}
	return r
}

func TestHandler_GetDeviceAddressLeaseRule_HappyPath(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev := createTestDevice(t, testServer, "lease-device-get")
	r := createDeviceAddressLeaseRule(t, testServer, dev.ID, 300)

	resp, err := client.GetDeviceAddressLeaseRuleWithResponse(ctx, dev.ID.Int64())
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	is.True(resp.JSON200.Id != nil)
	is.Equal(*resp.JSON200.Id, int64(r.ID))
	is.Equal(resp.JSON200.DeviceId, int64(dev.ID))
	is.Equal(resp.JSON200.Enabled, r.Enabled)
	is.True(resp.JSON200.TtlSeconds != nil)
	is.Equal(*resp.JSON200.TtlSeconds, r.Config.TTLSeconds)
}

func TestHandler_GetDeviceAddressLeaseRule_NotConfigured_ReturnsDisabled(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev := createTestDevice(t, testServer, "lease-device-no-rule")

	resp, err := client.GetDeviceAddressLeaseRuleWithResponse(ctx, dev.ID.Int64())
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	is.True(!resp.JSON200.Enabled)
	is.Equal(resp.JSON200.DeviceId, int64(dev.ID))
	is.True(resp.JSON200.Id == nil)
	is.True(resp.JSON200.TtlSeconds == nil)
}

func TestHandler_PutDeviceAddressLeaseRule_HappyPath(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev := createTestDevice(t, testServer, "lease-device-put")

	resp, err := client.PutDeviceAddressLeaseRuleWithResponse(ctx, dev.ID.Int64(), httpapi.PutDeviceAddressLeaseRuleJSONRequestBody{
		TtlSeconds: 600,
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	is.Equal(resp.JSON200.DeviceId, int64(dev.ID))
	is.True(resp.JSON200.TtlSeconds != nil)
	is.Equal(*resp.JSON200.TtlSeconds, 600)
	is.True(resp.JSON200.Enabled)
}

func TestHandler_PutDeviceAddressLeaseRule_InvalidBody(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev := createTestDevice(t, testServer, "lease-device-bad-body")

	// TtlSeconds=0 fails handler validation (must be >= 1) → 400.
	resp, err := client.PutDeviceAddressLeaseRuleWithResponse(ctx, dev.ID.Int64(), httpapi.PutDeviceAddressLeaseRuleJSONRequestBody{
		TtlSeconds: 0,
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusBadRequest)
}

func TestHandler_PutDeviceAddressLeaseRule_DeviceNotFound(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	nonExistentDeviceID := ids.DeviceID(999999)
	resp, err := client.PutDeviceAddressLeaseRuleWithResponse(ctx, int64(nonExistentDeviceID), httpapi.PutDeviceAddressLeaseRuleJSONRequestBody{
		TtlSeconds: 300,
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNotFound)
}

func TestHandler_DisableDeviceAddressLeaseRule_HappyPath(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev := createTestDevice(t, testServer, "lease-device-disable")
	createDeviceAddressLeaseRule(t, testServer, dev.ID, 120)

	resp, err := client.DisableDeviceAddressLeaseRuleWithResponse(ctx, dev.ID.Int64())
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNoContent)

	ttl, err := testServer.RuleService.GetDeviceAddressLeaseTTLSeconds(t.Context(), dev.ID)
	is.NoErr(err)
	is.True(ttl == nil)
}

func TestHandler_DisableDeviceAddressLeaseRule_IdempotentWhenMissing(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev := createTestDevice(t, testServer, "lease-device-disable-missing")

	resp, err := client.DisableDeviceAddressLeaseRuleWithResponse(ctx, dev.ID.Int64())
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNoContent)
}

func createMaxActiveAddressesRule(t *testing.T, testServer *app.App, deviceID ids.DeviceID, maxAddresses int) *rule.MaxActiveAddressesRule {
	t.Helper()

	r, err := testServer.RuleService.EnableMaxActiveAddressesRule(t.Context(), deviceID, maxAddresses)
	if err != nil {
		t.Fatalf("enable max active addresses rule for device %d: %v", deviceID, err)
	}
	return r
}

func TestHandler_GetMaxActiveAddressesRule_HappyPath(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev := createTestDevice(t, testServer, "max-active-get-device")
	r := createMaxActiveAddressesRule(t, testServer, dev.ID, 3)

	resp, err := client.GetMaxActiveAddressesRuleWithResponse(ctx, dev.ID.Int64())
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	is.True(resp.JSON200.Id != nil)
	is.Equal(*resp.JSON200.Id, int64(r.ID))
	is.Equal(resp.JSON200.DeviceId, int64(dev.ID))
	is.Equal(resp.JSON200.Enabled, r.Enabled)
	is.True(resp.JSON200.MaxAddresses != nil)
	is.Equal(*resp.JSON200.MaxAddresses, r.Config.MaxAddresses)
}

func TestHandler_GetMaxActiveAddressesRule_NotConfigured_ReturnsDisabled(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev := createTestDevice(t, testServer, "max-active-get-notfound")

	resp, err := client.GetMaxActiveAddressesRuleWithResponse(ctx, dev.ID.Int64())
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	is.True(!resp.JSON200.Enabled)
	is.Equal(resp.JSON200.DeviceId, int64(dev.ID))
	is.True(resp.JSON200.Id == nil)
	is.True(resp.JSON200.MaxAddresses == nil)
}

func TestHandler_PutMaxActiveAddressesRule_HappyPath(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev := createTestDevice(t, testServer, "max-active-put-device")

	resp, err := client.PutMaxActiveAddressesRuleWithResponse(ctx, dev.ID.Int64(), httpapi.PutMaxActiveAddressesRuleJSONRequestBody{
		MaxAddresses: 5,
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	is.Equal(resp.JSON200.DeviceId, int64(dev.ID))
	is.True(resp.JSON200.MaxAddresses != nil)
	is.Equal(*resp.JSON200.MaxAddresses, 5)
	is.True(resp.JSON200.Enabled)
}

func TestHandler_PutMaxActiveAddressesRule_InvalidBody(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev := createTestDevice(t, testServer, "max-active-put-invalid")

	resp, err := client.PutMaxActiveAddressesRuleWithResponse(ctx, dev.ID.Int64(), httpapi.PutMaxActiveAddressesRuleJSONRequestBody{
		MaxAddresses: 0,
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusBadRequest)
}

func TestHandler_PutMaxActiveAddressesRule_DeviceNotFound(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	nonExistentDeviceID := ids.DeviceID(999999)
	resp, err := client.PutMaxActiveAddressesRuleWithResponse(ctx, int64(nonExistentDeviceID), httpapi.PutMaxActiveAddressesRuleJSONRequestBody{
		MaxAddresses: 3,
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNotFound)
}

func TestHandler_DisableMaxActiveAddressesRule_HappyPath(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev := createTestDevice(t, testServer, "max-active-disable-device")
	createMaxActiveAddressesRule(t, testServer, dev.ID, 2)

	resp, err := client.DisableMaxActiveAddressesRuleWithResponse(ctx, dev.ID.Int64())
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNoContent)

	max, err := testServer.RuleService.GetMaxActiveAddresses(t.Context(), dev.ID)
	is.NoErr(err)
	is.True(max == nil)
}

func TestHandler_DisableMaxActiveAddressesRule_IdempotentWhenMissing(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev := createTestDevice(t, testServer, "max-active-disable-missing")

	resp, err := client.DisableMaxActiveAddressesRuleWithResponse(ctx, dev.ID.Int64())
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNoContent)
}
