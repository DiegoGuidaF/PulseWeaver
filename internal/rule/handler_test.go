//go:build test

package rule_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/app"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/rule"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

func createTestDevice(t *testing.T, testServer *app.App, name string) *device.Device {
	t.Helper()

	dev, _, err := testServer.DeviceService.CreateDevice(t.Context(), name)
	if err != nil {
		t.Fatalf("create device %q: %v", name, err)
	}
	return dev
}

func createDeviceAddressLeaseRule(t *testing.T, testServer *app.App, deviceID device.DeviceID, ttlSeconds int) *rule.DeviceAddressLeaseRule {
	t.Helper()

	r, err := testServer.RuleService.EnableDeviceAddressLeaseRule(t.Context(), deviceID, ttlSeconds)
	if err != nil {
		t.Fatalf("enable lease rule for device %d: %v", deviceID, err)
	}
	return r
}

func TestHandler_GetDeviceAddressLeaseRule_HappyPath(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	server := testServer.HTTPServer
	adminCookie := testutils.LoginCookie(t, server, "admin", "AdminPass123!")

	dev := createTestDevice(t, testServer, "lease-device-get")
	r := createDeviceAddressLeaseRule(t, testServer, dev.ID, 300)

	url := fmt.Sprintf("/api/v1/devices/%d/rules/address_lease", dev.ID)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.AddCookie(adminCookie)
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusOK)

	var resp httpapi.DeviceAddressLeaseRule
	err := json.NewDecoder(res.Body).Decode(&resp)
	is.NoErr(err)
	is.Equal(resp.Id, int64(r.ID))
	is.Equal(resp.DeviceId, int64(dev.ID))
	is.Equal(resp.Enabled, r.Enabled)
	is.Equal(resp.TtlSeconds, r.Config.TTLSeconds)
}

func TestHandler_GetDeviceAddressLeaseRule_NotFound(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	server := testServer.HTTPServer
	adminCookie := testutils.LoginCookie(t, server, "admin", "AdminPass123!")

	dev := createTestDevice(t, testServer, "lease-device-no-rule")

	url := fmt.Sprintf("/api/v1/devices/%d/rules/address_lease", dev.ID)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.AddCookie(adminCookie)
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusNotFound)

	var errResp httpapi.ErrorResponse
	err := json.NewDecoder(res.Body).Decode(&errResp)
	is.NoErr(err)
	is.True(errResp.Error != nil)
}

func TestHandler_PutDeviceAddressLeaseRule_HappyPath(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	server := testServer.HTTPServer
	adminCookie := testutils.LoginCookie(t, server, "admin", "AdminPass123!")

	dev := createTestDevice(t, testServer, "lease-device-put")

	body, _ := json.Marshal(map[string]int{
		"ttl_seconds": 600,
	})

	url := fmt.Sprintf("/api/v1/devices/%d/rules/address_lease", dev.ID)
	req := httptest.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(adminCookie)
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusOK)

	var resp httpapi.DeviceAddressLeaseRule
	err := json.NewDecoder(res.Body).Decode(&resp)
	is.NoErr(err)
	is.Equal(resp.DeviceId, int64(dev.ID))
	is.Equal(resp.TtlSeconds, 600)
	is.True(resp.Enabled)
}

func TestHandler_PutDeviceAddressLeaseRule_InvalidBody(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	server := testServer.HTTPServer
	adminCookie := testutils.LoginCookie(t, server, "admin", "AdminPass123!")

	dev := createTestDevice(t, testServer, "lease-device-bad-body")

	url := fmt.Sprintf("/api/v1/devices/%d/rules/address_lease", dev.ID)
	req := httptest.NewRequest(http.MethodPut, url, nil)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(adminCookie)
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusBadRequest)
}

func TestHandler_PutDeviceAddressLeaseRule_DeviceNotFound(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	server := testServer.HTTPServer
	adminCookie := testutils.LoginCookie(t, server, "admin", "AdminPass123!")

	body, _ := json.Marshal(map[string]int{
		"ttl_seconds": 300,
	})

	nonExistentDeviceID := device.DeviceID(999999)
	url := fmt.Sprintf("/api/v1/devices/%d/rules/address_lease", nonExistentDeviceID)
	req := httptest.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(adminCookie)
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusNotFound)
}

func TestHandler_DisableDeviceAddressLeaseRule_HappyPath(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	server := testServer.HTTPServer
	adminCookie := testutils.LoginCookie(t, server, "admin", "AdminPass123!")

	dev := createTestDevice(t, testServer, "lease-device-disable")
	createDeviceAddressLeaseRule(t, testServer, dev.ID, 120)

	url := fmt.Sprintf("/api/v1/devices/%d/rules/address_lease", dev.ID)
	req := httptest.NewRequest(http.MethodDelete, url, nil)
	req.AddCookie(adminCookie)
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusNoContent)

	ttl, err := testServer.RuleService.GetDeviceAddressLeaseTTLSeconds(t.Context(), dev.ID)
	is.NoErr(err)
	is.True(ttl == nil)
}

func TestHandler_DisableDeviceAddressLeaseRule_IdempotentWhenMissing(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	server := testServer.HTTPServer
	adminCookie := testutils.LoginCookie(t, server, "admin", "AdminPass123!")

	dev := createTestDevice(t, testServer, "lease-device-disable-missing")

	url := fmt.Sprintf("/api/v1/devices/%d/rules/address_lease", dev.ID)
	req := httptest.NewRequest(http.MethodDelete, url, nil)
	req.AddCookie(adminCookie)
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusNoContent)
}
