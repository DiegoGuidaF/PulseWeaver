package anomaly

import (
	"context"
	"errors"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

// HTTPHandler serves the anomaly mutation endpoints. Listing is a cross-domain
// read model in internal/queries; this handler owns only the single-domain
// acknowledge mutation, so it is registered even when the background scan is
// disabled — historical anomalies remain reviewable.
type HTTPHandler struct {
	repo   *Repository
	logger *slog.Logger
}

func NewHTTPHandler(repo *Repository, logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		repo:   repo,
		logger: logger.With(slog.String(logging.AttrKeyComponent, "anomaly")),
	}
}

func (h *HTTPHandler) AcknowledgeAnomaly(
	ctx context.Context,
	request httpapi.AcknowledgeAnomalyRequestObject,
) (httpapi.AcknowledgeAnomalyResponseObject, error) {
	ctx = logging.WithOperation(ctx, "AcknowledgeAnomaly")

	if err := h.repo.Acknowledge(ctx, request.Id); err != nil {
		if errors.Is(err, ErrNotFound) {
			h.logger.WarnContext(ctx, "anomaly not found", slog.Int64("anomaly_id", request.Id))
			return httpapi.AcknowledgeAnomaly404JSONResponse(errorMsgResponse("Anomaly not found")), nil
		}
		h.logger.ErrorContext(ctx, "failed to acknowledge anomaly", slog.Any(logging.AttrKeyError, err))
		return httpapi.AcknowledgeAnomaly500JSONResponse(errorMsgResponse("Failed to acknowledge anomaly")), nil
	}

	return httpapi.AcknowledgeAnomaly204Response{}, nil
}

func errorMsgResponse(msg string) httpapi.ErrorResponse {
	return httpapi.ErrorResponse{Error: &msg}
}
