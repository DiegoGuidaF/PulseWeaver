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

	deviceRepo := device.NewRepository(db)
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
	nonExistentDeviceId := device.DeviceId(123544)

	tests := []struct {
		name       string
		deviceID   device.DeviceId
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
				is.True(address.DisabledAt == nil)
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
		is.True(addr.DisabledAt == nil)
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

func TestHandler_ListDeviceIPs_OnlyActiveIPsReturned(t *testing.T) {
	is := is.New(t)

	testServer := setupTestServer(t)
	device1, _ := testServer.deviceService.CreateDevice(t.Context(), "device-1")

	// Assign IPs
	deviceIp1, _ := testServer.deviceService.AssignAddress(t.Context(), device1.ID, "10.0.0.1")
	testServer.deviceService.AssignAddress(t.Context(), device1.ID, "10.0.0.2")

	// Disable one IP
	testServer.deviceService.DisableAddress(t.Context(), device1.ID, deviceIp1.ID)

	// List should only return active address
	url := fmt.Sprintf("/api/v1/devices/%s/addresses", device1.ID)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()
	testServer.httpServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)

	var addresses []api.Address
	err := json.NewDecoder(w.Body).Decode(&addresses)
	is.NoErr(err)

	is.Equal(len(addresses), 1)
	is.Equal(addresses[0].IP, "10.0.0.2")
}

func TestHandler_DisableDeviceIP(t *testing.T) {
	is := is.New(t)

	testServer := setupTestServer(t)
	device1, _ := testServer.deviceService.CreateDevice(t.Context(), "device-1")
	deviceIp1, _ := testServer.deviceService.AssignAddress(t.Context(), device1.ID, "10.0.0.1")

	url := fmt.Sprintf("/api/v1/devices/%s/addresses/%d", device1.ID, deviceIp1.ID)
	req := httptest.NewRequest(http.MethodDelete, url, nil)
	w := httptest.NewRecorder()
	testServer.httpServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)

	var address api.Address
	json.NewDecoder(w.Body).Decode(&address)
	is.True(address.DisabledAt != nil)
}

func TestHandler_DisableDeviceIP_AlreadyDisabled(t *testing.T) {
	is := is.New(t)

	testServer := setupTestServer(t)
	device1, _ := testServer.deviceService.CreateDevice(t.Context(), "device-1")
	deviceIp1, _ := testServer.deviceService.AssignAddress(t.Context(), device1.ID, "10.0.0.1")

	// Disable once
	testServer.deviceService.DisableAddress(t.Context(), device1.ID, deviceIp1.ID)

	// Try to disable again
	url := fmt.Sprintf("/api/v1/devices/%s/addresses/%d", device1.ID, deviceIp1.ID)
	req := httptest.NewRequest(http.MethodDelete, url, nil)
	w := httptest.NewRecorder()
	testServer.httpServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusNotFound)
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
	deviceIp2, _ := testServer.deviceService.AssignAddress(t.Context(), device2.ID, "10.0.0.1")

	// Try to disable device 2's address using device 1's ID
	url := fmt.Sprintf("/api/v1/devices/%s/addresses/%d", device1.ID, deviceIp2.ID)
	req := httptest.NewRequest(http.MethodDelete, url, nil)
	w := httptest.NewRecorder()
	testServer.httpServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusNotFound)
}
