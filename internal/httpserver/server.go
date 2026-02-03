package httpserver

import (
	"fmt"
	"log/slog"
	"net/http"

	_ "forgejo.wally.mywire.org/diego/WallyDic.git/docs"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/config"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/health"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	slogchi "github.com/samber/slog-chi"
	httpSwagger "github.com/swaggo/http-swagger"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
)

//	@title			WallyDic Device Management API
//	@version		1.0
//	@description	Device and IP address management system
//	@termsOfService	http://swagger.io/terms/

//	@contact.name	API Support
//	@contact.email	support@wallydic.com

//	@license.name	MIT
//	@license.url	https://opensource.org/licenses/MIT

//	@BasePath	/api/v1

//	@schemes	http https

func NewServer(
	deviceHandler *device.Handler,
	logger *slog.Logger,
	config config.ConfServer,
) http.Handler {
	r := chi.NewRouter()

	loggerConfig := slogchi.Config{
		WithSpanID:  true,
		WithTraceID: true,
	}

	r.Use(slogchi.NewWithConfig(logger, loggerConfig))
	r.Use(middleware.Recoverer)

	addRoutes(r, deviceHandler, config)

	return r
}

func addRoutes(r *chi.Mux, deviceHandler *device.Handler, config config.ConfServer) {
	r.Get("/health", health.Handler)

	// Dynamic Swagger URL based on actual port
	swaggerURL := fmt.Sprintf("http://localhost:%d/swagger/doc.json", config.Port)

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL(swaggerURL),
	))

	// Devices
	r.Get("/api/v1/devices", deviceHandler.GetDevices)
	r.Post("/api/v1/devices", deviceHandler.CreateDevice)

	// IP routes
	r.Get("/api/v1/devices/{id}/ips", deviceHandler.ListDeviceIPs)
	r.Post("/api/v1/devices/{id}/ips", deviceHandler.AssignIP)
	r.Patch("/api/v1/devices/{id}/ips/{ip_id}/disable", deviceHandler.DisableDeviceIP)
}
