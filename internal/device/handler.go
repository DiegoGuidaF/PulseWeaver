package device

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
	"github.com/DiegoGuidaF/PulseWeaver/internal/timebucket"
)

type HTTPHandler struct {
	service *Service
	logger  *slog.Logger
}

func NewHTTPHandler(service *Service, logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		service: service,
		logger:  logger.With(slog.String(logging.AttrKeyComponent, "device")),
	}
}

func (h *HTTPHandler) CreateDevice(ctx context.Context, request httpapi.CreateDeviceRequestObject) (httpapi.CreateDeviceResponseObject, error) {
	ctx = logging.WithOperation(ctx, "CreateDevice")
	deviceName := request.Body.Name
	logger := h.logger.With(slog.String(AttrKeyDeviceName, deviceName))

	principal, ok := auth.PrincipalFromContext(ctx)
	if !ok {
		return httpapi.CreateDevice500JSONResponse(errorMsgResponse("Not authenticated")), nil
	}

	input := CreateDeviceInput{Name: deviceName}
	if request.Body.OwnerId != nil {
		input.OwnerID = new(ids.UserID(*request.Body.OwnerId))
	}
	if request.Body.DeviceType != nil {
		input.DeviceType = string(*request.Body.DeviceType)
	}
	if request.Body.Description.Set {
		input.Description = request.Body.Description.Value
	}
	if request.Body.Icon.Set {
		input.Icon = request.Body.Icon.Value
	}
	if request.Body.GenerateApiKey != nil {
		input.GenerateAPIKey = *request.Body.GenerateApiKey
	}

	device, rawKey, err := h.service.CreateDeviceWithOptions(ctx, principal, input)
	if err != nil {
		switch {
		case errors.Is(err, ErrDuplicateDeviceName):
			logger.WarnContext(ctx, "duplicate device name")
			return httpapi.CreateDevice409JSONResponse(errorMsgResponse("Device name already in use")), nil
		case errors.Is(err, ErrOwnerNotFound):
			logger.WarnContext(ctx, "owner not found", slog.Any(AttrKeyError, err))
			return httpapi.CreateDevice400JSONResponse(errorMsgResponse("Specified owner does not exist")), nil
		default:
			logger.ErrorContext(ctx, "failed to create device", slog.Any(AttrKeyError, err))
			return httpapi.CreateDevice500JSONResponse(errorMsgResponse("Failed to create device")), nil
		}
	}
	logger.InfoContext(ctx, "device created", slog.Int64(AttrKeyDeviceID, device.ID.Int64()))

	result := httpapi.CreateDeviceResult{Device: toDeviceResponse(device)}
	if rawKey != "" {
		result.ApiKey = &rawKey
	}
	return httpapi.CreateDevice201JSONResponse(result), nil
}

func (h *HTTPHandler) DeleteDevice(ctx context.Context, request httpapi.DeleteDeviceRequestObject) (httpapi.DeleteDeviceResponseObject, error) {
	ctx = logging.WithOperation(ctx, "DeleteDevice")
	deviceID := ids.DeviceID(request.DeviceId)
	logger := h.logger.With(slog.Int64(AttrKeyDeviceID, deviceID.Int64()))

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
	logger.InfoContext(ctx, "device deleted", slog.Int64(AttrKeyDeviceID, deviceID.Int64()))

	return httpapi.DeleteDevice204Response{}, nil
}

func (h *HTTPHandler) DisableDevice(ctx context.Context, request httpapi.DisableDeviceRequestObject) (httpapi.DisableDeviceResponseObject, error) {
	ctx = logging.WithOperation(ctx, "DisableDevice")
	deviceID := ids.DeviceID(request.DeviceId)
	logger := h.logger.With(slog.Int64(AttrKeyDeviceID, deviceID.Int64()))

	device, err := h.service.DisableDevice(ctx, deviceID)
	if err != nil {
		switch {
		case errors.Is(err, ErrDeviceNotFound):
			logger.WarnContext(ctx, "device not found")
			return httpapi.DisableDevice404JSONResponse(errorMsgResponse(fmt.Sprintf("Device with id %s not found", deviceID))), nil
		default:
			logger.ErrorContext(ctx, "failed to disable device", slog.Any(AttrKeyError, err))
			return httpapi.DisableDevice500JSONResponse(errorMsgResponse("Failed to disable device")), nil
		}
	}
	logger.InfoContext(ctx, "device disabled")

	return httpapi.DisableDevice200JSONResponse(toDeviceResponse(device)), nil
}

func (h *HTTPHandler) RegenerateDeviceAPIKey(ctx context.Context, request httpapi.RegenerateDeviceAPIKeyRequestObject) (httpapi.RegenerateDeviceAPIKeyResponseObject, error) {
	ctx = logging.WithOperation(ctx, "RegenerateDeviceAPIKey")
	deviceID := ids.DeviceID(request.DeviceId)
	logger := h.logger.With(slog.Int64(AttrKeyDeviceID, deviceID.Int64()))

	device, rawAPIKey, err := h.service.RegenerateAPIKey(ctx, deviceID)
	if err != nil {
		switch {
		case errors.Is(err, ErrDeviceNotFound):
			logger.WarnContext(ctx, "device not found")
			return httpapi.RegenerateDeviceAPIKey404JSONResponse(errorMsgResponse(fmt.Sprintf("Device with id %s not found", deviceID))), nil
		default:
			logger.ErrorContext(ctx, "failed to regenerate api key", slog.Any(AttrKeyError, err))
			return httpapi.RegenerateDeviceAPIKey500JSONResponse(errorMsgResponse("Failed to regenerate API key")), nil
		}
	}
	logger.InfoContext(ctx, "device api key regenerated")

	return httpapi.RegenerateDeviceAPIKey200JSONResponse(httpapi.DeviceAPIKeyResponse{
		Device: toDeviceResponse(device),
		ApiKey: rawAPIKey,
	}), nil
}

func (h *HTTPHandler) DeleteDeviceAPIKey(ctx context.Context, request httpapi.DeleteDeviceAPIKeyRequestObject) (httpapi.DeleteDeviceAPIKeyResponseObject, error) {
	ctx = logging.WithOperation(ctx, "DeleteDeviceAPIKey")
	deviceID := ids.DeviceID(request.DeviceId)
	logger := h.logger.With(slog.Int64(AttrKeyDeviceID, deviceID.Int64()))

	err := h.service.DeleteAPIKey(ctx, deviceID)
	if err != nil {
		switch {
		case errors.Is(err, ErrDeviceNotFound):
			logger.WarnContext(ctx, "device not found")
			return httpapi.DeleteDeviceAPIKey404JSONResponse(errorMsgResponse(fmt.Sprintf("Device with id %s not found", deviceID))), nil
		case errors.Is(err, ErrNoAPIKey):
			logger.WarnContext(ctx, "device has no api key")
			return httpapi.DeleteDeviceAPIKey404JSONResponse(errorMsgResponse(fmt.Sprintf("Device with id %s has no API key", deviceID))), nil
		default:
			logger.ErrorContext(ctx, "failed to delete api key", slog.Any(AttrKeyError, err))
			return httpapi.DeleteDeviceAPIKey500JSONResponse(errorMsgResponse("Failed to delete API key")), nil
		}
	}
	logger.InfoContext(ctx, "device api key deleted")

	return httpapi.DeleteDeviceAPIKey204Response{}, nil
}

func (h *HTTPHandler) AddAddress(ctx context.Context, request httpapi.AddAddressRequestObject) (httpapi.AddAddressResponseObject, error) {
	ctx = logging.WithOperation(ctx, "AddAddress")
	deviceID := ids.DeviceID(request.DeviceId)
	ipAddress := request.Body.Ip
	logger := h.logger.With(
		slog.Int64(AttrKeyDeviceID, deviceID.Int64()),
		slog.String(AttrKeyAddressIP, ipAddress),
	)

	address, eventType, err := h.service.RegisterAddressActivity(ctx, deviceID, ipAddress, EventSourceManual)
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
	logger.InfoContext(ctx,
		"manual address register successful",
		slog.Int64(AttrKeyAddressID, address.ID.Int64()),
		slog.String(AttrKeyAddressEventType, string(eventType)),
	)

	if eventType == EventTypeAddressCreated {
		return httpapi.AddAddress201JSONResponse(toAddressResponse(address)), nil
	}

	return httpapi.AddAddress200JSONResponse(toAddressResponse(address)), nil
}

func (h *HTTPHandler) DeviceHeartbeat(ctx context.Context, request httpapi.DeviceHeartbeatRequestObject) (httpapi.DeviceHeartbeatResponseObject, error) {
	ctx = logging.WithOperation(ctx, "DeviceHeartbeat")
	deviceID := ids.DeviceID(request.DeviceId)
	logger := h.logger.With(slog.Int64(AttrKeyDeviceID, deviceID.Int64()))

	clientIP, ok := httpapi.ClientIPFromContext(ctx)
	if !ok {
		logger.ErrorContext(ctx, "client IP not in context")
		return httpapi.DeviceHeartbeat500JSONResponse(errorMsgResponse("Failed to extract client IP address")), nil
	}
	logger = logger.With(slog.String(AttrKeyClientIP, clientIP))

	address, eventType, err := h.service.RegisterAddressActivity(ctx, deviceID, clientIP, EventSourceHeartbeat)
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

	logger.InfoContext(ctx,
		"manual device heartbeat successful",
		slog.Int64(AttrKeyAddressID, address.ID.Int64()),
		slog.String(AttrKeyAddressEventType, string(eventType)),
	)

	if eventType == EventTypeAddressCreated {
		return httpapi.DeviceHeartbeat201JSONResponse(toAddressResponse(address)), nil
	}

	return httpapi.DeviceHeartbeat200JSONResponse(toAddressResponse(address)), nil
}

func (h *HTTPHandler) DeviceHeartbeatByAPIKey(ctx context.Context, request httpapi.DeviceHeartbeatByAPIKeyRequestObject) (httpapi.DeviceHeartbeatByAPIKeyResponseObject, error) {
	ctx = logging.WithOperation(ctx, "DeviceHeartbeatByAPIKey")
	logger := h.logger

	// Extract deviceID from context
	principal, ok := PrincipalFromContext(ctx)
	if !ok {
		logger.ErrorContext(ctx, "invalid API key")
		return httpapi.DeviceHeartbeatByAPIKey500JSONResponse(errorMsgResponse("Failed to extract device api key")), nil
	}
	deviceID := principal.DeviceID
	logger = logger.With(slog.Int64(AttrKeyDeviceID, deviceID.Int64()))

	clientIP, ok := httpapi.ClientIPFromContext(ctx)
	if !ok {
		logger.ErrorContext(ctx, "client IP not in context")
		return httpapi.DeviceHeartbeatByAPIKey500JSONResponse(errorMsgResponse("Failed to extract client IP address")), nil
	}
	logger = logger.With(slog.String(AttrKeyClientIP, clientIP))

	address, eventType, err := h.service.RegisterAddressActivity(ctx, deviceID, clientIP, EventSourceHeartbeat)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidIPFormat):
			logger.WarnContext(ctx, "invalid request body")
			return httpapi.DeviceHeartbeatByAPIKey400JSONResponse(errorMsgResponse(fmt.Sprintf("Received address %s is not a valid IPv4 or IPv6 address", clientIP))), nil
		case errors.Is(err, ErrInvalidDeviceIP):
			logger.WarnContext(ctx, "invalid device IP address rejected")
			return httpapi.DeviceHeartbeatByAPIKey400JSONResponse(errorMsgResponse(fmt.Sprintf("Address %s cannot be registered (loopback, multicast, unspecified, or link-local addresses are not allowed)", clientIP))), nil
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

	logger.DebugContext(ctx,
		"apikey device heartbeat successful",
		slog.Int64(AttrKeyAddressID, address.ID.Int64()),
		slog.String(AttrKeyAddressEventType, string(eventType)),
	)

	if eventType == EventTypeAddressCreated {
		return httpapi.DeviceHeartbeatByAPIKey201JSONResponse(toAddressResponse(address)), nil
	}
	return httpapi.DeviceHeartbeatByAPIKey200JSONResponse(toAddressResponse(address)), nil
}

func (h *HTTPHandler) DisableAddress(ctx context.Context, request httpapi.DisableAddressRequestObject) (httpapi.DisableAddressResponseObject, error) {
	ctx = logging.WithOperation(ctx, "DisableAddress")
	deviceID := ids.DeviceID(request.DeviceId)
	addressID := ids.AddressID(request.AddressId)
	logger := h.logger.With(
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
	logger.DebugContext(ctx, "address disabled")

	return httpapi.DisableAddress200JSONResponse(toAddressResponse(address)), nil
}

func (h *HTTPHandler) GetAddressHistory(ctx context.Context, request httpapi.GetAddressHistoryRequestObject) (httpapi.GetAddressHistoryResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetAddressHistory")
	logger := h.logger

	params := request.Params

	query := AddressHistoryQuery{
		IsEnabled: params.IsEnabled,
		IP:        params.Ip,
		BeforeID:  params.BeforeId,
		Source:    (*string)(params.Source),
	}
	if params.From != nil {
		query.From = *params.From
	}
	if params.To != nil {
		query.To = *params.To
	}
	if params.Granularity != nil {
		query.Granularity = timebucket.Granularity(*params.Granularity)
	}
	if params.DeviceId != nil {
		for _, id := range *params.DeviceId {
			query.DeviceIDs = append(query.DeviceIDs, ids.DeviceID(id))
		}
	}
	if params.Limit != nil {
		query.Limit = *params.Limit
	}
	if params.IncludeAll != nil {
		query.IncludeAll = *params.IncludeAll
	}

	history, err := h.service.GetAddressHistory(ctx, query)
	if err != nil {
		switch {
		case errors.Is(err, timebucket.ErrInvalidGranularity):
			logger.WarnContext(ctx, "invalid query parameters", slog.Any(AttrKeyError, err))
			return httpapi.GetAddressHistory400JSONResponse(errorMsgResponse(err.Error())), nil
		default:
			logger.ErrorContext(ctx, "failed to get address history", slog.Any(AttrKeyError, err))
			return httpapi.GetAddressHistory500JSONResponse(errorMsgResponse("Failed to get address history")), nil
		}
	}

	return httpapi.GetAddressHistory200JSONResponse(toAddressHistoryResponse(history, history.QueryLimit)), nil
}

func (h *HTTPHandler) UpdateDevice(ctx context.Context, request httpapi.UpdateDeviceRequestObject) (httpapi.UpdateDeviceResponseObject, error) {
	ctx = logging.WithOperation(ctx, "UpdateDevice")
	deviceID := ids.DeviceID(request.DeviceId)
	logger := h.logger.With(slog.Int64(AttrKeyDeviceID, deviceID.Int64()))

	body := request.Body

	input := UpdateDeviceInput{
		Name: body.Name,
	}
	if body.DeviceType != nil {
		input.DeviceType = new(string(*body.DeviceType))
	}
	if body.Description.Set {
		input.Description = &body.Description.Value
	}
	if body.Icon.Set {
		input.Icon = &body.Icon.Value
	}
	if body.OwnerId != nil {
		input.OwnerID = new(ids.UserID(*body.OwnerId))
	}

	device, err := h.service.UpdateDevice(ctx, deviceID, input)
	if err != nil {
		switch {
		case errors.Is(err, ErrDeviceNotFound):
			logger.WarnContext(ctx, "device not found")
			return httpapi.UpdateDevice404JSONResponse(errorMsgResponse(fmt.Sprintf("Device with id %s not found", deviceID))), nil
		case errors.Is(err, ErrDuplicateDeviceName):
			logger.WarnContext(ctx, "duplicate device name")
			return httpapi.UpdateDevice409JSONResponse(errorMsgResponse("Device name already in use")), nil
		case errors.Is(err, ErrInvalidDeviceType), errors.Is(err, ErrInvalidDeviceName),
			errors.Is(err, ErrDescriptionTooLong), errors.Is(err, ErrIconTooLong),
			errors.Is(err, ErrOwnerNotFound):
			logger.WarnContext(ctx, "invalid update request", slog.Any(AttrKeyError, err))
			return httpapi.UpdateDevice400JSONResponse(errorMsgResponse(err.Error())), nil
		case errors.Is(err, auth.ErrAdminCredentialsRequired):
			logger.WarnContext(ctx, "non-admin attempted owner reassignment")
			return httpapi.UpdateDevice403JSONResponse(errorMsgResponse("Admin credentials required")), nil
		default:
			logger.ErrorContext(ctx, "failed to update device", slog.Any(AttrKeyError, err))
			return httpapi.UpdateDevice500JSONResponse(errorMsgResponse("Failed to update device")), nil
		}
	}

	logger.InfoContext(ctx, "device updated")
	return httpapi.UpdateDevice200JSONResponse(toDeviceResponse(device)), nil
}

func (h *HTTPHandler) ListDeviceTypes(_ context.Context, _ httpapi.ListDeviceTypesRequestObject) (httpapi.ListDeviceTypesResponseObject, error) {
	result := make([]httpapi.DeviceTypeItem, 0, len(AllowedDeviceTypes))
	for _, dt := range AllowedDeviceTypes {
		label := DeviceTypeLabels[dt]
		result = append(result, httpapi.DeviceTypeItem{
			Value: string(dt),
			Label: label,
		})
	}
	return httpapi.ListDeviceTypes200JSONResponse(result), nil
}

func (h *HTTPHandler) APIKeyAuthenticator() APIKeyAuthenticator {
	return h.service
}

func toDeviceResponse(d *Device) httpapi.Device {
	resp := httpapi.Device{
		Id:           d.ID.Int64(),
		Name:         d.Name,
		DeviceType:   httpapi.DeviceType(d.DeviceType),
		Description:  d.Description,
		Icon:         d.Icon,
		CreatedAt:    httpapi.UTCTime(d.CreatedAt),
		UpdatedAt:    httpapi.UTCTime(d.UpdatedAt),
		ApiKeyPrefix: d.KeyPrefix,
		OwnerId:      d.OwnerID.Int64(),
	}
	if d.DisabledAt != nil {
		resp.DisabledAt = new(httpapi.UTCTime(*d.DisabledAt))
	}
	return resp
}

func toAddressResponse(a *Address) httpapi.Address {
	return httpapi.Address{
		Id:        a.ID.Int64(),
		DeviceId:  a.DeviceID.Int64(),
		Ip:        a.IP,
		IsEnabled: a.IsEnabled,
		Source:    a.Source,
		CreatedAt: httpapi.UTCTime(a.CreatedAt),
		UpdatedAt: httpapi.UTCTime(a.UpdatedAt),
	}
}

func toAddressHistoryResponse(h AddressHistory, queryLimit int) httpapi.AddressHistoryResponse {
	buckets := make([]httpapi.AddressHistoryBucket, len(h.Buckets))
	for i, b := range h.Buckets {
		buckets[i] = httpapi.AddressHistoryBucket{
			Timestamp:   httpapi.UTCTime(b.Timestamp.Time),
			ActiveCount: b.ActiveCount,
			GapCount:    b.GapCount,
			EventCount:  b.EventCount,
		}
	}

	events := make([]httpapi.AddressHistoryEvent, len(h.Events))
	for i, e := range h.Events {
		events[i] = httpapi.AddressHistoryEvent{
			Id:         e.ID,
			Timestamp:  httpapi.UTCTime(e.CreatedAt),
			Ip:         e.IP,
			IsEnabled:  e.IsEnabled,
			Source:     e.Source,
			DeviceId:   e.DeviceID.Int64(),
			DeviceName: e.DeviceName,
		}
	}

	// Use len == limit as "has more" signal — reliable across all pages,
	// unlike comparing against TotalEvents which ignores the cursor offset.
	var nextCursor *int64
	if len(h.Events) == queryLimit {
		nextCursor = &h.Events[len(h.Events)-1].ID
	}

	return httpapi.AddressHistoryResponse{
		Buckets:     buckets,
		Events:      events,
		TotalEvents: h.TotalEvents,
		NextCursor:  nextCursor,
	}
}

func errorMsgResponse(errorMsg string) httpapi.ErrorResponse {
	return httpapi.ErrorResponse{Error: &errorMsg}
}
