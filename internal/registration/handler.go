package registration

import (
	"context"
	"errors"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

// HTTPHandler implements the registration subset of httpapi.StrictServerInterface.
type HTTPHandler struct {
	service *Service
	logger  *slog.Logger
}

func NewHTTPHandler(svc *Service, logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		service: svc,
		logger:  logger.With(slog.String(logging.AttrKeyComponent, "registration")),
	}
}

func (h *HTTPHandler) ClaimRegistration(ctx context.Context, request httpapi.ClaimRegistrationRequestObject) (httpapi.ClaimRegistrationResponseObject, error) {
	ctx = logging.WithOperation(ctx, "ClaimRegistration")
	logger := h.logger

	result, err := h.service.ClaimInvite(ctx, request.Body.Code)
	if err != nil {
		if errors.Is(err, ErrInviteNotFound) {
			// Deliberately vague — do not leak whether the code was unknown, used, or expired.
			return httpapi.ClaimRegistration404JSONResponse(errorMsgResponse("Registration code not found")), nil
		}
		logger.ErrorContext(ctx, "failed to claim registration invite", slog.Any(logging.AttrKeyError, err))
		return httpapi.ClaimRegistration500JSONResponse(errorMsgResponse("Failed to process registration")), nil
	}

	return httpapi.ClaimRegistration200JSONResponse(httpapi.ClaimRegistrationResponse{
		ServerUrl:              result.ServerURL,
		IntervalSeconds:        result.IntervalSeconds,
		BiometricEnabled:       result.BiometricEnabled,
		BiometricUserCanToggle: result.BiometricUserCanToggle,
		ApiKey:                 result.RawAPIKey,
	}), nil
}

func (h *HTTPHandler) CreateRegistration(ctx context.Context, request httpapi.CreateRegistrationRequestObject) (httpapi.CreateRegistrationResponseObject, error) {
	ctx = logging.WithOperation(ctx, "CreateRegistration")
	logger := h.logger

	principal, ok := auth.PrincipalFromContext(ctx)
	if !ok {
		return httpapi.CreateRegistration401Response{}, nil
	}
	if !principal.IsAdmin() {
		return httpapi.CreateRegistration403Response{}, nil
	}

	body := request.Body
	biometricEnabled := false
	if body.BiometricEnabled != nil {
		biometricEnabled = *body.BiometricEnabled
	}
	biometricUserCanToggle := true
	if body.BiometricUserCanToggle != nil {
		biometricUserCanToggle = *body.BiometricUserCanToggle
	}

	invite, err := h.service.CreateInvite(ctx, CreateInviteRequest{
		DeviceName:             body.DeviceName,
		HeartbeatServerURL:     body.HeartbeatServerUrl,
		IntervalSeconds:        body.IntervalSeconds,
		BiometricEnabled:       biometricEnabled,
		BiometricUserCanToggle: biometricUserCanToggle,
		ExpiresInHours:         int(body.ExpiresInHours),
	})
	if err != nil {
		logger.ErrorContext(ctx, "failed to create registration invite", slog.Any(logging.AttrKeyError, err))
		return httpapi.CreateRegistration500JSONResponse(errorMsgResponse("Failed to create registration invite")), nil
	}

	return httpapi.CreateRegistration201JSONResponse(toAPIRegistration(invite)), nil
}

func (h *HTTPHandler) ListRegistrations(ctx context.Context, request httpapi.ListRegistrationsRequestObject) (httpapi.ListRegistrationsResponseObject, error) {
	ctx = logging.WithOperation(ctx, "ListRegistrations")
	logger := h.logger

	principal, ok := auth.PrincipalFromContext(ctx)
	if !ok {
		return httpapi.ListRegistrations401Response{}, nil
	}
	if !principal.IsAdmin() {
		return httpapi.ListRegistrations403Response{}, nil
	}

	filter := InviteFilter{}
	if request.Params.Status != nil && *request.Params.Status == httpapi.ListRegistrationsParamsStatusAll {
		filter.IncludeAll = true
	}

	invites, err := h.service.ListInvites(ctx, filter)
	if err != nil {
		logger.ErrorContext(ctx, "failed to list registration invites", slog.Any(logging.AttrKeyError, err))
		return httpapi.ListRegistrations500JSONResponse(errorMsgResponse("Failed to list registration invites")), nil
	}

	resp := make(httpapi.ListRegistrations200JSONResponse, 0, len(invites))
	for _, inv := range invites {
		resp = append(resp, toAPIRegistration(inv))
	}
	return resp, nil
}

func (h *HTTPHandler) GetRegistration(ctx context.Context, request httpapi.GetRegistrationRequestObject) (httpapi.GetRegistrationResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetRegistration")
	logger := h.logger

	principal, ok := auth.PrincipalFromContext(ctx)
	if !ok {
		return httpapi.GetRegistration401Response{}, nil
	}
	if !principal.IsAdmin() {
		return httpapi.GetRegistration403Response{}, nil
	}

	invite, err := h.service.GetInvite(ctx, request.RegistrationId)
	if err != nil {
		if errors.Is(err, ErrInviteNotFound) {
			return httpapi.GetRegistration404JSONResponse(errorMsgResponse("Registration invite not found")), nil
		}
		logger.ErrorContext(ctx, "failed to get registration invite", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetRegistration500JSONResponse(errorMsgResponse("Failed to get registration invite")), nil
	}

	return httpapi.GetRegistration200JSONResponse(toAPIRegistration(invite)), nil
}

func (h *HTTPHandler) DeleteRegistration(ctx context.Context, request httpapi.DeleteRegistrationRequestObject) (httpapi.DeleteRegistrationResponseObject, error) {
	ctx = logging.WithOperation(ctx, "DeleteRegistration")
	logger := h.logger

	principal, ok := auth.PrincipalFromContext(ctx)
	if !ok {
		return httpapi.DeleteRegistration401Response{}, nil
	}
	if !principal.IsAdmin() {
		return httpapi.DeleteRegistration403Response{}, nil
	}

	err := h.service.InvalidateInvite(ctx, request.RegistrationId)
	if err != nil {
		if errors.Is(err, ErrInviteNotFound) {
			return httpapi.DeleteRegistration404JSONResponse(errorMsgResponse("Registration invite not found")), nil
		}
		if errors.Is(err, ErrInviteNotPending) {
			return httpapi.DeleteRegistration404JSONResponse(errorMsgResponse("Registration invite not found")), nil
		}
		logger.ErrorContext(ctx, "failed to delete registration invite", slog.Any(logging.AttrKeyError, err))
		return httpapi.DeleteRegistration500JSONResponse(errorMsgResponse("Failed to delete registration invite")), nil
	}

	return httpapi.DeleteRegistration204Response{}, nil
}

// toAPIRegistration converts a domain PendingRegistration to the httpapi type.
func toAPIRegistration(p *PendingRegistration) httpapi.PendingRegistration {
	reg := httpapi.PendingRegistration{
		Id:                     p.ID,
		DeviceName:             p.DeviceName,
		RegistrationCode:       p.RegistrationCode,
		DeviceApiKeyPrefix:     p.DeviceAPIKeyPrefix,
		HeartbeatServerUrl:     p.HeartbeatServerURL,
		IntervalSeconds:        p.IntervalSeconds,
		BiometricEnabled:       p.BiometricEnabled,
		BiometricUserCanToggle: p.BiometricUserCanToggle,
		ExpiresAt:              httpapi.UTCTime(p.ExpiresAt),
		CreatedAt:              httpapi.UTCTime(p.CreatedAt),
		Status:                 httpapi.PendingRegistrationStatus(p.Status()),
	}
	if p.UsedAt != nil {
		t := httpapi.UTCTime(*p.UsedAt)
		reg.UsedAt = &t
	}
	if p.CreatedDeviceID != nil {
		reg.CreatedDeviceId = p.CreatedDeviceID
	}
	return reg
}

func errorMsgResponse(msg string) httpapi.ErrorResponse {
	return httpapi.ErrorResponse{Error: &msg}
}
