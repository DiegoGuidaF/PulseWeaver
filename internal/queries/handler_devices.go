package queries

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

func (h *HTTPHandler) GetDeviceAddresses(
	ctx context.Context,
	request httpapi.GetDeviceAddressesRequestObject,
) (httpapi.GetDeviceAddressesResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetDeviceAddresses")
	deviceID := ids.DeviceID(request.DeviceId)
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

	devices, err := h.repo.GetDevices(ctx, nil)
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

func (h *HTTPHandler) GetDevicesByUser(
	ctx context.Context,
	request httpapi.GetDevicesByUserRequestObject,
) (httpapi.GetDevicesByUserResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetDevicesByUser")

	devices, err := h.repo.GetDevicesByUser(ctx, ids.UserID(request.UserId))
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrUserNotFound):
			return httpapi.GetDevicesByUser404JSONResponse(errorMsgResponse("User not found")), nil
		default:
			h.logger.ErrorContext(ctx, "failed to list devices by user", slog.Any(logging.AttrKeyError, err))
			return httpapi.GetDevicesByUser500JSONResponse(errorMsgResponse("Failed to list devices")), nil
		}
	}

	response := make([]httpapi.Device, len(devices))
	for i := range devices {
		response[i] = toDeviceViewResponse(&devices[i])
	}
	return httpapi.GetDevicesByUser200JSONResponse(response), nil
}

func (h *HTTPHandler) GetDevice(ctx context.Context, request httpapi.GetDeviceRequestObject) (httpapi.GetDeviceResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetDevice")
	deviceID := ids.DeviceID(request.DeviceId)
	logger := h.logger.With(slog.Int64(device.AttrKeyDeviceID, deviceID.Int64()))

	detail, err := h.repo.GetDeviceDetail(ctx, deviceID)
	if err != nil {
		switch {
		case errors.Is(err, device.ErrDeviceNotFound):
			logger.WarnContext(ctx, "device not found")
			return httpapi.GetDevice404JSONResponse(errorMsgResponse(fmt.Sprintf("Device with id %d not found", deviceID))), nil
		default:
			logger.ErrorContext(ctx, "failed to get device", slog.Any(logging.AttrKeyError, err))
			return httpapi.GetDevice500JSONResponse(errorMsgResponse("Failed to get device")), nil
		}
	}

	return httpapi.GetDevice200JSONResponse(toDeviceDetailResponse(detail)), nil
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
	var lastSeenAt *httpapi.UTCTime
	if d.LastSeenAt != nil {
		lastSeenAt = new(httpapi.UTCTime(d.LastSeenAt.Time))
	}
	return httpapi.Device{
		Id:           d.ID.Int64(),
		Name:         d.Name,
		DeviceType:   httpapi.DeviceDeviceType(d.DeviceType),
		Description:  d.Description,
		Icon:         d.Icon,
		CreatedAt:    httpapi.UTCTime(d.CreatedAt),
		UpdatedAt:    httpapi.UTCTime(d.UpdatedAt),
		ApiKeyPrefix: d.KeyPrefix,
		AddressCount: new(d.AddressCount),
		LastSeenAt:   lastSeenAt,
		OwnerId:      d.OwnerID.Int64(),
		OwnerName:    new(d.OwnerName),
	}
}

func toDeviceDetailResponse(d *DeviceDetail) httpapi.Device {
	var lastSeenAt *httpapi.UTCTime
	if d.LastSeenAt != nil {
		lastSeenAt = new(httpapi.UTCTime(d.LastSeenAt.Time))
	}
	return httpapi.Device{
		Id:           d.ID.Int64(),
		Name:         d.Name,
		DeviceType:   httpapi.DeviceDeviceType(d.DeviceType),
		Description:  d.Description,
		Icon:         d.Icon,
		CreatedAt:    httpapi.UTCTime(d.CreatedAt),
		UpdatedAt:    httpapi.UTCTime(d.UpdatedAt),
		ApiKeyPrefix: d.KeyPrefix,
		AddressCount: &d.AddressCount,
		LastSeenAt:   lastSeenAt,
		OwnerId:      d.OwnerID.Int64(),
		OwnerName:    new(d.OwnerName),
	}
}
