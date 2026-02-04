package device

import (
	"context"
	"log/slog"

	"forgejo.wally.mywire.org/diego/WallyDic.git/api"
)

// ptr Helper function to quickly make a pointer of an inline variable
func ptr[T any](v T) *T { return &v }

type OpenApiHandler struct {
	service *Service
	logger  *slog.Logger
}

func ErrorMsgResponse(errorMsg string) api.ErrorResponse {
	return api.ErrorResponse{Error: ptr(errorMsg)}
}

func NewOpenApiHandler(service *Service, logger *slog.Logger) *OpenApiHandler {
	return &OpenApiHandler{service: service, logger: logger}
}

func (h *OpenApiHandler) ListDevices(ctx context.Context, _ api.ListDevicesRequestObject) (api.ListDevicesResponseObject, error) {
	devices, err := h.service.GetDevices(ctx)
	h.logger.Info("Running Query!")
	if err != nil {
		h.logger.Error("Error fetching devices", slog.Any("error", err))
		return api.ListDevices500JSONResponse(ErrorMsgResponse("Error fetching devices")), nil
	}

	apiDevices := make([]api.Device, len(devices))

	for i := range devices {
		apiDevices[i] = api.Device{
			CreatedAt: devices[i].CreatedAt,
			Id:        devices[i].ID.Int64(),
			Name:      devices[i].Name,
		}
	}

	return api.ListDevices200JSONResponse(apiDevices), nil
}
