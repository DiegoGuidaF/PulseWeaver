package httpserver

import (
	"net/http"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/health"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
)

func NewServer(
	deviceHandler *device.Handler,
) http.Handler {
	r := chi.NewRouter()

	// global middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	addRoutes(r, deviceHandler)

	return r
}

func addRoutes(
	r *chi.Mux,
	deviceHandler *device.Handler,
) {
	r.Get("/health", health.Handler)

	// Devices
	r.Get("/api/v1/devices", deviceHandler.GetDevices)
	r.Post("/api/v1/devices", deviceHandler.CreateDevice)

	// IP routes
	r.Post("/api/v1/devices/{id}/ips", deviceHandler.AssignIP)
	r.Get("/api/v1/devices/{id}/ips", deviceHandler.ListDeviceIPs)
	r.Delete("/api/v1/devices/{id}/ips/{ip_id}", deviceHandler.DisableDeviceIP)
}
