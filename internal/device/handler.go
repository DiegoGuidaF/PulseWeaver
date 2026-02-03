package device

import (
	"errors"
	"log/slog"
	"net/http"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/api"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *Service
	logger  *slog.Logger
}

func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func (h *Handler) respondJSON(w http.ResponseWriter, status int, data any) {
	if err := api.EncodeJSON(w, status, data); err != nil {
		h.logger.Error("failed to encode json response",
			slog.Int("status", status),
			slog.Any("error", err),
		)
	}
}

func (h *Handler) respondError(w http.ResponseWriter, status int, message string) {
	if err := api.EncodeError(w, status, message); err != nil {
		h.logger.Error("failed to encode error response",
			slog.Int("status", status),
			slog.String("message", message),
			slog.Any("error", err),
		)
	}
}

// GetDevices godoc
//
//	@Summary		List all devices
//	@Description	Get all devices in the system
//	@Tags			devices
//	@Produce		json
//	@Success		200	{array}		Device
//	@Failure		500	{object}	api.ErrorResponse
//	@Router			/devices [get]
func (h *Handler) GetDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := h.service.GetDevices(r.Context())
	if err != nil {
		h.logger.Error("Error fetching devices", slog.Any("error", err))
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch devices")
		return
	}

	h.respondJSON(w, http.StatusOK, devices)
}

// CreateDevice godoc
//
//	@Summary		Create a device
//	@Description	Create a new device with a name
//	@Tags			devices
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateDeviceRequest	true	"Device creation request"
//	@Success		201		{object}	Device
//	@Failure		400		{object}	api.ErrorResponse
//	@Failure		500		{object}	api.ErrorResponse
//	@Router			/devices [post]
func (h *Handler) CreateDevice(w http.ResponseWriter, r *http.Request) {
	req, err := api.DecodeJSON[CreateDeviceRequest](r)
	if err != nil {
		h.logger.Warn("invalid json body", slog.Any("error", err))
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := req.Validate(); err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	device, err := h.service.CreateDevice(r.Context(), req.Name)
	if err != nil {
		h.logger.Error("failed to create device",
			slog.String("name", req.Name),
			slog.Any("error", err),
		)
		h.respondError(w, http.StatusInternalServerError, "Failed to create device")
		return
	}

	h.logger.Info("device created", slog.String("device_id", device.ID))
	h.respondJSON(w, http.StatusCreated, device)
}

// ListDeviceIPs godoc
//
//	@Summary		List device IPs
//	@Description	Get all enabled IPs for a specific device
//	@Tags			device-ips
//	@Produce		json
//	@Param			id	path		string	true	"Device ID (UUID)"
//	@Success		200	{array}		DeviceIP
//	@Failure		400	{object}	api.ErrorResponse
//	@Failure		404	{object}	api.ErrorResponse
//	@Failure		500	{object}	api.ErrorResponse
//	@Router			/devices/{id}/ips [get]
func (h *Handler) ListDeviceIPs(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "id")
	if deviceID == "" {
		h.respondError(w, http.StatusBadRequest, "device ID is required")
		return
	}

	ips, err := h.service.ListDeviceIPs(r.Context(), deviceID)
	if err != nil {
		if errors.Is(err, ErrDeviceNotFound) {
			h.respondError(w, http.StatusNotFound, err.Error())
			return
		}

		h.logger.Error("failed to list device ips",
			slog.String("device_id", deviceID),
			slog.Any("error", err),
		)
		h.respondError(w, http.StatusInternalServerError, "failed to list device IPs")
		return
	}

	h.respondJSON(w, http.StatusOK, ips)
}

// AssignIP godoc
//
//	@Summary		Assign IP to device
//	@Description	Add a new IPv4 address to a device
//	@Tags			device-ips
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Device ID (UUID)"
//	@Param			request	body		AssignDeviceIPRequest	true	"IP assignment request"
//	@Success		201		{object}	DeviceIP
//	@Failure		400		{object}	api.ErrorResponse	"Invalid IP format or IPv6 not supported"
//	@Failure		404		{object}	api.ErrorResponse	"Device not found"
//	@Failure		500		{object}	api.ErrorResponse
//	@Router			/devices/{id}/ips [post]
func (h *Handler) AssignIP(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "id")
	if deviceID == "" {
		h.respondError(w, http.StatusBadRequest, "device ID is required")
		return
	}

	req, err := api.DecodeJSON[AssignDeviceIPRequest](r)
	if err != nil {
		h.logger.Warn("invalid json body",
			slog.String("device_id", deviceID),
			slog.Any("error", err),
		)
		h.respondError(w, http.StatusBadRequest, "Failed to decode request body")
		return
	}

	if err := req.Validate(); err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	ip, err := h.service.AssignIP(r.Context(), deviceID, req.IPAddress)
	if err != nil {
		switch {
		case errors.Is(err, ErrDeviceNotFound):
			h.respondError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrInvalidIPFormat), errors.Is(err, ErrIPv6NotSupported):
			h.respondError(w, http.StatusBadRequest, err.Error())
		default:
			h.logger.Error("failed to assign ip",
				slog.String("device_id", deviceID),
				slog.String("ip", req.IPAddress),
				slog.Any("error", err),
			)
			h.respondError(w, http.StatusInternalServerError, "failed to assign IP")
		}
		return
	}

	h.logger.Info("ip assigned",
		slog.String("device_id", deviceID),
		slog.String("ip", ip.IPAddress),
	)
	h.respondJSON(w, http.StatusCreated, ip)
}

// DisableDeviceIP godoc
//
//	@Summary		Disable device IP
//	@Description	Mark an IP address as disabled for a device
//	@Tags			device-ips
//	@Param			id		path	string	true	"Device ID (UUID)"
//	@Param			ip_id	path	int		true	"Device IP ID"
//	@Success		204		"IP successfully disabled"
//	@Failure		400		{object}	api.ErrorResponse	"Missing device or IP ID"
//	@Failure		404		{object}	api.ErrorResponse	"Device IP not found or wrong device"
//	@Failure		409		{object}	api.ErrorResponse	"Device IP already disabled"
//	@Failure		500		{object}	api.ErrorResponse
//	@Router			/devices/{id}/ips/{ip_id} [patch]
func (h *Handler) DisableDeviceIP(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "id")
	if deviceID == "" {
		h.respondError(w, http.StatusBadRequest, "device ID is required")
		return
	}

	ipIDStr := chi.URLParam(r, "ip_id")
	if ipIDStr == "" {
		h.respondError(w, http.StatusBadRequest, "ip ID is required")
		return
	}

	err := h.service.DisableDeviceIP(r.Context(), deviceID, ipIDStr)
	if err != nil {
		switch {
		case errors.Is(err, ErrDeviceIPNotFound):
			h.respondError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrDeviceIPWrongDevice):
			h.respondError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrDeviceIPDisabled):
			h.respondError(w, http.StatusConflict, err.Error())
		default:
			h.logger.Error("failed to disable device ip",
				slog.String("device_id", deviceID),
				slog.String("ip_id", ipIDStr),
				slog.Any("error", err),
			)
			h.respondError(w, http.StatusInternalServerError, "failed to disable device IP")
		}
		return
	}

	h.logger.Info("device ip disabled",
		slog.String("device_id", deviceID),
		slog.String("ip_id", ipIDStr),
	)
	w.WriteHeader(http.StatusNoContent)
}
