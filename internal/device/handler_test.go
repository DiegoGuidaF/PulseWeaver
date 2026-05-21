//go:build test

package device_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

func TestHandler_AddressLifecycle(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	dev, err := testServer.DeviceService.CreateDevice(t.Context(), testutils.AdminPrincipal(t, testServer), "router", nil)
	is.NoErr(err)

	addBody, _ := json.Marshal(map[string]string{"ip": "192.168.1.100"})
	addURL := fmt.Sprintf("/api/v1/devices/%d/addresses", dev.ID)
	addReq := httptest.NewRequest(http.MethodPost, addURL, bytes.NewReader(addBody))
	addReq.Header.Set("Content-Type", "application/json")
	addReq.AddCookie(sessionCookie)
	addRes := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(addRes, addReq)
	is.Equal(addRes.Code, http.StatusCreated)

	var createdAddress httpapi.Address
	err = json.NewDecoder(addRes.Body).Decode(&createdAddress)
	is.NoErr(err)
	is.True(createdAddress.IsEnabled)

	listReq := httptest.NewRequest(http.MethodGet, addURL, nil)
	listReq.AddCookie(sessionCookie)
	listRes := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(listRes, listReq)
	is.Equal(listRes.Code, http.StatusOK)

	var addresses []httpapi.Address
	err = json.NewDecoder(listRes.Body).Decode(&addresses)
	is.NoErr(err)
	is.Equal(len(addresses), 1)
	is.Equal(addresses[0].Ip, "192.168.1.100")
	is.True(addresses[0].IsEnabled)

	disableURL := fmt.Sprintf("/api/v1/devices/%d/addresses/%d", dev.ID, createdAddress.Id)
	disableReq := httptest.NewRequest(http.MethodDelete, disableURL, nil)
	disableReq.AddCookie(sessionCookie)
	disableRes := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(disableRes, disableReq)
	is.Equal(disableRes.Code, http.StatusOK)

	var disabled httpapi.Address
	err = json.NewDecoder(disableRes.Body).Decode(&disabled)
	is.NoErr(err)
	is.True(!disabled.IsEnabled)
}

func TestHandler_DeviceHeartbeat(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	dev, err := testServer.DeviceService.CreateDevice(t.Context(), testutils.AdminPrincipal(t, testServer), "checkin-device", nil)
	is.NoErr(err)

	heartbeatURL := fmt.Sprintf("/api/v1/devices/%d/heartbeat", dev.ID)

	firstReq := httptest.NewRequest(http.MethodPost, heartbeatURL, nil)
	firstReq.RemoteAddr = "192.168.1.50:12345"
	firstReq.AddCookie(sessionCookie)
	firstRes := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(firstRes, firstReq)
	is.Equal(firstRes.Code, http.StatusCreated)

	secondReq := httptest.NewRequest(http.MethodPost, heartbeatURL, nil)
	secondReq.RemoteAddr = "192.168.1.50:54321"
	secondReq.AddCookie(sessionCookie)
	secondRes := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(secondRes, secondReq)
	is.Equal(secondRes.Code, http.StatusOK)
}

func TestHandler_CreateDevice(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	createBody, _ := json.Marshal(map[string]string{"name": "sensor-1"})
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/devices", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.AddCookie(sessionCookie)
	createRes := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(createRes, createReq)
	is.Equal(createRes.Code, http.StatusCreated)

	var resp httpapi.Device
	err := json.NewDecoder(createRes.Body).Decode(&resp)
	is.NoErr(err)
	is.Equal(resp.Name, "sensor-1")
	is.True(resp.Id != 0)
	// No API key returned on device creation — must be generated separately.
	is.True(resp.ApiKeyPrefix == nil)
}

func TestHandler_DeviceHeartbeatByApiKey_NoBody(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)

	dev, err := testServer.DeviceService.CreateDevice(t.Context(), testutils.AdminPrincipal(t, testServer), "apikey-heartbeat-device", nil)
	is.NoErr(err)
	_, apiKey, err := testServer.DeviceService.RegenerateAPIKey(t.Context(), dev.ID)
	is.NoErr(err)

	// POST /heartbeat with X-API-Key (no session cookie needed)
	// Send empty JSON body, should use client ip from request context
	emptyBody, _ := json.Marshal(map[string]interface{}{})
	heartbeatReq := httptest.NewRequest(http.MethodPost, "/api/v1/heartbeat", bytes.NewReader(emptyBody))
	heartbeatReq.Header.Set("Content-Type", "application/json")
	heartbeatReq.RemoteAddr = "192.168.1.99:12345"
	heartbeatReq.Header.Set("X-API-Key", apiKey)
	heartbeatRes := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(heartbeatRes, heartbeatReq)
	is.Equal(heartbeatRes.Code, http.StatusCreated)

	var addr httpapi.Address
	err = json.NewDecoder(heartbeatRes.Body).Decode(&addr)
	is.NoErr(err)
	is.Equal(addr.Ip, "192.168.1.99")
	is.True(addr.IsEnabled)
}

func TestHandler_DeviceHeartbeatByApiKey_WithBodyIP(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)

	dev, err := testServer.DeviceService.CreateDevice(t.Context(), testutils.AdminPrincipal(t, testServer), "apikey-heartbeat-with-body-ip", nil)
	is.NoErr(err)
	_, apiKey, err := testServer.DeviceService.RegenerateAPIKey(t.Context(), dev.ID)
	is.NoErr(err)

	// POST /heartbeat with X-API-Key and IP in request body
	// The IP in body should be used instead of RemoteAddr
	heartbeatBody, _ := json.Marshal(map[string]string{"ip": "10.0.0.42"})
	heartbeatReq := httptest.NewRequest(http.MethodPost, "/api/v1/heartbeat", bytes.NewReader(heartbeatBody))
	heartbeatReq.Header.Set("Content-Type", "application/json")
	heartbeatReq.RemoteAddr = "192.168.1.99:12345" // This should be ignored when body IP is provided
	heartbeatReq.Header.Set("X-API-Key", apiKey)
	heartbeatRes := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(heartbeatRes, heartbeatReq)
	is.Equal(heartbeatRes.Code, http.StatusCreated)

	var addr httpapi.Address
	err = json.NewDecoder(heartbeatRes.Body).Decode(&addr)
	is.NoErr(err)
	// Verify the IP from body is used, not RemoteAddr
	is.Equal(addr.Ip, "10.0.0.42")
	is.True(addr.IsEnabled)
}

func TestHandler_DeviceHeartbeatByApiKey_401_NoKey(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)

	heartbeatReq := httptest.NewRequest(http.MethodPost, "/api/v1/heartbeat", nil)
	heartbeatReq.RemoteAddr = "192.168.1.1:0"
	// No X-API-Key header
	heartbeatRes := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(heartbeatRes, heartbeatReq)
	is.Equal(heartbeatRes.Code, http.StatusUnauthorized)
}

func TestHandler_DeviceHeartbeatByApiKey_401_InvalidKey(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)

	heartbeatReq := httptest.NewRequest(http.MethodPost, "/api/v1/heartbeat", nil)
	heartbeatReq.RemoteAddr = "192.168.1.1:0"
	heartbeatReq.Header.Set("X-API-Key", "wdk_invalid_key_that_does_not_exist_in_db")
	heartbeatRes := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(heartbeatRes, heartbeatReq)
	is.Equal(heartbeatRes.Code, http.StatusUnauthorized)
}

func TestHandler_DeviceHeartbeat_404_DeviceNotFound(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	// Heartbeat for non-existent device_id (session auth)
	heartbeatURL := fmt.Sprintf("/api/v1/devices/%d/heartbeat", 99999)
	heartbeatReq := httptest.NewRequest(http.MethodPost, heartbeatURL, nil)
	heartbeatReq.RemoteAddr = "192.168.1.1:0"
	heartbeatReq.AddCookie(sessionCookie)
	heartbeatRes := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(heartbeatRes, heartbeatReq)
	is.Equal(heartbeatRes.Code, http.StatusNotFound)
}

func TestHandler_DeleteDevice_204(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	device, err := testServer.DeviceService.CreateDevice(t.Context(), testutils.AdminPrincipal(t, testServer), "to-delete", nil)
	is.NoErr(err)

	deleteURL := fmt.Sprintf("/api/v1/devices/%d", device.ID)
	deleteReq := httptest.NewRequest(http.MethodDelete, deleteURL, nil)
	deleteReq.AddCookie(sessionCookie)
	deleteRes := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(deleteRes, deleteReq)
	is.Equal(deleteRes.Code, http.StatusNoContent)

	// Device no longer in list
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	listReq.AddCookie(sessionCookie)
	listRes := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(listRes, listReq)
	is.Equal(listRes.Code, http.StatusOK)
	var groups []httpapi.DeviceOwnerGroup
	err = json.NewDecoder(listRes.Body).Decode(&groups)
	is.NoErr(err)
	is.Equal(len(groups), 0)
}

func TestHandler_DeleteDevice_404(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	deleteURL := fmt.Sprintf("/api/v1/devices/%d", 99999)
	deleteReq := httptest.NewRequest(http.MethodDelete, deleteURL, nil)
	deleteReq.AddCookie(sessionCookie)
	deleteRes := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(deleteRes, deleteReq)
	is.Equal(deleteRes.Code, http.StatusNotFound)
}

func TestHandler_RegenerateDeviceApiKey_200(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	device, err := testServer.DeviceService.CreateDevice(t.Context(), testutils.AdminPrincipal(t, testServer), "regen-device", nil)
	is.NoErr(err)

	regenURL := fmt.Sprintf("/api/v1/devices/%d/api-key/regenerate", device.ID)
	regenReq := httptest.NewRequest(http.MethodPost, regenURL, nil)
	regenReq.AddCookie(sessionCookie)
	regenRes := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(regenRes, regenReq)
	is.Equal(regenRes.Code, http.StatusOK)

	var resp httpapi.DeviceAPIKeyResponse
	err = json.NewDecoder(regenRes.Body).Decode(&resp)
	is.NoErr(err)
	is.Equal(resp.Device.Id, int64(device.ID))
	is.Equal(resp.Device.Name, device.Name)
	is.True(resp.ApiKey != "")
}

func TestHandler_RegenerateDeviceApiKey_404(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	regenURL := fmt.Sprintf("/api/v1/devices/%d/api-key/regenerate", 99999)
	regenReq := httptest.NewRequest(http.MethodPost, regenURL, nil)
	regenReq.AddCookie(sessionCookie)
	regenRes := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(regenRes, regenReq)
	is.Equal(regenRes.Code, http.StatusNotFound)
}

func TestHandler_DeleteDeviceApiKey_204(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	dev, err := testServer.DeviceService.CreateDevice(t.Context(), testutils.AdminPrincipal(t, testServer), "delete-key-device", nil)
	is.NoErr(err)
	// Generate a key first so there is something to delete
	_, _, err = testServer.DeviceService.RegenerateAPIKey(t.Context(), dev.ID)
	is.NoErr(err)

	deleteURL := fmt.Sprintf("/api/v1/devices/%d/api-key", dev.ID)
	deleteReq := httptest.NewRequest(http.MethodDelete, deleteURL, nil)
	deleteReq.AddCookie(sessionCookie)
	deleteRes := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(deleteRes, deleteReq)
	is.Equal(deleteRes.Code, http.StatusNoContent)
}

func TestHandler_DeleteDeviceApiKey_404_NoKey(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	dev, err := testServer.DeviceService.CreateDevice(t.Context(), testutils.AdminPrincipal(t, testServer), "no-key-device", nil)
	is.NoErr(err)

	deleteURL := fmt.Sprintf("/api/v1/devices/%d/api-key", dev.ID)
	deleteReq := httptest.NewRequest(http.MethodDelete, deleteURL, nil)
	deleteReq.AddCookie(sessionCookie)
	deleteRes := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(deleteRes, deleteReq)
	is.Equal(deleteRes.Code, http.StatusNotFound)
}

func TestHandler_DeleteDeviceApiKey_404_DeviceNotFound(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	deleteURL := fmt.Sprintf("/api/v1/devices/%d/api-key", 99999)
	deleteReq := httptest.NewRequest(http.MethodDelete, deleteURL, nil)
	deleteReq.AddCookie(sessionCookie)
	deleteRes := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(deleteRes, deleteReq)
	is.Equal(deleteRes.Code, http.StatusNotFound)
}

func TestHandler_CreateDevice_409_DuplicateName(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	// Create first device via service so name is taken
	_, err := testServer.DeviceService.CreateDevice(t.Context(), testutils.AdminPrincipal(t, testServer), "dup-name", nil)
	is.NoErr(err)

	// POST create with same name via HTTP (tests handler 409)
	createBody, _ := json.Marshal(map[string]string{"name": "dup-name"})
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/devices", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.AddCookie(sessionCookie)
	createRes := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(createRes, createReq)
	is.Equal(createRes.Code, http.StatusConflict)
}

func TestHandler_GetAddressHistory(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	dev, err := testServer.DeviceService.CreateDevice(t.Context(), testutils.AdminPrincipal(t, testServer), "history-device", nil)
	is.NoErr(err)

	// Register an address via service (creates an enable event)
	_, _, err = testServer.DeviceService.RegisterAddressActivity(t.Context(), dev.ID, "10.0.0.1", "heartbeat")
	is.NoErr(err)

	url := fmt.Sprintf("/api/v1/address-history?device_id=%d", dev.ID)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.AddCookie(sessionCookie)
	res := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusOK)

	var historyResp httpapi.AddressHistoryResponse
	err = json.NewDecoder(res.Body).Decode(&historyResp)
	is.NoErr(err)

	is.True(len(historyResp.Buckets) >= 1)
	is.True(len(historyResp.Events) >= 1)
	is.Equal(historyResp.Events[0].Ip, "10.0.0.1")
	is.True(historyResp.Events[0].IsEnabled)
	is.Equal(historyResp.Events[0].DeviceName, "history-device")
	is.True(historyResp.TotalEvents >= 1)
}

func TestHandler_GetAddressHistory_AllDevices(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	dev1, err := testServer.DeviceService.CreateDevice(t.Context(), testutils.AdminPrincipal(t, testServer), "dev-a", nil)
	is.NoErr(err)
	dev2, err := testServer.DeviceService.CreateDevice(t.Context(), testutils.AdminPrincipal(t, testServer), "dev-b", nil)
	is.NoErr(err)

	_, _, err = testServer.DeviceService.RegisterAddressActivity(t.Context(), dev1.ID, "10.0.0.1", "heartbeat")
	is.NoErr(err)
	_, _, err = testServer.DeviceService.RegisterAddressActivity(t.Context(), dev2.ID, "10.0.0.2", "manual")
	is.NoErr(err)

	// No device_id filter → all devices
	req := httptest.NewRequest(http.MethodGet, "/api/v1/address-history", nil)
	req.AddCookie(sessionCookie)
	res := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusOK)

	var historyResp httpapi.AddressHistoryResponse
	err = json.NewDecoder(res.Body).Decode(&historyResp)
	is.NoErr(err)

	is.True(historyResp.TotalEvents >= 2)
	is.True(len(historyResp.Events) >= 2)
}

func TestHandler_GetAddressHistory_InvalidGranularity(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/address-history?granularity=invalid", nil)
	req.AddCookie(sessionCookie)
	res := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusBadRequest)
}

func TestHandler_GetAddressHistory_Pagination(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	dev, err := testServer.DeviceService.CreateDevice(t.Context(), testutils.AdminPrincipal(t, testServer), "pagination-dev", nil)
	is.NoErr(err)

	// Create several events
	for i := 0; i < 5; i++ {
		_, _, err = testServer.DeviceService.RegisterAddressActivity(t.Context(), dev.ID, fmt.Sprintf("10.0.0.%d", i+1), "heartbeat")
		is.NoErr(err)
	}

	// Page 1: limit 2
	req := httptest.NewRequest(http.MethodGet, "/api/v1/address-history?limit=2", nil)
	req.AddCookie(sessionCookie)
	res := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(res, req)
	is.Equal(res.Code, http.StatusOK)

	var page1 httpapi.AddressHistoryResponse
	err = json.NewDecoder(res.Body).Decode(&page1)
	is.NoErr(err)
	is.Equal(len(page1.Events), 2)
	is.True(page1.NextCursor != nil) // more pages
	is.True(page1.TotalEvents >= 5)

	// Page 2: use cursor
	url := fmt.Sprintf("/api/v1/address-history?limit=2&before_id=%d", *page1.NextCursor)
	req = httptest.NewRequest(http.MethodGet, url, nil)
	req.AddCookie(sessionCookie)
	res = httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(res, req)
	is.Equal(res.Code, http.StatusOK)

	var page2 httpapi.AddressHistoryResponse
	err = json.NewDecoder(res.Body).Decode(&page2)
	is.NoErr(err)
	is.Equal(len(page2.Events), 2)
}

func TestHandler_UpdateDevice_RenameAndSetType(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	dev, err := testServer.DeviceService.CreateDevice(t.Context(), testutils.AdminPrincipal(t, testServer), "sensor", nil)
	is.NoErr(err)

	body, _ := json.Marshal(map[string]string{"name": "sensor-renamed", "device_type": "mobile"})
	url := fmt.Sprintf("/api/v1/devices/%d", dev.ID)
	req := httptest.NewRequest(http.MethodPatch, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)
	res := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusOK)

	var updated httpapi.Device
	err = json.NewDecoder(res.Body).Decode(&updated)
	is.NoErr(err)
	is.Equal(updated.Name, "sensor-renamed")
	is.Equal(string(updated.DeviceType), "mobile")
	is.True(updated.Description == nil)
}

func TestHandler_UpdateDevice_SetAndClearDescription(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	dev, err := testServer.DeviceService.CreateDevice(t.Context(), testutils.AdminPrincipal(t, testServer), "noted-device", nil)
	is.NoErr(err)
	url := fmt.Sprintf("/api/v1/devices/%d", dev.ID)

	// Set description
	setBody, _ := json.Marshal(map[string]string{"description": "my note"})
	req := httptest.NewRequest(http.MethodPatch, url, bytes.NewReader(setBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)
	res := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(res, req)
	is.Equal(res.Code, http.StatusOK)

	var withDesc httpapi.Device
	err = json.NewDecoder(res.Body).Decode(&withDesc)
	is.NoErr(err)
	is.True(withDesc.Description != nil)
	is.Equal(*withDesc.Description, "my note")

	// Clear description via explicit null
	clearBody := []byte(`{"description":null}`)
	req2 := httptest.NewRequest(http.MethodPatch, url, bytes.NewReader(clearBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.AddCookie(sessionCookie)
	res2 := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(res2, req2)
	is.Equal(res2.Code, http.StatusOK)

	var cleared httpapi.Device
	err = json.NewDecoder(res2.Body).Decode(&cleared)
	is.NoErr(err)
	is.True(cleared.Description == nil)
}

func TestHandler_UpdateDevice_InvalidType_Returns400(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	dev, err := testServer.DeviceService.CreateDevice(t.Context(), testutils.AdminPrincipal(t, testServer), "type-test", nil)
	is.NoErr(err)

	body, _ := json.Marshal(map[string]string{"device_type": "robot"})
	url := fmt.Sprintf("/api/v1/devices/%d", dev.ID)
	req := httptest.NewRequest(http.MethodPatch, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)
	res := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusBadRequest)
}

func TestHandler_UpdateDevice_DuplicateName_Returns409(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	_, err := testServer.DeviceService.CreateDevice(t.Context(), testutils.AdminPrincipal(t, testServer), "taken", nil)
	is.NoErr(err)
	dev2, err := testServer.DeviceService.CreateDevice(t.Context(), testutils.AdminPrincipal(t, testServer), "to-rename", nil)
	is.NoErr(err)

	body, _ := json.Marshal(map[string]string{"name": "taken"})
	url := fmt.Sprintf("/api/v1/devices/%d", dev2.ID)
	req := httptest.NewRequest(http.MethodPatch, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)
	res := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusConflict)
}

func TestHandler_UpdateDevice_NotFound_Returns404(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	body, _ := json.Marshal(map[string]string{"name": "ghost"})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/devices/9999", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)
	res := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusNotFound)
}

func TestHandler_ListDeviceTypes(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/device-types", nil)
	req.AddCookie(sessionCookie)
	res := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusOK)

	var types []httpapi.DeviceTypeItem
	err := json.NewDecoder(res.Body).Decode(&types)
	is.NoErr(err)
	is.Equal(len(types), 2)
	is.Equal(types[0].Value, "static")
	is.Equal(types[0].Label, "Static")
	is.Equal(types[1].Value, "mobile")
}
