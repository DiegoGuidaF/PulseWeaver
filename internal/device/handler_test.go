package device_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"forgejo.wally.mywire.org/diego/WallyDic.git/api"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/auth"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/config"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/database"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/httpserver"
	"github.com/matryer/is"
)

type integrationServer struct {
	httpServer    http.Handler
	deviceService *device.Service
}

func setupIntegrationServer(t *testing.T) integrationServer {
	t.Helper()

	conf := config.Conf{
		Server: config.ConfServer{
			Port:          2000,
			AdminPassword: "AdminPass123!",
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

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	authRepo := auth.NewRepository(db.DB())
	authService := auth.NewService(authRepo, logger)
	if err := authService.BootstrapAdmin(context.Background(), conf.Server); err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}
	authHandler := auth.NewHandler(authService, logger)

	deviceRepo := device.NewRepository(db.DB())
	deviceService := device.NewService(deviceRepo)
	deviceHandler := device.NewOpenApiHandler(deviceService, logger)

	return integrationServer{
		httpServer:    httpserver.NewServer(deviceHandler, authHandler, logger),
		deviceService: deviceService,
	}
}

func loginCookie(t *testing.T, server http.Handler, username, password string) *http.Cookie {
	t.Helper()

	body, _ := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("login failed with status %d", w.Code)
	}

	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie from login")
	}

	return cookies[0]
}

func TestHandler_CreateAndListDevices_HappyPath(t *testing.T) {
	is := is.New(t)
	testServer := setupIntegrationServer(t)
	sessionCookie := loginCookie(t, testServer.httpServer, "admin", "AdminPass123!")

	createBody, _ := json.Marshal(map[string]string{"name": "bedroom-sensor"})
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/devices", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.AddCookie(sessionCookie)
	createRes := httptest.NewRecorder()
	testServer.httpServer.ServeHTTP(createRes, createReq)
	is.Equal(createRes.Code, http.StatusCreated)

	var created api.Device
	err := json.NewDecoder(createRes.Body).Decode(&created)
	is.NoErr(err)
	is.Equal(created.Name, "bedroom-sensor")

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	listReq.AddCookie(sessionCookie)
	listRes := httptest.NewRecorder()
	testServer.httpServer.ServeHTTP(listRes, listReq)
	is.Equal(listRes.Code, http.StatusOK)

	var devices []api.Device
	err = json.NewDecoder(listRes.Body).Decode(&devices)
	is.NoErr(err)
	is.Equal(len(devices), 1)
	is.Equal(devices[0].Id, created.Id)
}

func TestHandler_AddressLifecycle_HappyPath(t *testing.T) {
	is := is.New(t)
	testServer := setupIntegrationServer(t)
	sessionCookie := loginCookie(t, testServer.httpServer, "admin", "AdminPass123!")

	dev, err := testServer.deviceService.CreateDevice(t.Context(), "router")
	is.NoErr(err)

	addBody, _ := json.Marshal(map[string]string{"ip": "192.168.1.100"})
	addURL := fmt.Sprintf("/api/v1/devices/%d/addresses", dev.ID)
	addReq := httptest.NewRequest(http.MethodPost, addURL, bytes.NewReader(addBody))
	addReq.Header.Set("Content-Type", "application/json")
	addReq.AddCookie(sessionCookie)
	addRes := httptest.NewRecorder()
	testServer.httpServer.ServeHTTP(addRes, addReq)
	is.Equal(addRes.Code, http.StatusCreated)

	var createdAddress api.Address
	err = json.NewDecoder(addRes.Body).Decode(&createdAddress)
	is.NoErr(err)
	is.True(createdAddress.Status)

	listReq := httptest.NewRequest(http.MethodGet, addURL, nil)
	listReq.AddCookie(sessionCookie)
	listRes := httptest.NewRecorder()
	testServer.httpServer.ServeHTTP(listRes, listReq)
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
	testServer.httpServer.ServeHTTP(disableRes, disableReq)
	is.Equal(disableRes.Code, http.StatusOK)

	var disabled api.Address
	err = json.NewDecoder(disableRes.Body).Decode(&disabled)
	is.NoErr(err)
	is.True(!disabled.Status)
}

func TestHandler_DeviceHeartbeat_HappyPath(t *testing.T) {
	is := is.New(t)
	testServer := setupIntegrationServer(t)
	sessionCookie := loginCookie(t, testServer.httpServer, "admin", "AdminPass123!")

	dev, err := testServer.deviceService.CreateDevice(t.Context(), "checkin-device")
	is.NoErr(err)

	heartbeatURL := fmt.Sprintf("/api/v1/devices/%d/heartbeat", dev.ID)

	firstReq := httptest.NewRequest(http.MethodPost, heartbeatURL, nil)
	firstReq.RemoteAddr = "192.168.1.50:12345"
	firstReq.AddCookie(sessionCookie)
	firstRes := httptest.NewRecorder()
	testServer.httpServer.ServeHTTP(firstRes, firstReq)
	is.Equal(firstRes.Code, http.StatusCreated)

	secondReq := httptest.NewRequest(http.MethodPost, heartbeatURL, nil)
	secondReq.RemoteAddr = "192.168.1.50:54321"
	secondReq.AddCookie(sessionCookie)
	secondRes := httptest.NewRecorder()
	testServer.httpServer.ServeHTTP(secondRes, secondReq)
	is.Equal(secondRes.Code, http.StatusOK)
}
