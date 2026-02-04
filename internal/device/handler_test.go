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

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/config"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/database"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/httpserver"
	"github.com/matryer/is"
)

func setupTestServer(t *testing.T) http.Handler {
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
	deviceHandler := device.NewHandler(deviceService, logger)

	return httpserver.NewServer(deviceHandler, logger, conf.Server)
}

func TestHandler_CreateDevice(t *testing.T) {
	srv := setupTestServer(t)

	tests := []struct {
		name       string
		body       map[string]string
		wantStatus int
		wantError  string
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
			wantError:  "name is required",
		},
		{
			name:       "name too short",
			body:       map[string]string{"name": "ab"},
			wantStatus: http.StatusBadRequest,
			wantError:  "name must be at least 3 characters",
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
			srv.ServeHTTP(w, req)

			is.Equal(w.Code, tt.wantStatus)

			if tt.wantError != "" {
				var resp map[string]string
				json.NewDecoder(w.Body).Decode(&resp)
				is.Equal(resp["error"], tt.wantError)
			}

			if tt.wantStatus == http.StatusCreated {
				var dev device.Device
				err := json.NewDecoder(w.Body).Decode(&dev)
				is.NoErr(err)

				is.True(dev.ID.String() != "")
				is.Equal(dev.Name, tt.body["name"])
				is.True(!dev.CreatedAt.IsZero())
			}
		})
	}
}

func TestHandler_CreateDevice_InvalidJSON(t *testing.T) {
	is := is.New(t)

	srv := setupTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusBadRequest)

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	is.Equal(resp["error"], "Invalid request body")
}

func TestHandler_GetDevices(t *testing.T) {
	is := is.New(t)

	srv := setupTestServer(t)

	// Create test data
	createDevice(t, srv, "device-1")
	createDevice(t, srv, "device-2")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)

	var devices []device.Device
	err := json.NewDecoder(w.Body).Decode(&devices)
	is.NoErr(err)

	is.Equal(len(devices), 2)
}

func TestHandler_GetDevices_EmptyList(t *testing.T) {
	is := is.New(t)

	srv := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)

	var devices []device.Device
	err := json.NewDecoder(w.Body).Decode(&devices)
	is.NoErr(err)

	is.Equal(len(devices), 0)
}

func TestHandler_AssignIP(t *testing.T) {
	srv := setupTestServer(t)

	deviceID := createDeviceAndGetID(t, srv, "test-device")
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
			deviceID:   deviceID,
			body:       map[string]string{"ip_address": "192.168.1.100"},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "another valid IPv4",
			deviceID:   deviceID,
			body:       map[string]string{"ip_address": "10.0.0.1"},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid IP format",
			deviceID:   deviceID,
			body:       map[string]string{"ip_address": "not-an-ip"},
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid IP address format",
		},
		{
			name:       "IPv6 not supported",
			deviceID:   deviceID,
			body:       map[string]string{"ip_address": "2001:db8::1"},
			wantStatus: http.StatusBadRequest,
			wantError:  "only IPv4 addresses are supported",
		},
		{
			name:       "empty IP",
			deviceID:   deviceID,
			body:       map[string]string{"ip_address": ""},
			wantStatus: http.StatusBadRequest,
			wantError:  "ip_address is required",
		},
		{
			name:       "missing IP field",
			deviceID:   deviceID,
			body:       map[string]string{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "device not found",
			deviceID:   nonExistentDeviceId,
			body:       map[string]string{"ip_address": "192.168.1.1"},
			wantStatus: http.StatusNotFound,
			wantError:  "device not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)

			body, _ := json.Marshal(tt.body)
			url := fmt.Sprintf("/api/v1/devices/%s/ips", tt.deviceID)
			req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			srv.ServeHTTP(w, req)

			is.Equal(w.Code, tt.wantStatus)

			if tt.wantError != "" {
				var resp map[string]string
				json.NewDecoder(w.Body).Decode(&resp)
				is.Equal(resp["error"], tt.wantError)
			}

			if tt.wantStatus == http.StatusCreated {
				var ip device.DeviceIP
				err := json.NewDecoder(w.Body).Decode(&ip)
				is.NoErr(err)

				is.True(ip.ID != 0)
				is.Equal(ip.DeviceID, tt.deviceID)
				is.Equal(ip.IPAddress, tt.body["ip_address"])
				is.True(ip.DisabledAt == nil)
				is.True(!ip.CreatedAt.IsZero())
			}
		})
	}
}

func TestHandler_AssignIP_InvalidJSON(t *testing.T) {
	is := is.New(t)

	srv := setupTestServer(t)
	deviceID := createDeviceAndGetID(t, srv, "test-device")

	url := fmt.Sprintf("/api/v1/devices/%s/ips", deviceID)
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusBadRequest)

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	is.Equal(resp["error"], "Failed to decode request body")
}

func TestHandler_AssignIP_SameIPToMultipleDevices(t *testing.T) {
	is := is.New(t)

	srv := setupTestServer(t)

	device1ID := createDeviceAndGetID(t, srv, "device-1")
	device2ID := createDeviceAndGetID(t, srv, "device-2")

	// Assign same IP to device 1
	assignIP(t, srv, device1ID, "10.0.0.1")

	// Assign same IP to device 2 - should succeed
	body, _ := json.Marshal(map[string]string{"ip_address": "10.0.0.1"})
	url := fmt.Sprintf("/api/v1/devices/%s/ips", device2ID)
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusCreated)
}

func TestHandler_ListDeviceIPs(t *testing.T) {
	is := is.New(t)

	srv := setupTestServer(t)
	deviceID := createDeviceAndGetID(t, srv, "test-device")

	// Assign multiple IPs
	assignIP(t, srv, deviceID, "192.168.1.1")
	assignIP(t, srv, deviceID, "192.168.1.2")
	assignIP(t, srv, deviceID, "192.168.1.3")

	url := fmt.Sprintf("/api/v1/devices/%s/ips", deviceID)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)

	var ips []device.DeviceIP
	err := json.NewDecoder(w.Body).Decode(&ips)
	is.NoErr(err)

	is.Equal(len(ips), 3)

	// Verify all IPs are active
	for _, ip := range ips {
		is.True(ip.DisabledAt == nil)
	}
}

func TestHandler_ListDeviceIPs_EmptyList(t *testing.T) {
	is := is.New(t)

	srv := setupTestServer(t)
	deviceID := createDeviceAndGetID(t, srv, "test-device")

	url := fmt.Sprintf("/api/v1/devices/%s/ips", deviceID)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)

	var ips []device.DeviceIP
	err := json.NewDecoder(w.Body).Decode(&ips)
	is.NoErr(err)

	is.Equal(len(ips), 0)
}

func TestHandler_ListDeviceIPs_DeviceNotFound(t *testing.T) {
	is := is.New(t)

	srv := setupTestServer(t)

	url := "/api/v1/devices/12346/ips"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusNotFound)

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	is.Equal(resp["error"], "device not found")
}

func TestHandler_ListDeviceIPs_OnlyActiveIPsReturned(t *testing.T) {
	is := is.New(t)

	srv := setupTestServer(t)
	deviceID := createDeviceAndGetID(t, srv, "test-device")

	// Assign IPs
	ip1ID := assignIPAndGetID(t, srv, deviceID, "192.168.1.1")
	assignIP(t, srv, deviceID, "192.168.1.2")

	// Disable one IP
	disableIP(t, srv, deviceID, ip1ID)

	// List should only return active IP
	url := fmt.Sprintf("/api/v1/devices/%s/ips", deviceID)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)

	var ips []device.DeviceIP
	err := json.NewDecoder(w.Body).Decode(&ips)
	is.NoErr(err)

	is.Equal(len(ips), 1)
	is.Equal(ips[0].IPAddress, "192.168.1.2")
}

func TestHandler_DisableDeviceIP(t *testing.T) {
	is := is.New(t)

	srv := setupTestServer(t)
	deviceID := createDeviceAndGetID(t, srv, "test-device")
	ipID := assignIPAndGetID(t, srv, deviceID, "192.168.1.100")

	url := fmt.Sprintf("/api/v1/devices/%s/ips/%d/disable", deviceID, ipID)
	req := httptest.NewRequest(http.MethodPatch, url, nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusNoContent)

	// Verify IP is no longer in active list
	listURL := fmt.Sprintf("/api/v1/devices/%s/ips", deviceID)
	listReq := httptest.NewRequest(http.MethodGet, listURL, nil)
	listW := httptest.NewRecorder()
	srv.ServeHTTP(listW, listReq)

	var ips []device.DeviceIP
	json.NewDecoder(listW.Body).Decode(&ips)
	is.Equal(len(ips), 0)
}

func TestHandler_DisableDeviceIP_AlreadyDisabled(t *testing.T) {
	is := is.New(t)

	srv := setupTestServer(t)
	deviceID := createDeviceAndGetID(t, srv, "test-device")
	ipID := assignIPAndGetID(t, srv, deviceID, "192.168.1.100")

	// Disable once
	disableIP(t, srv, deviceID, ipID)

	// Try to disable again
	url := fmt.Sprintf("/api/v1/devices/%s/ips/%d/disable", deviceID, ipID)
	req := httptest.NewRequest(http.MethodPatch, url, nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusConflict)

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	is.Equal(resp["error"], "device IP already disabled")
}

func TestHandler_DisableDeviceIP_IPNotFound(t *testing.T) {
	is := is.New(t)

	srv := setupTestServer(t)
	deviceID := createDeviceAndGetID(t, srv, "test-device")

	url := fmt.Sprintf("/api/v1/devices/%s/ips/99999/disable", deviceID)
	req := httptest.NewRequest(http.MethodPatch, url, nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusNotFound)

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	is.Equal(resp["error"], "device IP not found")
}

func TestHandler_DisableDeviceIP_WrongDevice(t *testing.T) {
	is := is.New(t)

	srv := setupTestServer(t)

	device1ID := createDeviceAndGetID(t, srv, "device-1")
	device2ID := createDeviceAndGetID(t, srv, "device-2")

	// Assign IP to device 2
	ipID := assignIPAndGetID(t, srv, device2ID, "10.0.0.1")

	// Try to disable device 2's IP using device 1's ID
	url := fmt.Sprintf("/api/v1/devices/%s/ips/%d/disable", device1ID, ipID)
	req := httptest.NewRequest(http.MethodPatch, url, nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusNotFound)

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	is.Equal(resp["error"], "device IP does not belong to device")
}

func createDevice(t *testing.T, srv http.Handler, name string) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"name": name})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create device failed: status %d, body: %s", w.Code, w.Body.String())
	}
}

func createDeviceAndGetID(t *testing.T, srv http.Handler, name string) device.DeviceID {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"name": name})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create device failed: status %d", w.Code)
	}

	var dev device.Device
	json.NewDecoder(w.Body).Decode(&dev)
	return dev.ID
}

func assignIP(t *testing.T, srv http.Handler, deviceID device.DeviceID, ipAddress string) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"ip_address": ipAddress})
	url := fmt.Sprintf("/api/v1/devices/%s/ips", deviceID)
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("assign IP failed: status %d, body: %s", w.Code, w.Body.String())
	}
}

func assignIPAndGetID(t *testing.T, srv http.Handler, deviceID device.DeviceID, ipAddress string) device.DeviceIpID {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"ip_address": ipAddress})
	url := fmt.Sprintf("/api/v1/devices/%s/ips", deviceID)
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("assign IP failed: status %d", w.Code)
	}

	var ip device.DeviceIP
	json.NewDecoder(w.Body).Decode(&ip)
	return ip.ID
}

func disableIP(t *testing.T, srv http.Handler, deviceID device.DeviceID, deviceIpID device.DeviceIpID) {
	t.Helper()
	url := fmt.Sprintf("/api/v1/devices/%s/ips/%d/disable", deviceID, deviceIpID)
	req := httptest.NewRequest(http.MethodPatch, url, nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("disable IP failed: status %d, body: %s", w.Code, w.Body.String())
	}
}
