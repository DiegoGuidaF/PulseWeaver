package queries

import (
	"context"
	"errors"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
	"github.com/DiegoGuidaF/PulseWeaver/internal/timebucket"
)

func (h *HTTPHandler) GetAddressHistory(
	ctx context.Context,
	request httpapi.GetAddressHistoryRequestObject,
) (httpapi.GetAddressHistoryResponseObject, error) {
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

	if err := query.Validate(); err != nil {
		if errors.Is(err, timebucket.ErrInvalidGranularity) {
			logger.WarnContext(ctx, "invalid query parameters", slog.Any(logging.AttrKeyError, err))
			return httpapi.GetAddressHistory400JSONResponse(errorMsgResponse(err.Error())), nil
		}
		logger.ErrorContext(ctx, "failed to validate address history query", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetAddressHistory500JSONResponse(errorMsgResponse("Failed to get address history")), nil
	}

	history, err := h.repo.GetAddressHistory(ctx, query)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get address history", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetAddressHistory500JSONResponse(errorMsgResponse("Failed to get address history")), nil
	}
	history.QueryLimit = query.Limit

	return httpapi.GetAddressHistory200JSONResponse(toAddressHistoryResponse(history)), nil
}

func toAddressHistoryResponse(h AddressHistory) httpapi.AddressHistoryResponse {
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
			Id:             e.ID,
			Timestamp:      httpapi.UTCTime(e.CreatedAt),
			Ip:             e.IP,
			IsEnabled:      e.IsEnabled,
			Source:         e.Source,
			DeviceId:       e.DeviceID.Int64(),
			DeviceName:     e.DeviceName,
			TimeGapSeconds: e.TimeGapSeconds,
			IpChanged:      e.IPChanged,
			IsRefresh:      e.IsRefresh,
			TtlSeconds:     e.TTLSeconds,
		}
	}

	// Use len == limit as "has more" signal — reliable across all pages,
	// unlike comparing against TotalEvents which ignores the cursor offset.
	var nextCursor *int64
	if len(h.Events) == h.QueryLimit {
		nextCursor = &h.Events[len(h.Events)-1].ID
	}

	return httpapi.AddressHistoryResponse{
		Buckets:     buckets,
		Events:      events,
		TotalEvents: h.TotalEvents,
		NextCursor:  nextCursor,
	}
}
