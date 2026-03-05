package device

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/DiegoGuidaF/WallyDex/internal/httpapi"
	"github.com/DiegoGuidaF/WallyDex/internal/logging"
)

// TODO: Rename to Handler
type HTTPHandler struct {
	service *Service
	logger  *slog.Logger
}

func NewHandler(service *Service, logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		service: service,
		logger:  logger.With(slog.String(logging.AttrKeyComponent, "device")),
	}
}

func (h *HTTPHandler) GetDevices(ctx context.Context, _ httpapi.GetDevicesRequestObject) (httpapi.GetDevicesResponseObject, error) {
	logger := h.logger.With(slog.String(AttrKeyOperation, "GetDevices"))

	devices, err := h.service.GetDevices(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "failed to list devices", slog.Any(AttrKeyError, err))
		return httpapi.GetDevices500JSONResponse(errorMsgResponse("Error fetching devices")), nil
	}

	logger.InfoContext(ctx, "devices listed", slog.Int(AttrKeyCount, len(devices)))

	apiDevices := make([]httpapi.Device, len(devices))
	for i := range devices {
		apiDevices[i] = toDeviceResponse(&devices[i])
	}

	return httpapi.GetDevices200JSONResponse(apiDevices), nil
}

func (h *HTTPHandler) CreateDevice(ctx context.Context, request httpapi.CreateDeviceRequestObject) (httpapi.CreateDeviceResponseObject, error) {
	deviceName := request.Body.Name
	logger := h.logger.With(
		slog.String(AttrKeyOperation, "CreateDevice"),
		slog.String(AttrKeyDeviceName, deviceName),
	)

	device, rawAPIKey, err := h.service.CreateDevice(ctx, deviceName)
	if err != nil {
		switch {
		case errors.Is(err, ErrDuplicateDeviceName):
			logger.WarnContext(ctx, "duplicate device name")
			return httpapi.CreateDevice409JSONResponse(errorMsgResponse("Device name already in use")), nil
		default:
			logger.ErrorContext(ctx, "failed to create device", slog.Any(AttrKeyError, err))
			return httpapi.CreateDevice500JSONResponse(errorMsgResponse("Failed to create device")), nil
		}
	}

	apiDevice := toDeviceResponse(device)
	return httpapi.CreateDevice201JSONResponse(httpapi.CreateDeviceResponse{
		Device: apiDevice,
		ApiKey: rawAPIKey,
	}), nil
}

func (h *HTTPHandler) GetDevice(ctx context.Context, request httpapi.GetDeviceRequestObject) (httpapi.GetDeviceResponseObject, error) {
	deviceID := DeviceID(request.DeviceId)
	logger := h.logger.With(
		slog.String(AttrKeyOperation, "GetDevice"),
		slog.Int64(AttrKeyDeviceID, deviceID.Int64()),
	)

	device, err := h.service.GetDevice(ctx, deviceID)
	if err != nil {
		switch {
		case errors.Is(err, ErrDeviceNotFound):
			logger.WarnContext(ctx, "device not found")
			return httpapi.GetDevice404JSONResponse(errorMsgResponse(fmt.Sprintf("Device with id %s not found", deviceID))), nil
		default:
			logger.ErrorContext(ctx, "failed to get device", slog.Any(AttrKeyError, err))
			return httpapi.GetDevice500JSONResponse(errorMsgResponse("Failed to get device")), nil
		}
	}

	return httpapi.GetDevice200JSONResponse(toDeviceResponse(device)), nil
}

func (h *HTTPHandler) DeleteDevice(ctx context.Context, request httpapi.DeleteDeviceRequestObject) (httpapi.DeleteDeviceResponseObject, error) {
	deviceID := DeviceID(request.DeviceId)
	logger := h.logger.With(
		slog.String(AttrKeyOperation, "DeleteDevice"),
		slog.Int64(AttrKeyDeviceID, deviceID.Int64()),
	)

	err := h.service.DeleteDevice(ctx, deviceID)
	if err != nil {
		switch {
		case errors.Is(err, ErrDeviceNotFound):
			logger.WarnContext(ctx, "device not found")
			return httpapi.DeleteDevice404JSONResponse(errorMsgResponse(fmt.Sprintf("Device with id %s not found", deviceID))), nil
		default:
			logger.ErrorContext(ctx, "failed to delete device", slog.Any(AttrKeyError, err))
			return httpapi.DeleteDevice500JSONResponse(errorMsgResponse("Failed to delete device")), nil
		}
	}

	return httpapi.DeleteDevice204Response{}, nil
}

func (h *HTTPHandler) GetDeviceAddresses(ctx context.Context, request httpapi.GetDeviceAddressesRequestObject) (httpapi.GetDeviceAddressesResponseObject, error) {
	deviceID := DeviceID(request.DeviceId)
	logger := h.logger.With(
		slog.String(AttrKeyOperation, "GetDeviceAddresses"),
		slog.Int64(AttrKeyDeviceID, deviceID.Int64()),
	)

	addresses, err := h.service.GetAddressesForDevice(ctx, deviceID)
	if err != nil {
		if errors.Is(err, ErrDeviceNotFound) {
			logger.WarnContext(ctx, "device not found")
			return httpapi.GetDeviceAddresses404JSONResponse(
				errorMsgResponse(fmt.Sprintf("Device with id %s not found", deviceID)),
			), nil
		}

		logger.ErrorContext(ctx, "failed to list device addresses", slog.Any(AttrKeyError, err))
		return httpapi.GetDeviceAddresses500JSONResponse(errorMsgResponse("Failed to list device IPs")), nil
	}

	logger.InfoContext(ctx, "device addresses listed", slog.Int(AttrKeyCount, len(addresses)))

	addressesResponse := make([]httpapi.Address, len(addresses))
	for i := range addresses {
		addressesResponse[i] = toAddressResponse(&addresses[i])
	}

	return httpapi.GetDeviceAddresses200JSONResponse(addressesResponse), nil
}

func (h *HTTPHandler) AddAddress(ctx context.Context, request httpapi.AddAddressRequestObject) (httpapi.AddAddressResponseObject, error) {
	deviceID := DeviceID(request.DeviceId)
	ipAddress := request.Body.Ip
	logger := h.logger.With(
		slog.String(AttrKeyOperation, "AddAddress"),
		slog.Int64(AttrKeyDeviceID, deviceID.Int64()),
		slog.String(AttrKeyAddressIP, ipAddress),
	)

	addressWithIP, wasCreated, err := h.service.AssignAddress(ctx, deviceID, ipAddress, StatusSourceManual)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidIPFormat):
			logger.WarnContext(ctx, "invalid request body")
			return httpapi.AddAddress400JSONResponse(errorMsgResponse(fmt.Sprintf("Received address %s is not a valid IPv4 or IPv6 address", ipAddress))), nil
		case errors.Is(err, ErrInvalidDeviceIP):
			logger.WarnContext(ctx, "invalid device IP address rejected")
			return httpapi.AddAddress400JSONResponse(errorMsgResponse(fmt.Sprintf("Address %s cannot be registered (loopback, multicast, unspecified, or link-local addresses are not allowed)", ipAddress))), nil
		case errors.Is(err, ErrTrustedProxyIPRejected):
			logger.WarnContext(ctx, "trusted proxy IP address rejected")
			return httpapi.AddAddress400JSONResponse(errorMsgResponse("Trusted proxy IP addresses cannot be registered")), nil
		case errors.Is(err, ErrDeviceNotFound):
			logger.WarnContext(ctx, "device not found")
			return httpapi.AddAddress404JSONResponse(errorMsgResponse(fmt.Sprintf("Device with id %s not found", deviceID))), nil
		default:
			logger.ErrorContext(ctx, "failed to assign address", slog.Any(AttrKeyError, err))
			return httpapi.AddAddress500JSONResponse(errorMsgResponse("Failed to assign address")), nil
		}
	}

	if wasCreated {
		return httpapi.AddAddress201JSONResponse(toAddressResponse(addressWithIP)), nil
	}

	return httpapi.AddAddress200JSONResponse(toAddressResponse(addressWithIP)), nil
}

func (h *HTTPHandler) DisableAddress(ctx context.Context, request httpapi.DisableAddressRequestObject) (httpapi.DisableAddressResponseObject, error) {
	deviceID := DeviceID(request.DeviceId)
	addressID := AddressID(request.AddressId)
	logger := h.logger.With(
		slog.String(AttrKeyOperation, "DisableAddress"),
		slog.Int64(AttrKeyDeviceID, deviceID.Int64()),
		slog.Int64(AttrKeyAddressID, addressID.Int64()),
	)

	address, err := h.service.DisableAddress(ctx, deviceID, addressID)
	if err != nil {
		switch {
		case errors.Is(err, ErrDeviceNotFound):
			logger.WarnContext(ctx, "device not found")
			return httpapi.DisableAddress404JSONResponse(errorMsgResponse(fmt.Sprintf("Device with id %s not found", deviceID))), nil
		case errors.Is(err, ErrAddressNotFound) || errors.Is(err, ErrAddressNotOwnedByDevice):
			logger.WarnContext(ctx, "address not found or not owned by device")
			return httpapi.DisableAddress404JSONResponse(errorMsgResponse(fmt.Sprintf("Address id %s for device id %s not found or already disabled", addressID, deviceID))), nil
		default:
			logger.ErrorContext(ctx, "failed to disable address", slog.Any(AttrKeyError, err))
			return httpapi.DisableAddress500JSONResponse(errorMsgResponse("Failed to disable address")), nil
		}
	}

	return httpapi.DisableAddress200JSONResponse(toAddressResponse(address)), nil
}

func (h *HTTPHandler) DeviceHeartbeat(ctx context.Context, request httpapi.DeviceHeartbeatRequestObject) (httpapi.DeviceHeartbeatResponseObject, error) {
	deviceID := DeviceID(request.DeviceId)
	logger := h.logger.With(
		slog.String(AttrKeyOperation, "DeviceHeartbeat"),
		slog.Int64(AttrKeyDeviceID, deviceID.Int64()),
	)

	clientIP, ok := httpapi.ClientIPFromContext(ctx)
	if !ok {
		logger.ErrorContext(ctx, "client IP not in context")
		return httpapi.DeviceHeartbeat500JSONResponse(errorMsgResponse("Failed to extract client IP address")), nil
	}
	logger = logger.With(slog.String(AttrKeyClientIP, clientIP))

	address, wasCreated, err := h.service.AssignAddress(ctx, deviceID, clientIP, StatusSourceHeartbeat)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidIPFormat):
			logger.WarnContext(ctx, "invalid request body")
			return httpapi.DeviceHeartbeat400JSONResponse(errorMsgResponse(fmt.Sprintf("Received address %s is not a valid IPv4 or IPv6 address", clientIP))), nil
		case errors.Is(err, ErrInvalidDeviceIP):
			logger.WarnContext(ctx, "invalid device IP address rejected")
			return httpapi.DeviceHeartbeat400JSONResponse(errorMsgResponse(fmt.Sprintf("Address %s cannot be registered (loopback, multicast, unspecified, or link-local addresses are not allowed)", clientIP))), nil
		case errors.Is(err, ErrTrustedProxyIPRejected):
			logger.WarnContext(ctx, "trusted proxy IP address rejected")
			return httpapi.DeviceHeartbeat400JSONResponse(errorMsgResponse("Trusted proxy IP addresses cannot be registered")), nil
		case errors.Is(err, ErrDeviceNotFound):
			logger.WarnContext(ctx, "device not found")
			return httpapi.DeviceHeartbeat404JSONResponse(errorMsgResponse(fmt.Sprintf("Device with id %s not found", deviceID))), nil
		default:
			logger.ErrorContext(ctx, "heartbeat request failed", slog.Any(AttrKeyError, err))
			return httpapi.DeviceHeartbeat500JSONResponse(errorMsgResponse("Failed to checkin device")), nil
		}
	}

	logger.InfoContext(ctx, "device heartbeat successful")
	if wasCreated {
		return httpapi.DeviceHeartbeat201JSONResponse(toAddressResponse(address)), nil
	}

	return httpapi.DeviceHeartbeat200JSONResponse(toAddressResponse(address)), nil
}

func (h *HTTPHandler) DeviceHeartbeatByAPIKey(ctx context.Context, request httpapi.DeviceHeartbeatByAPIKeyRequestObject) (httpapi.DeviceHeartbeatByAPIKeyResponseObject, error) {
	logger := h.logger.With(slog.String(AttrKeyOperation, "DeviceHeartbeatByAPIKey"))

	// Extract deviceID from context
	principal, ok := PrincipalFromContext(ctx)
	if !ok {
		logger.ErrorContext(ctx, "invalid API key")
		return httpapi.DeviceHeartbeatByAPIKey500JSONResponse(errorMsgResponse("Failed to extract device api key")), nil
	}
	deviceID := principal.DeviceID
	logger = logger.With(slog.Int64(AttrKeyDeviceID, deviceID.Int64()))

	// Determine IP to use: prefer body IP, fallback to context IP if not provided
	var ipToUse string
	requestBody := request.Body
	if requestBody != nil && requestBody.Ip != nil && *requestBody.Ip != "" {
		ipToUse = *requestBody.Ip
	} else {
		var ok bool
		ipToUse, ok = httpapi.ClientIPFromContext(ctx)
		if !ok {
			logger.ErrorContext(ctx, "client IP not in context and no IP provided in body")
			return httpapi.DeviceHeartbeatByAPIKey500JSONResponse(errorMsgResponse("Failed to extract client IP address")), nil
		}
	}
	logger = logger.With(slog.String(AttrKeyAddressIP, ipToUse))

	address, wasCreated, err := h.service.AssignAddress(ctx, deviceID, ipToUse, StatusSourceHeartbeat)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidIPFormat):
			logger.WarnContext(ctx, "invalid request body")
			return httpapi.DeviceHeartbeatByAPIKey400JSONResponse(errorMsgResponse(fmt.Sprintf("Received address %s is not a valid IPv4 or IPv6 address", ipToUse))), nil
		case errors.Is(err, ErrInvalidDeviceIP):
			logger.WarnContext(ctx, "invalid device IP address rejected")
			return httpapi.DeviceHeartbeatByAPIKey400JSONResponse(errorMsgResponse(fmt.Sprintf("Address %s cannot be registered (loopback, multicast, unspecified, or link-local addresses are not allowed)", ipToUse))), nil
		case errors.Is(err, ErrTrustedProxyIPRejected):
			logger.WarnContext(ctx, "trusted proxy IP address rejected")
			return httpapi.DeviceHeartbeatByAPIKey400JSONResponse(errorMsgResponse("Trusted proxy IP addresses cannot be registered")), nil
		case errors.Is(err, ErrDeviceNotFound):
			logger.WarnContext(ctx, "device not found")
			return httpapi.DeviceHeartbeatByAPIKey404JSONResponse(errorMsgResponse(fmt.Sprintf("Device with id %s not found", deviceID))), nil
		default:
			logger.ErrorContext(ctx, "heartbeat request failed", slog.Any(AttrKeyError, err))
			return httpapi.DeviceHeartbeatByAPIKey500JSONResponse(errorMsgResponse("Failed to checkin device")), nil
		}
	}
	logger.InfoContext(ctx, "apikey heartbeat successful")

	if wasCreated {
		return httpapi.DeviceHeartbeatByAPIKey201JSONResponse(toAddressResponse(address)), nil
	}

	return httpapi.DeviceHeartbeatByAPIKey200JSONResponse(toAddressResponse(address)), nil
}

func (h *HTTPHandler) APIKeyAuthenticator() APIKeyAuthenticator {
	return h.service
}
func toDeviceResponse(d *Device) httpapi.Device {
	return httpapi.Device{
		Id:           d.ID.Int64(),
		Name:         d.Name,
		CreatedAt:    d.CreatedAt,
		ApiKeyPrefix: d.KeyPrefix,
	}
}

func toAddressResponse(a *Address) httpapi.Address {
	return httpapi.Address{
		Id:        a.ID.Int64(),
		DeviceId:  a.DeviceID.Int64(),
		Ip:        a.IP,
		Status:    a.Status,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
	}
}

func errorMsgResponse(errorMsg string) httpapi.ErrorResponse {
	return httpapi.ErrorResponse{Error: &errorMsg}
}
