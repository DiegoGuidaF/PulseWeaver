package queries

import (
	"context"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

// defaultAnomalyLimit matches the OpenAPI default; oapi-codegen leaves the param
// nil when omitted, so the handler applies it.
const defaultAnomalyLimit = 100

func (h *HTTPHandler) ListAnomalies(
	ctx context.Context,
	request httpapi.ListAnomaliesRequestObject,
) (httpapi.ListAnomaliesResponseObject, error) {
	ctx = logging.WithOperation(ctx, "ListAnomalies")

	q := AnomalyListQuery{Limit: defaultAnomalyLimit}
	if request.Params.Limit != nil {
		q.Limit = *request.Params.Limit
	}
	if request.Params.Status != nil {
		status := string(*request.Params.Status)
		q.Status = &status
	}
	if request.Params.Kind != nil {
		kinds := make([]string, len(*request.Params.Kind))
		for i, k := range *request.Params.Kind {
			kinds[i] = string(k)
		}
		q.Kinds = kinds
	}

	rows, err := h.repo.ListAnomalies(ctx, q)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to list anomalies", slog.Any(logging.AttrKeyError, err))
		return httpapi.ListAnomalies500JSONResponse(errorMsgResponse("Failed to list anomalies")), nil
	}

	return httpapi.ListAnomalies200JSONResponse(httpapi.AnomalyListResponse{Anomalies: rows}), nil
}
