package device

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/httpapi"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/logging"
)

type HTTPHandler struct {
	service *Service
}

func NewHandler(service *Service) *HTTPHandler {
	return &HTTPHandler{service: service}
}

func (h *HTTPHandler) GetDevices(ctx context.Context, _ httpapi.GetDevicesRequestObject) (httpapi.GetDevicesResponseObject, error) {
	ctx, logger := logging.Enrich(ctx, slog.String(AttrKeyOperation, "GetDevices"))

	devices, err := h.service.GetDevices(ctx)
	if err != nil {
		logger.Error("failed to list devices", slog.Any(AttrKeyError, err))
		return httpapi.GetDevices500JSONResponse(errorMsgResponse("Error fetching devices")), nil
	}

	logger.Info("devices listed", slog.Int(AttrKeyCount, len(devices)))

	apiDevices := make([]httpapi.Device, len(devices))
	for i := range devices {
		apiDevices[i] = toDeviceResponse(&devices[i])
	}

	return httpapi.GetDevices200JSONResponse(apiDevices), nil
}

func (h *HTTPHandler) CreateDevice(ctx context.Context, request httpapi.CreateDeviceRequestObject) (httpapi.CreateDeviceResponseObject, error) {
	deviceName := request.Body.Name
	ctx, logger := logging.Enrich(ctx,
		slog.String(AttrKeyOperation, "CreateDevice"),
		slog.String(AttrKeyDeviceName, deviceName),
	)

	device, rawAPIKey, err := h.service.CreateDevice(ctx, deviceName)
	if err != nil {
		logger.Error("failed to create device", slog.Any(AttrKeyError, err))
		return httpapi.CreateDevice500JSONResponse(errorMsgResponse("Failed to create device")), nil
	}

	apiDevice := toDeviceResponse(device)
	return httpapi.CreateDevice201JSONResponse(httpapi.CreateDeviceResponse{
		Device: apiDevice,
		ApiKey: rawAPIKey,
	}), nil
}

func (h *HTTPHandler) GetDeviceAddresses(ctx context.Context, request httpapi.GetDeviceAddressesRequestObject) (httpapi.GetDeviceAddressesResponseObject, error) {
	deviceID := DeviceID(request.DeviceId)
	ctx, logger := logging.Enrich(ctx,
		slog.String(AttrKeyOperation, "GetDeviceAddresses"),
		slog.Int64(AttrKeyDeviceID, deviceID.Int64()),
	)

	addresses, err := h.service.GetAddressesForDevice(ctx, deviceID)
	if err != nil {
		if errors.Is(err, ErrDeviceNotFound) {
			logger.Warn("device not found")
			return httpapi.GetDeviceAddresses404JSONResponse(
				errorMsgResponse(fmt.Sprintf("Device with id %s not found", deviceID)),
			), nil
		}

		logger.Error("failed to list device addresses", slog.Any(AttrKeyError, err))
		return httpapi.GetDeviceAddresses500JSONResponse(errorMsgResponse("Failed to list device IPs")), nil
	}

	logger.Info("device addresses listed", slog.Int(AttrKeyCount, len(addresses)))

	addressesResponse := make([]httpapi.Address, len(addresses))
	for i := range addresses {
		addressesResponse[i] = toAddressResponse(&addresses[i])
	}

	return httpapi.GetDeviceAddresses200JSONResponse(addressesResponse), nil
}

func (h *HTTPHandler) AddAddress(ctx context.Context, request httpapi.AddAddressRequestObject) (httpapi.AddAddressResponseObject, error) {
	deviceID := DeviceID(request.DeviceId)
	ipAddress := request.Body.Ip
	ctx, logger := logging.Enrich(ctx,
		slog.String(AttrKeyOperation, "AddAddress"),
		slog.Int64(AttrKeyDeviceID, deviceID.Int64()),
		slog.String(AttrKeyAddressIP, ipAddress),
	)

	addressWithIP, wasCreated, err := h.service.AssignAddress(ctx, deviceID, ipAddress)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidIPFormat):
			logger.Warn("invalid request body")
			return httpapi.AddAddress400JSONResponse(errorMsgResponse(fmt.Sprintf("Received address %s is not a valid IPv4 or IPv6 address", ipAddress))), nil
		case errors.Is(err, ErrDeviceNotFound):
			logger.Warn("device not found")
			return httpapi.AddAddress404JSONResponse(errorMsgResponse(fmt.Sprintf("Device with id %s not found", deviceID))), nil
		default:
			logger.Error("failed to assign address", slog.Any(AttrKeyError, err))
			return httpapi.AddAddress500JSONResponse(errorMsgResponse("Failed to assign address")), nil
		}
	}

	logger.Info("Address added or enabled")

	if wasCreated {
		return httpapi.AddAddress201JSONResponse(toAddressResponse(addressWithIP)), nil
	}

	return httpapi.AddAddress200JSONResponse(toAddressResponse(addressWithIP)), nil
}

func (h *HTTPHandler) DisableAddress(ctx context.Context, request httpapi.DisableAddressRequestObject) (httpapi.DisableAddressResponseObject, error) {
	deviceID := DeviceID(request.DeviceId)
	addressID := AddressID(request.AddressId)
	ctx, logger := logging.Enrich(ctx,
		slog.String(AttrKeyOperation, "DisableAddress"),
		slog.Int64(AttrKeyDeviceID, deviceID.Int64()),
		slog.Int64(AttrKeyAddressID, addressID.Int64()),
	)

	address, err := h.service.DisableAddress(ctx, deviceID, addressID)
	if err != nil {
		if errors.Is(err, ErrAddressNotFound) || errors.Is(err, ErrAddressNotOwnedByDevice) {
			logger.Warn("address not found or not owned by device")
			return httpapi.DisableAddress404JSONResponse(errorMsgResponse(fmt.Sprintf("Address id %s for device id %s not found or already disabled", addressID, deviceID))), nil
		}
		logger.Error("failed to disable address", slog.Any(AttrKeyError, err))
		return httpapi.DisableAddress500JSONResponse(errorMsgResponse("Failed to disable address")), nil
	}

	return httpapi.DisableAddress200JSONResponse(toAddressResponse(address)), nil
}

func (h *HTTPHandler) DeviceHeartbeat(ctx context.Context, request httpapi.DeviceHeartbeatRequestObject) (httpapi.DeviceHeartbeatResponseObject, error) {
	deviceID := DeviceID(request.DeviceId)

	ctx, logger := logging.Enrich(ctx,
		slog.String(AttrKeyOperation, "DeviceHeartbeat"),
		slog.Int64(AttrKeyDeviceID, deviceID.Int64()),
	)

	clientIP, ok := httpapi.ClientIPFromContext(ctx)
	if !ok {
		logger.Error("client IP not in context")
		return httpapi.DeviceHeartbeat500JSONResponse(errorMsgResponse("Failed to extract client IP address")), nil
	}
	ctx, logger = logging.Enrich(ctx, slog.String(AttrKeyClientIP, clientIP))

	address, isNew, err := h.service.AssignAddress(ctx, deviceID, clientIP)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidIPFormat):
			logger.Warn("invalid request body")
			return httpapi.DeviceHeartbeat400JSONResponse(errorMsgResponse(fmt.Sprintf("Received address %s is not a valid IPv4 or IPv6 address", clientIP))), nil
		case errors.Is(err, ErrDeviceNotFound):
			logger.Warn("device not found")
			return httpapi.DeviceHeartbeat404JSONResponse(errorMsgResponse(fmt.Sprintf("Device with id %s not found", deviceID))), nil
		default:
			logger.Error("heartbeat request failed", slog.Any(AttrKeyError, err))
			return httpapi.DeviceHeartbeat500JSONResponse(errorMsgResponse("Failed to checkin device")), nil
		}
	}

	logger.Info("device heartbeat successful")
	if isNew {
		return httpapi.DeviceHeartbeat201JSONResponse(toAddressResponse(address)), nil
	}

	return httpapi.DeviceHeartbeat200JSONResponse(toAddressResponse(address)), nil
}

func (h *HTTPHandler) DeviceHeartbeatByAPIKey(ctx context.Context, request httpapi.DeviceHeartbeatByAPIKeyRequestObject) (httpapi.DeviceHeartbeatByAPIKeyResponseObject, error) {
	ctx, logger := logging.Enrich(ctx, slog.String(AttrKeyOperation, "DeviceHeartbeatByAPIKey"))

	// Extract deviceID from context
	principal, ok := PrincipalFromContext(ctx)
	if !ok {
		logger.Error("invalid API key")
		return httpapi.DeviceHeartbeatByAPIKey500JSONResponse(errorMsgResponse("Failed to extract device api key")), nil
	}
	deviceID := principal.DeviceID
	ctx, logger = logging.Enrich(ctx, slog.Int64(AttrKeyDeviceID, deviceID.Int64()))

	// Determine IP to use: prefer body IP, fallback to context IP if not provided
	var ipToUse string
	requestBody := request.Body
	if requestBody != nil && requestBody.Ip != nil && *requestBody.Ip != "" {
		ipToUse = *requestBody.Ip
	} else {
		var ok bool
		ipToUse, ok = httpapi.ClientIPFromContext(ctx)
		if !ok {
			logger.Error("client IP not in context and no IP provided in body")
			return httpapi.DeviceHeartbeatByAPIKey500JSONResponse(errorMsgResponse("Failed to extract client IP address")), nil
		}
	}
	ctx, logger = logging.Enrich(ctx, slog.String(AttrKeyAddressIP, ipToUse))

	address, isNew, err := h.service.AssignAddress(ctx, deviceID, ipToUse)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidIPFormat):
			logger.Warn("invalid request body")
			return httpapi.DeviceHeartbeatByAPIKey400JSONResponse(errorMsgResponse(fmt.Sprintf("Received address %s is not a valid IPv4 or IPv6 address", ipToUse))), nil
		case errors.Is(err, ErrDeviceNotFound):
			logger.Warn("device not found")
			return httpapi.DeviceHeartbeatByAPIKey404JSONResponse(errorMsgResponse(fmt.Sprintf("Device with id %s not found", deviceID))), nil
		default:
			logger.Error("heartbeat request failed", slog.Any(AttrKeyError, err))
			return httpapi.DeviceHeartbeatByAPIKey500JSONResponse(errorMsgResponse("Failed to checkin device")), nil
		}
	}
	logger.Info("apikey heartbeat successful")

	if isNew {
		return httpapi.DeviceHeartbeatByAPIKey201JSONResponse(toAddressResponse(address)), nil
	}

	return httpapi.DeviceHeartbeatByAPIKey200JSONResponse(toAddressResponse(address)), nil
}

func (h *HTTPHandler) APIKeyAuthenticator() APIKeyAuthenticator {
	return h.service
}
func toDeviceResponse(d *DeviceWithAPIKeyPrefix) httpapi.Device {
	return httpapi.Device{
		Id:           d.ID.Int64(),
		Name:         d.Name,
		CreatedAt:    d.CreatedAt,
		ApiKeyPrefix: d.KeyPrefix,
	}
}

func toAddressResponse(a *AddressWithStatus) httpapi.Address {
	return httpapi.Address{
		Id:        a.ID.Int64(),
		DeviceId:  a.DeviceID.Int64(),
		Ip:        a.IP,
		Status:    a.Status,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt.Time,
	}
}

func errorMsgResponse(errorMsg string) httpapi.ErrorResponse {
	return httpapi.ErrorResponse{Error: &errorMsg}
}
