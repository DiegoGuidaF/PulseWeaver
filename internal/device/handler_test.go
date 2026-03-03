//go:build test

package device_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/httpapi"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/testutils"
	"github.com/matryer/is"
)

func TestHandler_CreateAndListDevices(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	device, _, err := testServer.DeviceService.CreateDevice(t.Context(), "bedroom-sensor")
	is.NoErr(err)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	listReq.AddCookie(sessionCookie)
	listRes := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(listRes, listReq)
	is.Equal(listRes.Code, http.StatusOK)

	var devices []httpapi.Device
	err = json.NewDecoder(listRes.Body).Decode(&devices)
	is.NoErr(err)
	is.Equal(len(devices), 1)
	is.Equal(devices[0].Id, int64(device.ID))
	is.True(devices[0].ApiKeyPrefix != "")
}

func TestHandler_AddressLifecycle(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	dev, _, err := testServer.DeviceService.CreateDevice(t.Context(), "router")
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
	is.True(createdAddress.Status)

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
	is.True(addresses[0].Status)

	disableURL := fmt.Sprintf("/api/v1/devices/%d/addresses/%d", dev.ID, createdAddress.Id)
	disableReq := httptest.NewRequest(http.MethodDelete, disableURL, nil)
	disableReq.AddCookie(sessionCookie)
	disableRes := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(disableRes, disableReq)
	is.Equal(disableRes.Code, http.StatusOK)

	var disabled httpapi.Address
	err = json.NewDecoder(disableRes.Body).Decode(&disabled)
	is.NoErr(err)
	is.True(!disabled.Status)
}

func TestHandler_DeviceHeartbeat(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	dev, _, err := testServer.DeviceService.CreateDevice(t.Context(), "checkin-device")
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

	var resp httpapi.CreateDeviceResponse
	err := json.NewDecoder(createRes.Body).Decode(&resp)
	is.NoErr(err)
	is.Equal(resp.Device.Name, "sensor-1")
	is.True(resp.Device.Id != 0)
	is.True(resp.ApiKey != "")
}

func TestHandler_GetDevice_200(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	device, _, err := testServer.DeviceService.CreateDevice(t.Context(), "single-device")
	is.NoErr(err)

	getURL := fmt.Sprintf("/api/v1/devices/%d", device.ID)
	getReq := httptest.NewRequest(http.MethodGet, getURL, nil)
	getReq.AddCookie(sessionCookie)
	getRes := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(getRes, getReq)
	is.Equal(getRes.Code, http.StatusOK)

	var resp httpapi.Device
	err = json.NewDecoder(getRes.Body).Decode(&resp)
	is.NoErr(err)
	is.Equal(resp.Id, int64(device.ID))
	is.Equal(resp.Name, device.Name)
	is.True(resp.ApiKeyPrefix != "")
}

func TestHandler_GetDevice_404(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	getURL := fmt.Sprintf("/api/v1/devices/%d", 99999)
	getReq := httptest.NewRequest(http.MethodGet, getURL, nil)
	getReq.AddCookie(sessionCookie)
	getRes := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(getRes, getReq)
	is.Equal(getRes.Code, http.StatusNotFound)
}

func TestHandler_GetDevices_ReturnsAPIKeyPrefix(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	_, _, err := testServer.DeviceService.CreateDevice(t.Context(), "listed-device")
	is.NoErr(err)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	listReq.AddCookie(sessionCookie)
	listRes := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(listRes, listReq)
	is.Equal(listRes.Code, http.StatusOK)

	var devices []httpapi.Device
	err = json.NewDecoder(listRes.Body).Decode(&devices)
	is.NoErr(err)
	is.Equal(len(devices), 1)
	is.True(devices[0].ApiKeyPrefix != "")
}

func TestHandler_DeviceHeartbeatByApiKey_NoBody(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)

	_, apiKey, err := testServer.DeviceService.CreateDevice(t.Context(), "apikey-heartbeat-device")
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
	is.True(addr.Status)
}

func TestHandler_DeviceHeartbeatByApiKey_WithBodyIP(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)

	_, apiKey, err := testServer.DeviceService.CreateDevice(t.Context(), "apikey-heartbeat-with-body-ip")
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
	is.True(addr.Status)
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

	device, _, err := testServer.DeviceService.CreateDevice(t.Context(), "to-delete")
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
	var devices []httpapi.Device
	err = json.NewDecoder(listRes.Body).Decode(&devices)
	is.NoErr(err)
	is.Equal(len(devices), 0)
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

func TestHandler_CreateDevice_409_DuplicateName(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	// Create first device via service so name is taken
	_, _, err := testServer.DeviceService.CreateDevice(t.Context(), "dup-name")
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
