package queries

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/DiegoGuidaF/WallyDex/internal/device"
	"github.com/DiegoGuidaF/WallyDex/internal/httpapi"
	"github.com/DiegoGuidaF/WallyDex/internal/logging"
)

type HTTPHandler struct {
	repo   *Repository
	logger *slog.Logger
}

func NewHTTPHandler(repo *Repository, logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		repo:   repo,
		logger: logger.With(slog.String(logging.AttrKeyComponent, "queries")),
	}
}

func (h *HTTPHandler) GetDeviceAddresses(
	ctx context.Context,
	request httpapi.GetDeviceAddressesRequestObject,
) (httpapi.GetDeviceAddressesResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetDeviceAddresses")
	deviceID := device.DeviceID(request.DeviceId)
	logger := h.logger.With(slog.Int64("device_id", deviceID.Int64()))

	exists, err := h.repo.DeviceExists(ctx, deviceID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to check device existence", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetDeviceAddresses500JSONResponse(errorMsgResponse("Failed to list device IPs")), nil
	}
	if !exists {
		logger.WarnContext(ctx, "device not found")
		return httpapi.GetDeviceAddresses404JSONResponse(
			errorMsgResponse(fmt.Sprintf("Device with id %d not found", deviceID)),
		), nil
	}

	addresses, err := h.repo.GetDeviceAddresses(ctx, deviceID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to list device addresses", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetDeviceAddresses500JSONResponse(errorMsgResponse("Failed to list device IPs")), nil
	}

	response := make([]httpapi.Address, len(addresses))
	for i := range addresses {
		response[i] = toAddressResponse(&addresses[i])
	}
	return httpapi.GetDeviceAddresses200JSONResponse(response), nil
}

func toAddressResponse(a *AddressView) httpapi.Address {
	address := httpapi.Address{
		Id:        a.ID.Int64(),
		DeviceId:  a.DeviceID.Int64(),
		Ip:        a.IP,
		IsEnabled: a.IsEnabled,
		CreatedAt: httpapi.UTCTime(a.CreatedAt),
		UpdatedAt: httpapi.UTCTime(a.UpdatedAt),
	}
	if a.ExpiresAt != nil {
		address.ExpiresAt = a.ExpiresAt
	}

	return address
}

func errorMsgResponse(msg string) httpapi.ErrorResponse {
	return httpapi.ErrorResponse{Error: &msg}
}
