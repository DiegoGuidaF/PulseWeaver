package device

import (
	"context"
	"log/slog"

	"forgejo.wally.mywire.org/diego/WallyDic.git/api"
)

type OpenApiHandler struct {
	service *Service
	logger  *slog.Logger
}

func NewOpenApiHandler(service *Service, logger *slog.Logger) *OpenApiHandler {
	return &OpenApiHandler{service: service, logger: logger}
}

func (h *OpenApiHandler) GetDevices(ctx context.Context, _ api.GetDevicesRequestObject) (api.GetDevicesResponseObject, error) {
	devices, err := h.service.GetDevices(ctx)
	h.logger.Info("Running Query!")
	if err != nil {
		h.logger.Error("Error fetching devices", slog.Any("error", err))
		return api.GetDevices500JSONResponse(errorMsgResponse("Error fetching devices")), nil
	}

	apiDevices := make([]api.Device, len(devices))

	for i := range devices {
		apiDevices[i] = devices[i].toResponse()
	}

	return api.GetDevices200JSONResponse(apiDevices), nil
}

func (h *OpenApiHandler) CreateDevice(ctx context.Context, request api.CreateDeviceRequestObject) (api.CreateDeviceResponseObject, error) {
	deviceName := request.Body.Name

	device, err := h.service.CreateDevice(ctx, deviceName)
	if err != nil {
		h.logger.Error("failed to create device",
			slog.String("name", deviceName),
			slog.Any("error", err),
		)
		return api.CreateDevice500JSONResponse(errorMsgResponse("Failed to create device")), nil
	}

	h.logger.Info("device created", slog.Int64("device_id", device.ID.Int64()))
	return api.CreateDevice201JSONResponse(device.toResponse()), nil
}

func errorMsgResponse(errorMsg string) api.ErrorResponse {
	return api.ErrorResponse{Error: &errorMsg}
}
