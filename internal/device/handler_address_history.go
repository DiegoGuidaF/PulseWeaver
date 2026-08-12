package device

import (
	"context"
	"errors"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
	"github.com/DiegoGuidaF/PulseWeaver/internal/queries/filterx"
)

func (h *HTTPHandler) GetAddressHistory(
	ctx context.Context,
	request httpapi.GetAddressHistoryRequestObject,
) (httpapi.GetAddressHistoryResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetAddressHistory")
	logger := h.logger

	query, err := NewAddressHistoryQuery(request.Params)
	if err != nil {
		if errors.Is(err, filterx.ErrInvalidFilter) {
			logger.WarnContext(ctx, "invalid query parameters", slog.Any(AttrKeyError, err))
			return httpapi.GetAddressHistory400JSONResponse(errorMsgResponse(err.Error())), nil
		}
		logger.ErrorContext(ctx, "failed to validate address history query", slog.Any(AttrKeyError, err))
		return httpapi.GetAddressHistory500JSONResponse(errorMsgResponse("Failed to get address history")), nil
	}

	result, err := h.repo.GetAddressHistoryEvents(ctx, query)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get address history events", slog.Any(AttrKeyError, err))
		return httpapi.GetAddressHistory500JSONResponse(errorMsgResponse("Failed to get address history")), nil
	}

	return httpapi.GetAddressHistory200JSONResponse(toAddressHistoryResponse(result, query.Limit, h.geo)), nil
}

func (h *HTTPHandler) GetAddressHistoryHistogram(
	ctx context.Context,
	request httpapi.GetAddressHistoryHistogramRequestObject,
) (httpapi.GetAddressHistoryHistogramResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetAddressHistoryHistogram")
	logger := h.logger

	query, err := NewAddressHistoryHistogramQuery(request.Params)
	if err != nil {
		if errors.Is(err, filterx.ErrInvalidFilter) {
			logger.WarnContext(ctx, "invalid query parameters", slog.Any(AttrKeyError, err))
			return httpapi.GetAddressHistoryHistogram400JSONResponse(errorMsgResponse(err.Error())), nil
		}
		logger.ErrorContext(ctx, "failed to validate address history histogram query", slog.Any(AttrKeyError, err))
		return httpapi.GetAddressHistoryHistogram500JSONResponse(errorMsgResponse("Failed to get address history histogram")), nil
	}

	response, err := h.repo.GetAddressHistoryHistogram(ctx, query)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get address history histogram", slog.Any(AttrKeyError, err))
		return httpapi.GetAddressHistoryHistogram500JSONResponse(errorMsgResponse("Failed to get address history histogram")), nil
	}

	return httpapi.GetAddressHistoryHistogram200JSONResponse(response), nil
}

func (h *HTTPHandler) GetAddressHistoryTuning(
	ctx context.Context,
	request httpapi.GetAddressHistoryTuningRequestObject,
) (httpapi.GetAddressHistoryTuningResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetAddressHistoryTuning")
	logger := h.logger

	query := NewAddressHistoryTuningQuery(request.Params)

	response, err := h.repo.GetAddressHistoryTuning(ctx, query)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get address history tuning", slog.Any(AttrKeyError, err))
		return httpapi.GetAddressHistoryTuning500JSONResponse(errorMsgResponse("Failed to get address history tuning")), nil
	}

	return httpapi.GetAddressHistoryTuning200JSONResponse(response), nil
}

func toAddressHistoryResponse(result AddressHistoryEvents, limit int, geo GeoResolver) httpapi.AddressHistoryResponse {
	events := make([]httpapi.AddressHistoryEvent, len(result.Events))
	for i, e := range result.Events {
		events[i] = httpapi.AddressHistoryEvent{
			Id:                e.ID,
			Timestamp:         httpapi.UTCTime(e.CreatedAt),
			Ip:                e.IP,
			IsEnabled:         e.IsEnabled,
			Source:            e.Source,
			DeviceId:          e.DeviceID.Int64(),
			DeviceName:        e.DeviceName,
			RenewalGapSeconds: e.RenewalGapSeconds,
			EventKind:         AddressEventKindToAPI(e.EventKind),
			TtlRisk:           TTLRiskToAPI(e.TTLRisk),
			TtlSeconds:        e.TTLSeconds,
		}
		if geo != nil {
			events[i].Geo = httpapi.GeoInfoFromResult(geo.Resolve(e.IP))
		}
	}

	// Use len == limit as "has more" signal — reliable across all pages,
	// unlike comparing against Total which ignores the cursor offset.
	var nextCursor *int64
	if len(result.Events) == limit && limit > 0 {
		nextCursor = &result.Events[len(result.Events)-1].ID
	}

	return httpapi.AddressHistoryResponse{
		Events:     events,
		Total:      result.Total,
		NextCursor: nextCursor,
	}
}
