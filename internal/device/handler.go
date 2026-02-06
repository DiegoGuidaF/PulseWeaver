package device

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"

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

func (h *OpenApiHandler) GetDeviceAddresses(ctx context.Context, request api.GetDeviceAddressesRequestObject) (api.GetDeviceAddressesResponseObject, error) {

	deviceId := DeviceId(request.DeviceId)

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
		addressesResponse[i] = addresses[i].toResponse()
	}

	return api.GetDeviceAddresses200JSONResponse(addressesResponse), nil
}

func (h *OpenApiHandler) AddAddress(ctx context.Context, request api.AddAddressRequestObject) (api.AddAddressResponseObject, error) {
	deviceId := DeviceId(request.DeviceId)
	ipAddress := request.Body.Ip

	// Validate IPv4 format
	if err := validateIPv4(ipAddress); err != nil {
		return api.AddAddress400JSONResponse(errorMsgResponse(fmt.Sprintf("Received address %s is not a valid ipv4", ipAddress))), nil
	}

	deviceIp, err := h.service.AssignAddress(ctx, deviceId, ipAddress)
	if err != nil {
		switch {
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

	h.logger.Info("deviceIp assigned",
		slog.Int64("device_id", deviceId.Int64()),
		slog.String("deviceIp", deviceIp.IP),
	)
	return api.AddAddress201JSONResponse(deviceIp.toResponse()), nil
}
func (h *OpenApiHandler) DisableAddress(ctx context.Context, request api.DisableAddressRequestObject) (api.DisableAddressResponseObject, error) {
	deviceId := DeviceId(request.DeviceId)
	addressId := AddressId(request.AddressId)

	deviceIp, err := h.service.DisableAddress(ctx, deviceId, addressId)
	if err != nil {
		if errors.Is(err, ErrAddressNotFound) {
			return api.DisableAddress404JSONResponse(errorMsgResponse(fmt.Sprintf("Address id %s for device id %s not found or already disabled", addressId, deviceId))), nil
		}
		h.logger.Error("failed to disable address",
			slog.Int64("device_id", deviceId.Int64()),
			slog.Int64("address_id", addressId.Int64()),
			slog.Any("error", err),
		)
		return api.DisableAddress500JSONResponse(errorMsgResponse("Failed to disable address")), nil
	}

	h.logger.Info("address disabled",
		slog.Int64("device_id", deviceId.Int64()),
		slog.Int64("address_id", addressId.Int64()),
	)
	return api.DisableAddress200JSONResponse(deviceIp.toResponse()), nil
}

func errorMsgResponse(errorMsg string) api.ErrorResponse {
	return api.ErrorResponse{Error: &errorMsg}
}

func validateIPv4(ipAddress string) error {
	ip := net.ParseIP(ipAddress)
	if ip == nil {
		return ErrInvalidIPFormat
	}

	// Check it's IPv4 (net.ParseIP accepts both IPv4 and IPv6)
	if ip.To4() == nil {
		return ErrIPv6NotSupported
	}

	return nil
}
