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
		// This only logs if json.Encode itself fails (rare: broken pipe, cyclic reference)
		// The middleware already logged the request, so we just log the encoding issue
		h.logger.Error("failed to encode json response",
			slog.Int("status", status),
			slog.Any("error", err),
		)
	}
}

// respondError safely encodes error response and logs any encoding errors
func (h *Handler) respondError(w http.ResponseWriter, status int, message string) {
	if err := api.EncodeError(w, status, message); err != nil {
		h.logger.Error("failed to encode error response",
			slog.Int("status", status),
			slog.String("message", message),
			slog.Any("error", err),
		)
	}
}

func (h *Handler) GetDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := h.service.GetDevices(r.Context())
	if err != nil {
		h.logger.Error("Error fetching devices", slog.Any("error", err))
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch devices")
		return
	}

	h.respondJSON(w, http.StatusOK, devices)
}

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
	}

	h.logger.Info("ip assigned",
		slog.String("device_id", deviceID),
		slog.String("ip", ip.IPAddress),
	)
	h.respondJSON(w, http.StatusCreated, ip)
}

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
	}

	h.logger.Info("device ip disabled",
		slog.String("device_id", deviceID),
		slog.String("ip_id", ipIDStr),
	)
	w.WriteHeader(http.StatusNoContent)
}
