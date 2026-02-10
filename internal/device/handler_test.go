package device_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"forgejo.wally.mywire.org/diego/WallyDic.git/api"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/config"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/database"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/httpserver"
	"github.com/matryer/is"
)

type TestServer struct {
	deviceService *device.Service
	httpServer    http.Handler
}

func setupTestServer(t *testing.T) TestServer {
	t.Helper()

	conf := config.Conf{
		Server: config.ConfServer{
			Port: 2000,
		},
		DB: config.ConfDB{
			Dsn:   fmt.Sprintf("file:%s?mode=memory&_loc=auto", t.Name()),
			Debug: false,
		},
	}

	db, err := database.NewSQLite(conf.DB)
	if err != nil {
		t.Fatalf("setup db: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	if err := db.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	deviceRepo := device.NewRepository(db.DB())
	deviceService := device.NewService(deviceRepo)
	deviceHandler := device.NewOpenApiHandler(deviceService, logger)

	httpServer := httpserver.NewServer(deviceHandler, logger)
	return TestServer{
		deviceService: deviceService,
		httpServer:    httpServer,
	}
}

func TestHandler_CreateDevice(t *testing.T) {
	testServer := setupTestServer(t)

	tests := []struct {
		name       string
		body       map[string]string
		wantStatus int
	}{
		{
			name:       "valid device",
			body:       map[string]string{"name": "bedroom-sensor"},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "empty name",
			body:       map[string]string{"name": ""},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing name field",
			body:       map[string]string{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "valid device with special characters",
			body:       map[string]string{"name": "device-123_test"},
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/devices", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			testServer.httpServer.ServeHTTP(w, req)

			is.Equal(w.Code, tt.wantStatus)

			if tt.wantStatus == http.StatusCreated {
				var dev api.Device
				err := json.NewDecoder(w.Body).Decode(&dev)
				is.NoErr(err)

				//is.True(check.NotNil(dev.ID))
				is.Equal(dev.Name, tt.body["name"])
				is.True(!dev.CreatedAt.IsZero())
			}
		})
	}
}

func TestHandler_GetDevices(t *testing.T) {
	is := is.New(t)

	testServer := setupTestServer(t)

	testServer.deviceService.CreateDevice(t.Context(), "device-1")
	testServer.deviceService.CreateDevice(t.Context(), "device-2")
	// Create test data

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	w := httptest.NewRecorder()
	testServer.httpServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)

	var devices []api.Device
	err := json.NewDecoder(w.Body).Decode(&devices)
	is.NoErr(err)

	is.Equal(len(devices), 2)
}

func TestHandler_GetDevices_EmptyList(t *testing.T) {
	is := is.New(t)

	testServer := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	w := httptest.NewRecorder()
	testServer.httpServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)

	var devices []api.Device
	err := json.NewDecoder(w.Body).Decode(&devices)
	is.NoErr(err)

	is.Equal(len(devices), 0)
}

func TestHandler_AssignIP(t *testing.T) {
	testServer := setupTestServer(t)

	device1, _ := testServer.deviceService.CreateDevice(t.Context(), "device1-1")
	deviceId := device1.ID
	nonExistentDeviceId := device.DeviceID(123544)

	tests := []struct {
		name       string
		deviceID   device.DeviceID
		body       map[string]string
		wantStatus int
		wantError  string
	}{
		{
			name:       "valid IPv4",
			deviceID:   deviceId,
			body:       map[string]string{"ip": "192.168.1.100"},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "valid IPv6",
			deviceID:   deviceId,
			body:       map[string]string{"ip": "2001:db8::68"},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid IP format",
			deviceID:   deviceId,
			body:       map[string]string{"ip": "not-an-ip"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty IP",
			deviceID:   deviceId,
			body:       map[string]string{"ip": ""},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing IP field",
			deviceID:   deviceId,
			body:       map[string]string{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "device1 not found",
			deviceID:   nonExistentDeviceId,
			body:       map[string]string{"ip": "192.168.1.1"},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)

			body, _ := json.Marshal(tt.body)
			url := fmt.Sprintf("/api/v1/devices/%s/addresses", tt.deviceID)
			req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			testServer.httpServer.ServeHTTP(w, req)

			is.Equal(w.Code, tt.wantStatus)

			if tt.wantStatus == http.StatusCreated {
				var address api.Address
				err := json.NewDecoder(w.Body).Decode(&address)
				is.NoErr(err)

				is.True(address.ID != 0)
				is.Equal(address.DeviceId, tt.deviceID.Int64())
				is.Equal(address.IP, tt.body["ip"])
				is.True(address.Status) // Address should be enabled when created
				is.True(!address.CreatedAt.IsZero())
			}
		})
	}
}

func TestHandler_AssignIP_SameIPToMultipleDevices(t *testing.T) {
	is := is.New(t)

	testServer := setupTestServer(t)

	device1, _ := testServer.deviceService.CreateDevice(t.Context(), "device-1")
	device2, _ := testServer.deviceService.CreateDevice(t.Context(), "device-2")

	testServer.deviceService.AssignAddress(t.Context(), device1.ID, "10.0.0.1")

	// Assign same IP to device 2 - should succeed
	body, _ := json.Marshal(map[string]string{"ip": "10.0.0.1"})
	url := fmt.Sprintf("/api/v1/devices/%d/addresses", device2.ID)
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	testServer.httpServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusCreated)
}

func TestHandler_ListDeviceIPs(t *testing.T) {
	is := is.New(t)

	testServer := setupTestServer(t)

	device1, _ := testServer.deviceService.CreateDevice(t.Context(), "device-1")

	// Assign multiple IPs
	testServer.deviceService.AssignAddress(t.Context(), device1.ID, "10.0.0.1")
	testServer.deviceService.AssignAddress(t.Context(), device1.ID, "10.0.0.2")
	testServer.deviceService.AssignAddress(t.Context(), device1.ID, "10.0.0.3")

	url := fmt.Sprintf("/api/v1/devices/%d/addresses", device1.ID)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()
	testServer.httpServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)

	var addresses []api.Address
	err := json.NewDecoder(w.Body).Decode(&addresses)
	is.NoErr(err)

	is.Equal(len(addresses), 3)

	// Verify all addresses are active
	for _, addr := range addresses {
		is.True(addr.Status) // All addresses should be enabled
	}
}

func TestHandler_ListDeviceIPs_EmptyList(t *testing.T) {
	is := is.New(t)

	testServer := setupTestServer(t)

	device1, _ := testServer.deviceService.CreateDevice(t.Context(), "device-1")

	url := fmt.Sprintf("/api/v1/devices/%d/addresses", device1.ID)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()
	testServer.httpServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)

	var addresses []api.Address
	err := json.NewDecoder(w.Body).Decode(&addresses)
	is.NoErr(err)

	is.Equal(len(addresses), 0)
}

func TestHandler_ListDeviceIPs_DeviceNotFound(t *testing.T) {
	is := is.New(t)

	testServer := setupTestServer(t)

	url := "/api/v1/devices/12346/addresses"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()
	testServer.httpServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusNotFound)
}

func TestHandler_ListDeviceIPs_AllAddressesReturned(t *testing.T) {
	is := is.New(t)

	testServer := setupTestServer(t)
	device1, _ := testServer.deviceService.CreateDevice(t.Context(), "device-1")

	// Assign IPs
	deviceIp1, _, _ := testServer.deviceService.AssignAddress(t.Context(), device1.ID, "10.0.0.1")
	testServer.deviceService.AssignAddress(t.Context(), device1.ID, "10.0.0.2")

	// Disable one IP
	testServer.deviceService.DisableAddress(t.Context(), device1.ID, deviceIp1.AddressId)

	// List should return all addresses (enabled and disabled)
	url := fmt.Sprintf("/api/v1/devices/%d/addresses", device1.ID)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()
	testServer.httpServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)

	var addresses []api.Address
	err := json.NewDecoder(w.Body).Decode(&addresses)
	is.NoErr(err)

	is.Equal(len(addresses), 2) // Both addresses should be returned

	// Verify status of addresses
	var disabledAddr, enabledAddr *api.Address
	for i := range addresses {
		if addresses[i].IP == "10.0.0.1" {
			disabledAddr = &addresses[i]
		} else if addresses[i].IP == "10.0.0.2" {
			enabledAddr = &addresses[i]
		}
	}
	is.True(disabledAddr != nil)
	is.True(enabledAddr != nil)
	is.True(!disabledAddr.Status) // First address should be disabled
	is.True(enabledAddr.Status)   // Second address should be enabled
}

func TestHandler_DisableDeviceIP(t *testing.T) {
	is := is.New(t)

	testServer := setupTestServer(t)
	device1, _ := testServer.deviceService.CreateDevice(t.Context(), "device-1")
	deviceIp1, _, _ := testServer.deviceService.AssignAddress(t.Context(), device1.ID, "10.0.0.1")

	url := fmt.Sprintf("/api/v1/devices/%d/addresses/%d", device1.ID, deviceIp1.AddressId)
	req := httptest.NewRequest(http.MethodDelete, url, nil)
	w := httptest.NewRecorder()
	testServer.httpServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)

	var address api.Address
	err := json.NewDecoder(w.Body).Decode(&address)
	is.NoErr(err)
	is.True(!address.Status) // Address should be disabled
}

func TestHandler_DisableDeviceIP_AlreadyDisabled_IsOk(t *testing.T) {
	is := is.New(t)

	testServer := setupTestServer(t)
	device1, _ := testServer.deviceService.CreateDevice(t.Context(), "device-1")
	deviceIp1, _, _ := testServer.deviceService.AssignAddress(t.Context(), device1.ID, "10.0.0.1")

	// Disable once
	testServer.deviceService.DisableAddress(t.Context(), device1.ID, deviceIp1.AddressId)

	// Try to disable again
	url := fmt.Sprintf("/api/v1/devices/%d/addresses/%d", device1.ID, deviceIp1.AddressId)
	req := httptest.NewRequest(http.MethodDelete, url, nil)
	w := httptest.NewRecorder()
	testServer.httpServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)
}

func TestHandler_DisableDeviceIP_IPNotFound(t *testing.T) {
	is := is.New(t)

	testServer := setupTestServer(t)

	device1, _ := testServer.deviceService.CreateDevice(t.Context(), "device-1")

	url := fmt.Sprintf("/api/v1/devices/%s/addresses/99999", device1.ID)
	req := httptest.NewRequest(http.MethodDelete, url, nil)
	w := httptest.NewRecorder()
	testServer.httpServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusNotFound)
}

func TestHandler_DisableDeviceIP_WrongDevice(t *testing.T) {
	is := is.New(t)

	testServer := setupTestServer(t)

	device1, _ := testServer.deviceService.CreateDevice(t.Context(), "device-1")
	device2, _ := testServer.deviceService.CreateDevice(t.Context(), "device-2")

	// Assign IP to device 2
	deviceIp2, _, _ := testServer.deviceService.AssignAddress(t.Context(), device2.ID, "10.0.0.1")

	// Try to disable device 2's address using device 1's ID
	url := fmt.Sprintf("/api/v1/devices/%d/addresses/%d", device1.ID, deviceIp2.AddressId)
	req := httptest.NewRequest(http.MethodDelete, url, nil)
	w := httptest.NewRecorder()
	testServer.httpServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusNotFound)
}

func TestHandler_CheckinDevice_NewAddress(t *testing.T) {
	is := is.New(t)

	testServer := setupTestServer(t)

	dev, _ := testServer.deviceService.CreateDevice(t.Context(), "checkin-device")

	url := fmt.Sprintf("/api/v1/devices/%d/heartbeat", dev.ID)
	req := httptest.NewRequest(http.MethodPost, url, nil)
	req.RemoteAddr = "192.168.1.50:12345"

	w := httptest.NewRecorder()
	testServer.httpServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusCreated)

	var address api.Address
	err := json.NewDecoder(w.Body).Decode(&address)
	is.NoErr(err)

	is.True(address.ID != 0)
	is.Equal(address.DeviceId, dev.ID.Int64())
	is.Equal(address.IP, "192.168.1.50")
	is.True(address.Status) // Address should be enabled
	is.True(!address.CreatedAt.IsZero())
}

func TestHandler_CheckinDevice_ExistingAddress(t *testing.T) {
	is := is.New(t)

	testServer := setupTestServer(t)

	dev, _ := testServer.deviceService.CreateDevice(t.Context(), "checkin-device")

	// First assign creates the address
	_, _, _ = testServer.deviceService.AssignAddress(t.Context(), dev.ID, "10.0.0.5")

	// Checkin with same IP should return 200
	url := fmt.Sprintf("/api/v1/devices/%d/heartbeat", dev.ID)
	req := httptest.NewRequest(http.MethodPost, url, nil)
	req.RemoteAddr = "10.0.0.5:9999"

	w := httptest.NewRecorder()
	testServer.httpServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)

	var address api.Address
	err := json.NewDecoder(w.Body).Decode(&address)
	is.NoErr(err)

	is.Equal(address.IP, "10.0.0.5")
	is.True(address.Status) // Address should be enabled
}

func TestHandler_CheckinDevice_ReEnableDisabledAddress(t *testing.T) {
	is := is.New(t)

	testServer := setupTestServer(t)

	dev, _ := testServer.deviceService.CreateDevice(t.Context(), "checkin-device")

	// Create and then disable an address
	addr, _, _ := testServer.deviceService.AssignAddress(t.Context(), dev.ID, "10.0.0.10")
	testServer.deviceService.DisableAddress(t.Context(), dev.ID, addr.AddressId)

	time.Sleep(2 * time.Millisecond) // Wait for SQLite resolution

	// Checkin with the same IP should re-enable it (200)
	url := fmt.Sprintf("/api/v1/devices/%d/heartbeat", dev.ID)
	req := httptest.NewRequest(http.MethodPost, url, nil)
	req.RemoteAddr = "10.0.0.10:8080"

	w := httptest.NewRecorder()
	testServer.httpServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)

	var address api.Address
	err := json.NewDecoder(w.Body).Decode(&address)
	is.NoErr(err)

	is.Equal(address.IP, "10.0.0.10")
	is.True(address.Status) // Address should be re-enabled
}

func TestHandler_CheckinDevice_DeviceNotFound(t *testing.T) {
	is := is.New(t)

	testServer := setupTestServer(t)

	url := "/api/v1/devices/99999/heartbeat"
	req := httptest.NewRequest(http.MethodPost, url, nil)
	req.RemoteAddr = "192.168.1.1:5555"

	w := httptest.NewRecorder()
	testServer.httpServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusNotFound)
}

func TestHandler_CheckinDevice_IPv6Accepted(t *testing.T) {
	is := is.New(t)

	testServer := setupTestServer(t)

	dev, _ := testServer.deviceService.CreateDevice(t.Context(), "checkin-device")

	url := fmt.Sprintf("/api/v1/devices/%d/heartbeat", dev.ID)
	req := httptest.NewRequest(http.MethodPost, url, nil)
	req.RemoteAddr = "[2001:db8::1]:12345"

	w := httptest.NewRecorder()
	testServer.httpServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusCreated)

	var address api.Address
	err := json.NewDecoder(w.Body).Decode(&address)
	is.NoErr(err)

	is.Equal(address.IP, "2001:db8::1")
	is.True(address.Status) // Address should be enabled
}

func TestHandler_CheckinDevice_ClientIPExtractionFails(t *testing.T) {
	is := is.New(t)

	testServer := setupTestServer(t)

	dev, _ := testServer.deviceService.CreateDevice(t.Context(), "checkin-device")

	// Create request with empty RemoteAddr - middleware will set empty string in context
	url := fmt.Sprintf("/api/v1/devices/%d/heartbeat", dev.ID)
	req := httptest.NewRequest(http.MethodPost, url, nil)
	req.RemoteAddr = "" // Empty RemoteAddr should cause IP extraction to fail

	w := httptest.NewRecorder()
	testServer.httpServer.ServeHTTP(w, req)

	// HTTPHandler should return 400 when IP extraction fails
	is.Equal(w.Code, http.StatusBadRequest)

	var errorResp api.ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&errorResp)
	is.NoErr(err)
	is.True(errorResp.Error != nil)
	is.True(*errorResp.Error != "")
}
