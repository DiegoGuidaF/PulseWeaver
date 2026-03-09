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
	logger := h.logger.With(slog.Int64(device.AttrKeyDeviceID, deviceID.Int64()))

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
		response[i] = toAddressViewResponse(&addresses[i])
	}
	return httpapi.GetDeviceAddresses200JSONResponse(response), nil
}

func (h *HTTPHandler) GetDevices(
	ctx context.Context,
	_ httpapi.GetDevicesRequestObject,
) (httpapi.GetDevicesResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetDevices")

	devices, err := h.repo.GetDevices(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to list devices", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetDevices500JSONResponse(errorMsgResponse("Failed to list devices")), nil
	}

	response := make([]httpapi.Device, len(devices))
	for i := range devices {
		response[i] = toDeviceViewResponse(&devices[i])
	}
	return httpapi.GetDevices200JSONResponse(response), nil
}

func toAddressViewResponse(a *AddressView) httpapi.Address {
	address := httpapi.Address{
		Id:        a.ID.Int64(),
		DeviceId:  a.DeviceID.Int64(),
		Ip:        a.IP,
		IsEnabled: a.IsEnabled,
		CreatedAt: httpapi.UTCTime(a.CreatedAt),
		UpdatedAt: httpapi.UTCTime(a.UpdatedAt),
	}
	if a.ExpiresAt != nil {
		address.ExpiresAt = new(httpapi.UTCTime(*a.ExpiresAt))
	}

	return address
}

func toDeviceViewResponse(d *DeviceView) httpapi.Device {
	return httpapi.Device{
		Id:           d.ID.Int64(),
		Name:         d.Name,
		CreatedAt:    httpapi.UTCTime(d.CreatedAt),
		ApiKeyPrefix: d.KeyPrefix,
		AddressCount: new(d.AddressCount),
	}
}

func errorMsgResponse(msg string) httpapi.ErrorResponse {
	return httpapi.ErrorResponse{Error: &msg}
}
