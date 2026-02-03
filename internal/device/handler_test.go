package device_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/config"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/database"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/httpserver"
	"github.com/matryer/is"
)

func setupTestServer(t *testing.T) http.Handler {
	t.Helper()

	// Use in-memory SQLite
	// Set Dsn conf variable to easily override prod parameters
	// Test name is set so that each test gets a new DB - test isolation
	conf := &config.ConfDB{
		Dsn:   fmt.Sprintf("file:%s?mode=memory", t.Name()),
		Debug: false,
	}

	db, err := database.NewSQLite(conf)
	if err != nil {
		t.Fatalf("setup db: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	if err := db.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Wire up layers
	deviceRepo := device.NewRepository(db)
	deviceService := device.NewService(deviceRepo)
	deviceHandler := device.NewHandler(deviceService)

	return httpserver.NewServer(deviceHandler)
}

func TestCreateDevice(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			// Marshal request body
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/devices", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			// Record response
			w := httptest.NewRecorder()
			srv.ServeHTTP(w, req)

			// Assert status
			is.Equal(w.Code, tt.wantStatus)

			// Assert error message if expected
			if tt.wantError != "" {
				var resp map[string]string
				json.NewDecoder(w.Body).Decode(&resp)
				is.Equal(resp["error"], tt.wantError)
			}

			// Assert success response shape
			if tt.wantStatus == http.StatusCreated {
				var dev device.Device
				err := json.NewDecoder(w.Body).Decode(&dev)
				is.NoErr(err)

				is.True(dev.ID != "") // Should be set
				is.Equal(dev.Name, tt.body["name"])
				is.True(!dev.CreatedAt.IsZero()) // timestamp is set
			}
		})
	}
}

func TestGetDevices(t *testing.T) {
	is := is.New(t)

	srv := setupTestServer(t)

	// Create test data
	createDevice(t, srv, "device-1")
	createDevice(t, srv, "device-2")

	// Test GET
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)

	var devices []device.Device
	err := json.NewDecoder(w.Body).Decode(&devices)
	is.NoErr(err)

	is.Equal(len(devices), 2)
}

// Helper to create device in tests
func createDevice(t *testing.T, srv http.Handler, name string) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"name": name})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create device failed: status %d", w.Code)
	}
}
