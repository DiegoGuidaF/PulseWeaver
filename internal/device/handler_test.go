package device_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"forgejo.wally.mywire.org/diego/WallyDic.git/api"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/testutils"
	"github.com/matryer/is"
)

func TestHandler_CreateAndListDevices(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	createBody, _ := json.Marshal(map[string]string{"name": "bedroom-sensor"})
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/devices", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.AddCookie(sessionCookie)
	createRes := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(createRes, createReq)
	is.Equal(createRes.Code, http.StatusCreated)

	var createResp api.CreateDeviceResponse
	err := json.NewDecoder(createRes.Body).Decode(&createResp)
	is.NoErr(err)
	is.Equal(createResp.Device.Name, "bedroom-sensor")
	is.True(createResp.ApiKey != "")
	is.True(len(createResp.ApiKey) > 4) // wdk_...

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	listReq.AddCookie(sessionCookie)
	listRes := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(listRes, listReq)
	is.Equal(listRes.Code, http.StatusOK)

	var devices []api.Device
	err = json.NewDecoder(listRes.Body).Decode(&devices)
	is.NoErr(err)
	is.Equal(len(devices), 1)
	is.Equal(devices[0].Id, createResp.Device.Id)
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

	var createdAddress api.Address
	err = json.NewDecoder(addRes.Body).Decode(&createdAddress)
	is.NoErr(err)
	is.True(createdAddress.Status)

	listReq := httptest.NewRequest(http.MethodGet, addURL, nil)
	listReq.AddCookie(sessionCookie)
	listRes := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(listRes, listReq)
	is.Equal(listRes.Code, http.StatusOK)

	var addresses []api.Address
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

	var disabled api.Address
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

	var resp api.CreateDeviceResponse
	err := json.NewDecoder(createRes.Body).Decode(&resp)
	is.NoErr(err)
	is.Equal(resp.Device.Name, "sensor-1")
	is.True(resp.Device.Id != 0)
	is.True(resp.ApiKey != "")
}

func TestHandler_GetDevices_ReturnsApiKeyPrefix(t *testing.T) {
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

	var devices []api.Device
	err = json.NewDecoder(listRes.Body).Decode(&devices)
	is.NoErr(err)
	is.Equal(len(devices), 1)
	is.True(devices[0].ApiKeyPrefix != "")
}

func TestHandler_DeviceHeartbeatByApiKey_NoBody(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	// Create device via HTTP to get api_key in response
	createBody, _ := json.Marshal(map[string]string{"name": "apikey-heartbeat-device"})
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/devices", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.AddCookie(sessionCookie)
	createRes := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(createRes, createReq)
	is.Equal(createRes.Code, http.StatusCreated)

	var createResp api.CreateDeviceResponse
	err := json.NewDecoder(createRes.Body).Decode(&createResp)
	is.NoErr(err)
	is.True(createResp.ApiKey != "")

	// POST /heartbeat with X-API-Key (no session cookie needed)
	// Send empty JSON body, should use client ip from request context
	emptyBody, _ := json.Marshal(map[string]interface{}{})
	heartbeatReq := httptest.NewRequest(http.MethodPost, "/api/v1/heartbeat", bytes.NewReader(emptyBody))
	heartbeatReq.Header.Set("Content-Type", "application/json")
	heartbeatReq.RemoteAddr = "192.168.1.99:12345"
	heartbeatReq.Header.Set("X-API-Key", createResp.ApiKey)
	heartbeatRes := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(heartbeatRes, heartbeatReq)
	is.Equal(heartbeatRes.Code, http.StatusCreated)

	var addr api.Address
	err = json.NewDecoder(heartbeatRes.Body).Decode(&addr)
	is.NoErr(err)
	is.Equal(addr.Ip, "192.168.1.99")
	is.True(addr.Status)
}

func TestHandler_DeviceHeartbeatByApiKey_WithBodyIP(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	// Create device via HTTP to get api_key in response
	createBody, _ := json.Marshal(map[string]string{"name": "apikey-heartbeat-with-body-ip"})
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/devices", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.AddCookie(sessionCookie)
	createRes := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(createRes, createReq)
	is.Equal(createRes.Code, http.StatusCreated)

	var createResp api.CreateDeviceResponse
	err := json.NewDecoder(createRes.Body).Decode(&createResp)
	is.NoErr(err)
	is.True(createResp.ApiKey != "")

	// POST /heartbeat with X-API-Key and IP in request body
	// The IP in body should be used instead of RemoteAddr
	heartbeatBody, _ := json.Marshal(map[string]string{"ip": "10.0.0.42"})
	heartbeatReq := httptest.NewRequest(http.MethodPost, "/api/v1/heartbeat", bytes.NewReader(heartbeatBody))
	heartbeatReq.Header.Set("Content-Type", "application/json")
	heartbeatReq.RemoteAddr = "192.168.1.99:12345" // This should be ignored when body IP is provided
	heartbeatReq.Header.Set("X-API-Key", createResp.ApiKey)
	heartbeatRes := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(heartbeatRes, heartbeatReq)
	is.Equal(heartbeatRes.Code, http.StatusCreated)

	var addr api.Address
	err = json.NewDecoder(heartbeatRes.Body).Decode(&addr)
	is.NoErr(err)
	// Verify the IP from body is used, not RemoteAddr
	is.Equal(addr.Ip, "10.0.0.42")
	is.True(addr.Status)

	// Test case: Second heartbeat with same body IP should return 200 (address already exists)
	heartbeatBody2, _ := json.Marshal(map[string]string{"ip": "10.0.0.42"})
	heartbeatReq2 := httptest.NewRequest(http.MethodPost, "/api/v1/heartbeat", bytes.NewReader(heartbeatBody2))
	heartbeatReq2.Header.Set("Content-Type", "application/json")
	heartbeatReq2.RemoteAddr = "192.168.1.99:0" // This should be ignored
	heartbeatReq2.Header.Set("X-API-Key", createResp.ApiKey)
	heartbeatRes2 := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(heartbeatRes2, heartbeatReq2)
	is.Equal(heartbeatRes2.Code, http.StatusOK) // Should return 200 since address already exists

	var addr2 api.Address
	err = json.NewDecoder(heartbeatRes2.Body).Decode(&addr2)
	is.NoErr(err)
	is.Equal(addr2.Ip, "10.0.0.42") // Should use the same IP from body
	is.True(addr2.Status)

	// Test case: Third heartbeat with different body IP should create a new address (201)
	heartbeatBody3, _ := json.Marshal(map[string]string{"ip": "10.0.0.43"})
	heartbeatReq3 := httptest.NewRequest(http.MethodPost, "/api/v1/heartbeat", bytes.NewReader(heartbeatBody3))
	heartbeatReq3.Header.Set("Content-Type", "application/json")
	heartbeatReq3.RemoteAddr = "192.168.1.99:0" // This should be ignored
	heartbeatReq3.Header.Set("X-API-Key", createResp.ApiKey)
	heartbeatRes3 := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(heartbeatRes3, heartbeatReq3)
	is.Equal(heartbeatRes3.Code, http.StatusCreated) // Should return 201 since it's a new IP address

	var addr3 api.Address
	err = json.NewDecoder(heartbeatRes3.Body).Decode(&addr3)
	is.NoErr(err)
	is.Equal(addr3.Ip, "10.0.0.43") // Should use the new IP from body
	is.True(addr3.Status)
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
