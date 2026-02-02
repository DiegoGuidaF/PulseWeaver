package device

import (
	"encoding/json"
	"log"
	"net/http"
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
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to fetch devices",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(devices)
}

func (h *Handler) CreateDevice(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	device, err := h.service.CreateDevice(ctx, "test_name")
	if err != nil {
		log.Printf("Error creating device: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to create device",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(device)
}
