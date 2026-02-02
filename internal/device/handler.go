package device

import (
	"log"
	"net/http"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/api"
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
