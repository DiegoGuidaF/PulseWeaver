package device

import (
	"context"
	"errors"
	"fmt"
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
	if err != nil {
		h.logger.Error("Error fetching devices", slog.Any("error", err))
		return api.GetDevices500JSONResponse(errorMsgResponse("Error fetching devices")), nil
	}

	apiDevices := make([]api.Device, len(devices))
	for i := range devices {
		apiDevices[i] = toDeviceResponse(&devices[i])
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

	return api.CreateDevice201JSONResponse(toDeviceResponse(device)), nil
}

func (h *OpenApiHandler) GetDeviceAddresses(ctx context.Context, request api.GetDeviceAddressesRequestObject) (api.GetDeviceAddressesResponseObject, error) {
	deviceId := DeviceID(request.DeviceId)

	addresses, err := h.service.GetAddressesForDevice(ctx, deviceId)
	if err != nil {
		if errors.Is(err, ErrDeviceNotFound) {
			return api.GetDeviceAddresses404JSONResponse(
				errorMsgResponse(fmt.Sprintf("Device with id %s not found", deviceId)),
			), nil
		}

		h.logger.Error("failed to list device addresses",
			slog.Int64("device_id", deviceId.Int64()),
			slog.Any("error", err),
		)
		return api.GetDeviceAddresses500JSONResponse(errorMsgResponse("Failed to list device IPs")), nil
	}

	addressesResponse := make([]api.Address, len(addresses))
	for i := range addresses {
		addressesResponse[i] = toAddressResponse(&addresses[i])
	}

	return api.GetDeviceAddresses200JSONResponse(addressesResponse), nil
}

func (h *OpenApiHandler) AddAddress(ctx context.Context, request api.AddAddressRequestObject) (api.AddAddressResponseObject, error) {
	deviceId := DeviceID(request.DeviceId)
	ipAddress := request.Body.Ip

	addresswIp, wasCreated, err := h.service.AssignAddress(ctx, deviceId, ipAddress)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidIPFormat):
			return api.AddAddress400JSONResponse(errorMsgResponse(fmt.Sprintf("Received address %s is not a valid IPv4 or IPv6 address", ipAddress))), nil
		case errors.Is(err, ErrDeviceNotFound):
			return api.AddAddress404JSONResponse(errorMsgResponse(fmt.Sprintf("Device with id %s not found", deviceId))), nil
		default:
			h.logger.Error("failed to assign address",
				slog.Int64("device_id", deviceId.Int64()),
				slog.String("ip", ipAddress),
				slog.Any("error", err),
			)
			return api.AddAddress500JSONResponse(errorMsgResponse("Failed to assign address")), nil
		}
	}
	if wasCreated {
		return api.AddAddress201JSONResponse(toAddressResponse(addresswIp)), nil
	}

	return api.AddAddress200JSONResponse(toAddressResponse(addresswIp)), nil
}

func (h *OpenApiHandler) DisableAddress(ctx context.Context, request api.DisableAddressRequestObject) (api.DisableAddressResponseObject, error) {
	deviceId := DeviceID(request.DeviceId)
	addressId := AddressID(request.AddressId)

	address, err := h.service.DisableAddress(ctx, deviceId, addressId)
	if err != nil {
		if errors.Is(err, ErrAddressNotFound) || errors.Is(err, ErrAddressNotOwnedByDevice) {
			return api.DisableAddress404JSONResponse(errorMsgResponse(fmt.Sprintf("Address id %s for device id %s not found or already disabled", addressId, deviceId))), nil
		}
		h.logger.Error("failed to disable address",
			slog.Int64("device_id", deviceId.Int64()),
			slog.Int64("address_id", addressId.Int64()),
			slog.Any("error", err),
		)
		return api.DisableAddress500JSONResponse(errorMsgResponse("Failed to disable address")), nil
	}

	return api.DisableAddress200JSONResponse(toAddressResponse(address)), nil
}

func (h *OpenApiHandler) DeviceHeartbeat(ctx context.Context, request api.DeviceHeartbeatRequestObject) (api.DeviceHeartbeatResponseObject, error) {
	deviceId := DeviceID(request.DeviceId)

	// Extract client IP from context (set by middleware)
	clientIP, ok := ClientIPFromContext(ctx)
	if !ok {
		h.logger.Error("failed to extract client IP from request")
		return api.DeviceHeartbeat400JSONResponse(errorMsgResponse("Failed to extract client IP address")), nil
	}
	h.logger.Debug(
		"Received heartbeat request",
		slog.Int64("device", deviceId.Int64()),
		slog.String("client_ip", clientIP))

	// Call service to checkin the device
	address, isNew, err := h.service.AssignAddress(ctx, deviceId, clientIP)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidIPFormat):
			return api.DeviceHeartbeat400JSONResponse(errorMsgResponse(fmt.Sprintf("Received address %s is not a valid IPv4 or IPv6 address", clientIP))), nil
		case errors.Is(err, ErrDeviceNotFound):
			return api.DeviceHeartbeat404JSONResponse(errorMsgResponse(fmt.Sprintf("Device with id %s not found", deviceId))), nil
		default:
			h.logger.Error("failed to checkin device",
				slog.Int64("device_id", deviceId.Int64()),
				slog.String("ip", clientIP),
				slog.Any("error", err),
			)
			return api.DeviceHeartbeat500JSONResponse(errorMsgResponse("Failed to checkin device")), nil
		}
	}

	if isNew {
		return api.DeviceHeartbeat201JSONResponse(toAddressResponse(address)), nil
	}

	return api.DeviceHeartbeat200JSONResponse(toAddressResponse(address)), nil
}

func toDeviceResponse(d *Device) api.Device {
	return api.Device{
		ID:        d.ID.Int64(),
		Name:      d.Name,
		CreatedAt: d.CreatedAt,
	}
}

func toAddressResponse(a *AddressWithStatus) api.Address {
	return api.Address{
		ID:        a.AddressId.Int64(),
		DeviceId:  a.DeviceId.Int64(),
		IP:        a.IP,
		Status:    a.Status,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt.Time,
	}
}

func errorMsgResponse(errorMsg string) api.ErrorResponse {
	return api.ErrorResponse{Error: &errorMsg}
}
