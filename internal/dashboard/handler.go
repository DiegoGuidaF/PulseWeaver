package dashboard

import (
	"context"
	"log/slog"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

type readRepo interface {
	GetSummaryStats(ctx context.Context, from, to time.Time) (SummaryStats, error)
	GetTrafficSeries(ctx context.Context, from, to time.Time) ([]TrafficBucket, error)
	GetTopDeniedIPs(ctx context.Context, from, to time.Time, limit int) ([]IPCount, error)
	GetServiceSplit(ctx context.Context, from, to time.Time) ([]ServiceCount, error)
	GetAttributionSplit(ctx context.Context, kind AttributionKind, from, to time.Time) ([]AttributionCount, error)
}

type HTTPHandler struct {
	repo   readRepo
	logger *slog.Logger
}

func NewHTTPHandler(repo readRepo, logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		repo:   repo,
		logger: logger.With(slog.String(logging.AttrKeyComponent, "dashboard")),
	}
}

func (h *HTTPHandler) GetDashboardStats(
	ctx context.Context,
	request httpapi.GetDashboardStatsRequestObject,
) (httpapi.GetDashboardStatsResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetDashboardStats")

	from, to := parseTimeRange(request.Params.From, request.Params.To)

	stats, err := h.repo.GetSummaryStats(ctx, from, to)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to get dashboard stats", slog.Any(AttrKeyError, err))
		return httpapi.GetDashboardStats500JSONResponse(errorMsgResponse("Failed to get dashboard stats")), nil
	}

	return httpapi.GetDashboardStats200JSONResponse{
		TotalRequests: stats.TotalRequests,
		AllowCount:    stats.AllowedCount,
		DenyCount:     stats.DeniedCount,
		UniqueIps:     stats.UniqueIPs,
		AvgDurationUs: stats.AvgDurationUs,
	}, nil
}

func (h *HTTPHandler) GetDashboardTraffic(
	ctx context.Context,
	request httpapi.GetDashboardTrafficRequestObject,
) (httpapi.GetDashboardTrafficResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetDashboardTraffic")

	from, to := parseTimeRange(request.Params.From, request.Params.To)

	buckets, err := h.repo.GetTrafficSeries(ctx, from, to)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to get traffic series", slog.Any(AttrKeyError, err))
		return httpapi.GetDashboardTraffic500JSONResponse(errorMsgResponse("Failed to get traffic data")), nil
	}

	httpBuckets := make([]httpapi.DashboardTrafficBucket, len(buckets))
	for i := range buckets {
		httpBuckets[i] = httpapi.DashboardTrafficBucket{
			Timestamp:  httpapi.UTCTime(buckets[i].Timestamp.Time),
			AllowCount: buckets[i].AllowCount,
			DenyCount:  buckets[i].DenyCount,
		}
	}

	return httpapi.GetDashboardTraffic200JSONResponse{
		Buckets: httpBuckets,
	}, nil
}

func (h *HTTPHandler) GetDashboardServices(
	ctx context.Context,
	request httpapi.GetDashboardServicesRequestObject,
) (httpapi.GetDashboardServicesResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetDashboardServices")

	from, to := parseTimeRange(request.Params.From, request.Params.To)

	services, err := h.repo.GetServiceSplit(ctx, from, to)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to get service split", slog.Any(AttrKeyError, err))
		return httpapi.GetDashboardServices500JSONResponse(errorMsgResponse("Failed to get service data")), nil
	}

	httpServices := make([]httpapi.DashboardServiceCount, len(services))
	for i := range services {
		httpServices[i] = httpapi.DashboardServiceCount{
			Host:       services[i].Host,
			AllowCount: services[i].AllowCount,
			DenyCount:  services[i].DenyCount,
		}
	}

	return httpapi.GetDashboardServices200JSONResponse{
		Services: httpServices,
	}, nil
}

func (h *HTTPHandler) GetDashboardTopDeniedIps(
	ctx context.Context,
	request httpapi.GetDashboardTopDeniedIpsRequestObject,
) (httpapi.GetDashboardTopDeniedIpsResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetDashboardTopDeniedIps")

	from, to := parseTimeRange(request.Params.From, request.Params.To)

	limit := 10
	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	ips, err := h.repo.GetTopDeniedIPs(ctx, from, to, limit)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to get top denied IPs", slog.Any(AttrKeyError, err))
		return httpapi.GetDashboardTopDeniedIps500JSONResponse(errorMsgResponse("Failed to get top denied IPs")), nil
	}

	httpIPs := make([]httpapi.DashboardTopDeniedIp, len(ips))
	for i := range ips {
		httpIPs[i] = httpapi.DashboardTopDeniedIp{
			Ip:    ips[i].IP,
			Count: ips[i].Count,
		}
	}

	return httpapi.GetDashboardTopDeniedIps200JSONResponse{
		Ips: httpIPs,
	}, nil
}

func (h *HTTPHandler) GetDashboardAttributionSplit(
	ctx context.Context,
	request httpapi.GetDashboardAttributionSplitRequestObject,
) (httpapi.GetDashboardAttributionSplitResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetDashboardAttributionSplit")

	from, to := parseTimeRange(request.Params.From, request.Params.To)
	kind := AttributionKind(request.Params.Kind)

	entities, err := h.repo.GetAttributionSplit(ctx, kind, from, to)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to get attribution split", slog.Any(AttrKeyError, err))
		return httpapi.GetDashboardAttributionSplit500JSONResponse(errorMsgResponse("Failed to get attribution data")), nil
	}

	httpEntities := make([]httpapi.DashboardAttributionCount, len(entities))
	for i := range entities {
		httpEntities[i] = httpapi.DashboardAttributionCount{
			EntityId:   entities[i].EntityID,
			EntityName: entities[i].EntityName,
			AllowCount: entities[i].AllowCount,
			DenyCount:  entities[i].DenyCount,
		}
	}

	return httpapi.GetDashboardAttributionSplit200JSONResponse{
		Entities: httpEntities,
	}, nil
}

// parseTimeRange extracts from/to with defaults: from = 24h ago, to = now.
func parseTimeRange(from, to *time.Time) (time.Time, time.Time) {
	now := time.Now().UTC()
	f := now.Add(-24 * time.Hour)
	t := now
	if from != nil {
		f = from.UTC()
	}
	if to != nil {
		t = to.UTC()
	}
	return f, t
}

func errorMsgResponse(msg string) httpapi.ErrorResponse {
	return httpapi.ErrorResponse{Error: &msg}
}
