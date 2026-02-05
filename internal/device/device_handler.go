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

func (h *OpenApiHandler) GetDeviceIps(ctx context.Context, request api.GetDeviceIpsRequestObject) (api.GetDeviceIpsResponseObject, error) {

	deviceId := DeviceID(request.Id)

	deviceIps, err := h.service.ListDeviceIPs(ctx, deviceId)
	if err != nil {
		if errors.Is(err, ErrDeviceNotFound) {
			return api.GetDeviceIps404JSONResponse(
				errorMsgResponse(fmt.Sprintf("Device with id %s not found", deviceId)),
			), nil
		}

		h.logger.Error("failed to list device deviceIps",
			slog.Int64("device_id", deviceId.Int64()),
			slog.Any("error", err),
		)
		return api.GetDeviceIps500JSONResponse(errorMsgResponse("Failed to list device IPs")), nil
	}

	deviceIpsResponse := make([]api.DeviceIP, len(deviceIps))
	for i := range deviceIps {
		deviceIpsResponse[i] = deviceIps[i].toResponse()
	}

	return api.GetDeviceIps200JSONResponse(deviceIpsResponse), nil
}

func (h *OpenApiHandler) AddDeviceIp(ctx context.Context, request api.AddDeviceIpRequestObject) (api.AddDeviceIpResponseObject, error) {
	deviceId := DeviceID(request.Id)
	ipAddress := request.Body.IpAddress

	// Validate IPv4 format
	if err := validateIPv4(ipAddress); err != nil {
		return api.AddDeviceIp400JSONResponse(errorMsgResponse(fmt.Sprintf("Received IP %s is not a valid ipv4", ipAddress))), nil
	}

	deviceIp, err := h.service.AssignIP(ctx, deviceId, ipAddress)
	if err != nil {
		switch {
		case errors.Is(err, ErrDeviceNotFound):
			return api.AddDeviceIp404JSONResponse(errorMsgResponse(fmt.Sprintf("Device with id %s not found", deviceId))), nil
		default:
			h.logger.Error("failed to assign deviceIp",
				slog.Int64("device_id", deviceId.Int64()),
				slog.String("deviceIp", ipAddress),
				slog.Any("error", err),
			)
			return api.AddDeviceIp500JSONResponse(errorMsgResponse("Failed to assign IP")), nil
		}
	}

	h.logger.Info("deviceIp assigned",
		slog.Int64("device_id", deviceId.Int64()),
		slog.String("deviceIp", deviceIp.IPAddress),
	)
	return api.AddDeviceIp201JSONResponse(deviceIp.toResponse()), nil
}
func (h *OpenApiHandler) DisableDeviceIp(ctx context.Context, request api.DisableDeviceIpRequestObject) (api.DisableDeviceIpResponseObject, error) {
	deviceId := DeviceID(request.Id)
	deviceIpId := DeviceIpID(request.IpId)

	deviceIp, err := h.service.DisableDeviceIP(ctx, deviceId, deviceIpId)
	if err != nil {
		switch {
		case errors.Is(err, ErrDeviceIPNotFound):
			return api.DisableDeviceIp404JSONResponse(errorMsgResponse(fmt.Sprintf("Device with id %s not found", deviceId))), nil
		case errors.Is(err, ErrDeviceIPWrongDevice):
			return api.DisableDeviceIp404JSONResponse(errorMsgResponse(fmt.Sprintf("Device with id %s not found", deviceId))), nil
		case errors.Is(err, ErrDeviceIPDisabled):
			return api.DisableDeviceIp409JSONResponse(errorMsgResponse(fmt.Sprintf("Device with id %s is already disabled", deviceId))), nil
		default:
			h.logger.Error("failed to disable device ip",
				slog.Int64("device_id", deviceId.Int64()),
				slog.Int64("ip_id", deviceIpId.Int64()),
				slog.Any("error", err),
			)
			return api.DisableDeviceIp500JSONResponse(errorMsgResponse("Failed to disable device IP")), nil
		}
	}

	h.logger.Info("device ip disabled",
		slog.Int64("device_id", deviceId.Int64()),
		slog.Int64("ip_id", deviceIpId.Int64()),
	)
	return api.DisableDeviceIp204JSONResponse(deviceIp.toResponse()), nil
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
