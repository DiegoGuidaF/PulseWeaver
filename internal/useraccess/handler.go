package useraccess

import (
	"context"
	"errors"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

type HTTPHandler struct {
	service *Service
	logger  *slog.Logger
}

func NewHTTPHandler(service *Service, logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		service: service,
		logger:  logger.With(slog.String(logging.AttrKeyComponent, "useraccess")),
	}
}

func (h *HTTPHandler) SetUserAccess(
	ctx context.Context,
	req httpapi.SetUserAccessRequestObject,
) (httpapi.SetUserAccessResponseObject, error) {
	ctx = logging.WithOperation(ctx, "SetUserHostGrants")
	userID := ids.UserID(req.UserId)

	groupIDs := make([]ids.HostGroupID, len(req.Body.GroupIds))
	for i, id := range req.Body.GroupIds {
		groupIDs[i] = ids.HostGroupID(id)
	}

	if err := h.service.SetUserAccess(ctx, userID, req.Body.BypassHostCheck, groupIDs); err != nil {
		switch {
		case errors.Is(err, ErrReferenceNotFound), errors.Is(err, auth.ErrUserNotFound):
			return httpapi.SetUserAccess404JSONResponse(errResp("User or one of the referenced hosts/groups not found")), nil
		default:
			h.logger.ErrorContext(ctx, "set user host grants failed", slog.Any(logging.AttrKeyError, err))
			return httpapi.SetUserAccess500JSONResponse(errResp("Failed to set user grants")), nil
		}
	}

	return httpapi.SetUserAccess204Response{}, nil
}

func errResp(msg string) httpapi.ErrorResponse {
	return httpapi.ErrorResponse{Error: &msg}
}
