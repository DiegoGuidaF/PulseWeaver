package device

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"forgejo.wally.mywire.org/diego/WallyDic.git/api"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/logging"
)

type HTTPHandler struct {
	service *Service
	logger  *slog.Logger
}

func NewOpenApiHandler(service *Service, logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{service: service, logger: logger}
}

func (h *HTTPHandler) GetDevices(ctx context.Context, _ api.GetDevicesRequestObject) (api.GetDevicesResponseObject, error) {
	ctx, logger := logging.Enrich(ctx, slog.String(AttrKeyOperation, "GetDevices"))

	devices, err := h.service.GetDevices(ctx)
	if err != nil {
		logger.Error("failed to list devices", slog.Any(AttrKeyError, err))
		return api.GetDevices500JSONResponse(errorMsgResponse("Error fetching devices")), nil
	}

	logger.Info("devices listed", slog.Int(AttrKeyCount, len(devices)))

	apiDevices := make([]api.Device, len(devices))
	for i := range devices {
		apiDevices[i] = toDeviceResponse(&devices[i])
	}

	return api.GetDevices200JSONResponse(apiDevices), nil
}

func (h *HTTPHandler) CreateDevice(ctx context.Context, request api.CreateDeviceRequestObject) (api.CreateDeviceResponseObject, error) {
	deviceName := request.Body.Name
	ctx, logger := logging.Enrich(ctx,
		slog.String(AttrKeyOperation, "CreateDevice"),
		slog.String(AttrKeyDeviceName, deviceName),
	)

	device, rawApiKey, err := h.service.CreateDevice(ctx, deviceName)
	if err != nil {
		logger.Error("failed to create device", slog.Any(AttrKeyError, err))
		return api.CreateDevice500JSONResponse(errorMsgResponse("Failed to create device")), nil
	}

	logger.Info("device created")

	apiDevice := toDeviceResponse(device)
	return api.CreateDevice201JSONResponse(api.CreateDeviceResponse{
		Device: apiDevice,
		ApiKey: rawApiKey,
	}), nil
}

func (h *HTTPHandler) GetDeviceAddresses(ctx context.Context, request api.GetDeviceAddressesRequestObject) (api.GetDeviceAddressesResponseObject, error) {
	deviceId := DeviceID(request.DeviceId)
	ctx, logger := logging.Enrich(ctx,
		slog.String(AttrKeyOperation, "GetDeviceAddresses"),
		slog.Int64(AttrKeyDeviceID, deviceId.Int64()),
	)

	addresses, err := h.service.GetAddressesForDevice(ctx, deviceId)
	if err != nil {
		if errors.Is(err, ErrDeviceNotFound) {
			logger.Warn("device not found")
			return api.GetDeviceAddresses404JSONResponse(
				errorMsgResponse(fmt.Sprintf("Device with id %s not found", deviceId)),
			), nil
		}

		logger.Error("failed to list device addresses", slog.Any(AttrKeyError, err))
		return api.GetDeviceAddresses500JSONResponse(errorMsgResponse("Failed to list device IPs")), nil
	}

	logger.Info("device addresses listed", slog.Int(AttrKeyCount, len(addresses)))

	addressesResponse := make([]api.Address, len(addresses))
	for i := range addresses {
		addressesResponse[i] = toAddressResponse(&addresses[i])
	}

	return api.GetDeviceAddresses200JSONResponse(addressesResponse), nil
}

func (h *HTTPHandler) AddAddress(ctx context.Context, request api.AddAddressRequestObject) (api.AddAddressResponseObject, error) {
	deviceId := DeviceID(request.DeviceId)
	ipAddress := request.Body.Ip
	ctx, logger := logging.Enrich(ctx,
		slog.String(AttrKeyOperation, "AddAddress"),
		slog.Int64(AttrKeyDeviceID, deviceId.Int64()),
		slog.String(AttrKeyAddressIP, ipAddress),
	)

	addresswIp, wasCreated, err := h.service.AssignAddress(ctx, deviceId, ipAddress)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidIPFormat):
			logger.Warn("invalid request body")
			return api.AddAddress400JSONResponse(errorMsgResponse(fmt.Sprintf("Received address %s is not a valid IPv4 or IPv6 address", ipAddress))), nil
		case errors.Is(err, ErrDeviceNotFound):
			logger.Warn("device not found")
			return api.AddAddress404JSONResponse(errorMsgResponse(fmt.Sprintf("Device with id %s not found", deviceId))), nil
		default:
			logger.Error("failed to assign address", slog.Any(AttrKeyError, err))
			return api.AddAddress500JSONResponse(errorMsgResponse("Failed to assign address")), nil
		}
	}

	logger.Info("Address added or enabled")

	if wasCreated {
		return api.AddAddress201JSONResponse(toAddressResponse(addresswIp)), nil
	}

	return api.AddAddress200JSONResponse(toAddressResponse(addresswIp)), nil
}

func (h *HTTPHandler) DisableAddress(ctx context.Context, request api.DisableAddressRequestObject) (api.DisableAddressResponseObject, error) {
	deviceId := DeviceID(request.DeviceId)
	addressId := AddressID(request.AddressId)
	ctx, logger := logging.Enrich(ctx,
		slog.String(AttrKeyOperation, "DisableAddress"),
		slog.Int64(AttrKeyDeviceID, deviceId.Int64()),
		slog.Int64(AttrKeyAddressID, addressId.Int64()),
	)

	address, err := h.service.DisableAddress(ctx, deviceId, addressId)
	if err != nil {
		if errors.Is(err, ErrAddressNotFound) || errors.Is(err, ErrAddressNotOwnedByDevice) {
			logger.Warn("address not found or not owned by device")
			return api.DisableAddress404JSONResponse(errorMsgResponse(fmt.Sprintf("Address id %s for device id %s not found or already disabled", addressId, deviceId))), nil
		}
		logger.Error("failed to disable address", slog.Any(AttrKeyError, err))
		return api.DisableAddress500JSONResponse(errorMsgResponse("Failed to disable address")), nil
	}

	logger.Info("address disabled")

	return api.DisableAddress200JSONResponse(toAddressResponse(address)), nil
}

func (h *HTTPHandler) DeviceHeartbeat(ctx context.Context, request api.DeviceHeartbeatRequestObject) (api.DeviceHeartbeatResponseObject, error) {
	deviceId := DeviceID(request.DeviceId)

	ctx, logger := logging.Enrich(ctx,
		slog.String(AttrKeyOperation, "DeviceHeartbeat"),
		slog.Int64(AttrKeyDeviceID, deviceId.Int64()),
	)

	clientIp, ok := api.ClientIPFromContext(ctx)
	if !ok {
		logger.Error("client IP not in context")
		return api.DeviceHeartbeat500JSONResponse(errorMsgResponse("Failed to extract client IP address")), nil
	}
	ctx, logger = logging.Enrich(ctx, slog.String(AttrKeyClientIP, clientIp))

	address, isNew, err := h.service.AssignAddress(ctx, deviceId, clientIp)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidIPFormat):
			logger.Warn("invalid request body")
			return api.DeviceHeartbeat400JSONResponse(errorMsgResponse(fmt.Sprintf("Received address %s is not a valid IPv4 or IPv6 address", clientIp))), nil
		case errors.Is(err, ErrDeviceNotFound):
			logger.Warn("device not found")
			return api.DeviceHeartbeat404JSONResponse(errorMsgResponse(fmt.Sprintf("Device with id %s not found", deviceId))), nil
		default:
			logger.Error("heartbeat request failed", slog.Any(AttrKeyError, err))
			return api.DeviceHeartbeat500JSONResponse(errorMsgResponse("Failed to checkin device")), nil
		}
	}

	logger.Info("device heartbeat successful")
	if isNew {
		return api.DeviceHeartbeat201JSONResponse(toAddressResponse(address)), nil
	}

	return api.DeviceHeartbeat200JSONResponse(toAddressResponse(address)), nil
}

func (h *HTTPHandler) DeviceHeartbeatByApiKey(ctx context.Context, request api.DeviceHeartbeatByApiKeyRequestObject) (api.DeviceHeartbeatByApiKeyResponseObject, error) {
	ctx, logger := logging.Enrich(ctx, slog.String(AttrKeyOperation, "DeviceHeartbeatByApiKey"))

	// Extract deviceId from context
	principal, ok := PrincipalFromContext(ctx)
	if !ok {
		logger.Error("invalid API key")
		return api.DeviceHeartbeatByApiKey500JSONResponse(errorMsgResponse("Failed to extract device api key")), nil
	}
	deviceId := principal.DeviceID
	ctx, logger = logging.Enrich(ctx, slog.Int64(AttrKeyDeviceID, deviceId.Int64()))

	// Determine IP to use: prefer body IP, fallback to context IP if not provided
	var ipToUse string
	requestBody := request.Body
	if requestBody != nil && requestBody.Ip != nil && *requestBody.Ip != "" {
		ipToUse = *requestBody.Ip
	} else {
		var ok bool
		ipToUse, ok = api.ClientIPFromContext(ctx)
		if !ok {
			logger.Error("client IP not in context and no IP provided in body")
			return api.DeviceHeartbeatByApiKey500JSONResponse(errorMsgResponse("Failed to extract client IP address")), nil
		}
	}
	ctx, logger = logging.Enrich(ctx, slog.String(AttrKeyAddressIP, ipToUse))

	address, isNew, err := h.service.AssignAddress(ctx, deviceId, ipToUse)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidIPFormat):
			logger.Warn("invalid request body")
			return api.DeviceHeartbeatByApiKey400JSONResponse(errorMsgResponse(fmt.Sprintf("Received address %s is not a valid IPv4 or IPv6 address", ipToUse))), nil
		case errors.Is(err, ErrDeviceNotFound):
			logger.Warn("device not found")
			return api.DeviceHeartbeatByApiKey404JSONResponse(errorMsgResponse(fmt.Sprintf("Device with id %s not found", deviceId))), nil
		default:
			logger.Error("heartbeat request failed", slog.Any(AttrKeyError, err))
			return api.DeviceHeartbeatByApiKey500JSONResponse(errorMsgResponse("Failed to checkin device")), nil
		}
	}
	logger.Info("apikey heartbeat successful")

	if isNew {
		return api.DeviceHeartbeatByApiKey201JSONResponse(toAddressResponse(address)), nil
	}

	return api.DeviceHeartbeatByApiKey200JSONResponse(toAddressResponse(address)), nil
}

func (h *HTTPHandler) ApiKeyAuthenticator() ApiKeyAuthenticator {
	return h.service
}
func toDeviceResponse(d *DeviceWithApiKeyPrefix) api.Device {
	return api.Device{
		Id:           d.ID.Int64(),
		Name:         d.Name,
		CreatedAt:    d.CreatedAt,
		ApiKeyPrefix: d.KeyPrefix,
	}
}

func toAddressResponse(a *AddressWithStatus) api.Address {
	return api.Address{
		Id:        a.Id.Int64(),
		DeviceId:  a.DeviceId.Int64(),
		Ip:        a.IP,
		Status:    a.Status,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt.Time,
	}
}

func errorMsgResponse(errorMsg string) api.ErrorResponse {
	return api.ErrorResponse{Error: &errorMsg}
}
