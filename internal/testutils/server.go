//go:build test

package testutils

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/auth"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/config"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/database"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/httpserver"
)

// IntegrationServer holds the HTTP server and services for integration testing.
type IntegrationServer struct {
	HTTPServer    http.Handler
	DeviceService *device.Service
	AuthService   *auth.Service
}

// SetupIntegrationServer creates a complete integration test server with database,
// services, and handlers configured.
func SetupIntegrationServer(t *testing.T) IntegrationServer {
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

	return IntegrationServer{
		HTTPServer:    httpserver.NewServer(deviceHandler, authHandler, logger),
		DeviceService: deviceService,
		AuthService:   authService,
	}
}
