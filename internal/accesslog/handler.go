package accesslog

import (
	"context"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

type denyReasonsLister interface {
	ListDenyReasons(ctx context.Context) ([]string, error)
}

type HTTPHandler struct {
	repo   denyReasonsLister
	logger *slog.Logger
}

func NewHTTPHandler(repo denyReasonsLister, logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		repo:   repo,
		logger: logger.With(slog.String(logging.AttrKeyComponent, "accesslog")),
	}
}

func (h *HTTPHandler) GetAccessLogDenyReasons(
	ctx context.Context,
	_ httpapi.GetAccessLogDenyReasonsRequestObject,
) (httpapi.GetAccessLogDenyReasonsResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetAccessLogDenyReasons")

	reasons, err := h.repo.ListDenyReasons(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to list deny reasons", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetAccessLogDenyReasons500JSONResponse(errorMsgResponse("Failed to list deny reasons")), nil
	}
	return httpapi.GetAccessLogDenyReasons200JSONResponse(reasons), nil
}

func errorMsgResponse(msg string) httpapi.ErrorResponse {
	return httpapi.ErrorResponse{Error: &msg}
}
