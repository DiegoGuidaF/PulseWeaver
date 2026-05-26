package devicepairing

import (
	"context"
	"errors"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

// HTTPHandler implements the device pairing subset of httpapi.StrictServerInterface.
type HTTPHandler struct {
	service *Service
	logger  *slog.Logger
}

func NewHTTPHandler(svc *Service, logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		service: svc,
		logger:  logger.With(slog.String(logging.AttrKeyComponent, "devicepairing")),
	}
}

func (h *HTTPHandler) ClaimPairing(ctx context.Context, request httpapi.ClaimPairingRequestObject) (httpapi.ClaimPairingResponseObject, error) {
	ctx = logging.WithOperation(ctx, "ClaimPairing")
	logger := h.logger

	result, err := h.service.ClaimPairing(ctx, request.Body.Code)
	if err != nil {
		if errors.Is(err, ErrPairingNotFound) || errors.Is(err, ErrPairingExpired) || errors.Is(err, ErrPairingNotClaimable) {
			// Deliberately vague — do not leak whether the code was unknown, used, or expired.
			return httpapi.ClaimPairing404JSONResponse(errorMsgResponse("Pairing code not found")), nil
		}
		logger.ErrorContext(ctx, "failed to claim pairing", slog.Any(logging.AttrKeyError, err))
		return httpapi.ClaimPairing500JSONResponse(errorMsgResponse("Failed to process pairing")), nil
	}

	return httpapi.ClaimPairing200JSONResponse(httpapi.ClaimPairingResponse{
		ServerUrl:           result.ServerURL,
		IntervalSeconds:     result.IntervalSeconds,
		AppBiometricEnabled: result.AppBiometricEnabled,
		AppSettingsLocked:   result.AppSettingsLocked,
		ApiKey:              result.RawAPIKey,
	}), nil
}

func (h *HTTPHandler) CreateDevicePairing(ctx context.Context, request httpapi.CreateDevicePairingRequestObject) (httpapi.CreateDevicePairingResponseObject, error) {
	ctx = logging.WithOperation(ctx, "CreateDevicePairing")
	logger := h.logger

	body := request.Body
	appBiometricEnabled := false
	if body.AppBiometricEnabled != nil {
		appBiometricEnabled = *body.AppBiometricEnabled
	}
	appSettingsLocked := false
	if body.AppSettingsLocked != nil {
		appSettingsLocked = *body.AppSettingsLocked
	}

	pairing, err := h.service.CreatePairing(ctx, CreatePairingRequest{
		DeviceID:            ids.DeviceID(request.Id),
		HeartbeatServerURL:  body.HeartbeatServerUrl,
		IntervalSeconds:     body.IntervalSeconds,
		AppBiometricEnabled: appBiometricEnabled,
		AppSettingsLocked:   appSettingsLocked,
		ExpiresInHours:      int(body.ExpiresInHours),
	})
	if err != nil {
		logger.ErrorContext(ctx, "failed to create device pairing", slog.Any(logging.AttrKeyError, err))
		return httpapi.CreateDevicePairing500JSONResponse(errorMsgResponse("Failed to create device pairing")), nil
	}

	return httpapi.CreateDevicePairing201JSONResponse(toAPIPairing(pairing)), nil
}

func (h *HTTPHandler) ListDevicePairings(ctx context.Context, request httpapi.ListDevicePairingsRequestObject) (httpapi.ListDevicePairingsResponseObject, error) {
	ctx = logging.WithOperation(ctx, "ListDevicePairings")
	logger := h.logger

	filter := PairingFilter{DeviceID: ids.DeviceID(request.Id)}
	if request.Params.Status != nil && *request.Params.Status == httpapi.ListDevicePairingsParamsStatusAll {
		filter.IncludeAll = true
	}

	pairings, err := h.service.ListPairings(ctx, filter)
	if err != nil {
		logger.ErrorContext(ctx, "failed to list device pairings", slog.Any(logging.AttrKeyError, err))
		return httpapi.ListDevicePairings500JSONResponse(errorMsgResponse("Failed to list device pairings")), nil
	}

	resp := make(httpapi.ListDevicePairings200JSONResponse, 0, len(pairings))
	for _, p := range pairings {
		resp = append(resp, toAPIPairing(&p))
	}
	return resp, nil
}

func (h *HTTPHandler) GetDevicePairing(ctx context.Context, request httpapi.GetDevicePairingRequestObject) (httpapi.GetDevicePairingResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetDevicePairing")
	logger := h.logger

	pairing, err := h.service.GetPairing(ctx, ids.DevicePairingID(request.PairingId))
	if err != nil {
		if errors.Is(err, ErrPairingNotFound) {
			return httpapi.GetDevicePairing404JSONResponse(errorMsgResponse("Device pairing not found")), nil
		}
		logger.ErrorContext(ctx, "failed to get device pairing", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetDevicePairing500JSONResponse(errorMsgResponse("Failed to get device pairing")), nil
	}

	return httpapi.GetDevicePairing200JSONResponse(toAPIPairing(pairing)), nil
}

func (h *HTTPHandler) DeleteDevicePairing(ctx context.Context, request httpapi.DeleteDevicePairingRequestObject) (httpapi.DeleteDevicePairingResponseObject, error) {
	ctx = logging.WithOperation(ctx, "DeleteDevicePairing")
	logger := h.logger

	err := h.service.InvalidatePairing(ctx, ids.DeviceID(request.Id), ids.DevicePairingID(request.PairingId))
	if err != nil {
		if errors.Is(err, ErrPairingNotFound) {
			return httpapi.DeleteDevicePairing404JSONResponse(errorMsgResponse("Device pairing not found")), nil
		}
		logger.ErrorContext(ctx, "failed to delete device pairing", slog.Any(logging.AttrKeyError, err))
		return httpapi.DeleteDevicePairing500JSONResponse(errorMsgResponse("Failed to delete device pairing")), nil
	}

	return httpapi.DeleteDevicePairing204Response{}, nil
}

// toAPIPairing converts a domain DevicePairing to the httpapi type.
func toAPIPairing(p *DevicePairing) httpapi.DevicePairing {
	return httpapi.DevicePairing{
		Id:                  p.ID.Int64(),
		DeviceId:            p.DeviceID.Int64(),
		PairingCode:         p.PairingCode,
		HeartbeatServerUrl:  p.HeartbeatServerURL,
		IntervalSeconds:     p.HeartbeatIntervalSeconds,
		AppBiometricEnabled: p.AppBiometricEnabled,
		AppSettingsLocked:   p.AppSettingsLocked,
		ExpiresAt:           httpapi.UTCTime(p.ExpiresAt),
		CreatedAt:           httpapi.UTCTime(p.CreatedAt),
		UpdatedAt:           httpapi.UTCTime(p.UpdatedAt),
		Status:              httpapi.DevicePairingStatus(p.Status),
	}
}

func errorMsgResponse(msg string) httpapi.ErrorResponse {
	return httpapi.ErrorResponse{Error: &msg}
}
