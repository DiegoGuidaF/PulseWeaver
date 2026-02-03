package device

import (
	"log"
	"net/http"
	"strings"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/api"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetDevices(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	devices, err := h.service.GetDevices(ctx)
	if err != nil {
		log.Printf("Error fetching devices: %v", err)
		api.EncodeError(w, http.StatusInternalServerError, "Failed to fetch devices")
		return
	}

	api.EncodeJSON(w, http.StatusOK, devices)
}

func (h *Handler) CreateDevice(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Decode the JSON request body
	req, err := api.DecodeJSON[CreateDeviceRequest](r)
	if err != nil {
		log.Printf("Decode error: %v", err)
		api.EncodeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate
	if err := req.Validate(); err != nil {
		api.EncodeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Create
	device, err := h.service.CreateDevice(ctx, req.Name)
	if err != nil {
		log.Printf("Create error: %v", err)
		api.EncodeError(w, http.StatusInternalServerError, "Failed to create device")
		return
	}

	api.EncodeJSON(w, http.StatusCreated, device)
}

func (h *Handler) AssignIP(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "id")
	if deviceID == "" {
		log.Printf("Device ID is required")
		api.EncodeError(w, http.StatusBadRequest, "device ID is required")
		return
	}

	var req AssignIPRequest
	req, err := api.DecodeJSON[AssignIPRequest](r)
	if err != nil {
		log.Printf("Cannot parse body: %v", err)
		api.EncodeError(w, http.StatusBadRequest, "Failed to decode request body")
		return
	}

	if err := req.Validate(); err != nil {
		log.Printf("Invalid IP: %v", err)
		api.EncodeError(w, http.StatusBadRequest, "Invalid IP")
		return
	}

	ip, err := h.service.AssignIP(r.Context(), deviceID, req.IPAddress)
	if err != nil {
		log.Printf("Cannot create IP: %v", err)
		if strings.Contains(err.Error(), "not found") {
			api.EncodeError(w, http.StatusNotFound, err.Error())
			return
		}
		if strings.Contains(err.Error(), "invalid IP") || strings.Contains(err.Error(), "only IPv4") {
			api.EncodeError(w, http.StatusBadRequest, err.Error())
			return
		}
		api.EncodeError(w, http.StatusInternalServerError, "failed to assign IP")
		return
	}

	api.EncodeJSON(w, http.StatusCreated, ip)
}

func (h *Handler) ListDeviceIPs(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "id")
	if deviceID == "" {
		api.EncodeError(w, http.StatusBadRequest, "device ID is required")
		return
	}

	ips, err := h.service.ListDeviceIPs(r.Context(), deviceID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			api.EncodeError(w, http.StatusNotFound, err.Error())
			return
		}
		api.EncodeError(w, http.StatusInternalServerError, "failed to list device IPs")
		return
	}

	api.EncodeJSON(w, http.StatusOK, ips)
}

func (h *Handler) DisableDeviceIP(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "id")
	if deviceID == "" {
		api.EncodeError(w, http.StatusBadRequest, "device ID is required")
		return
	}

	ipIDStr := chi.URLParam(r, "ip_id")
	if ipIDStr == "" {
		api.EncodeError(w, http.StatusBadRequest, "ip ID is required")
		return
	}

	//ipID, err := strconv.ParseInt(ipIDStr, 10, 64)
	//if err != nil {
	//	api.EncodeError(w, http.StatusBadRequest, "invalid ip ID format")
	//	return
	//}

	err := h.service.DisableDeviceIP(r.Context(), deviceID, ipIDStr)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "does not belong") {
			api.EncodeError(w, http.StatusNotFound, err.Error())
			return
		}
		if strings.Contains(err.Error(), "already disabled") {
			api.EncodeError(w, http.StatusConflict, err.Error())
			return
		}
		api.EncodeError(w, http.StatusInternalServerError, "failed to disable device IP")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
