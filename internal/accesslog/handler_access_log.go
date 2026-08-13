package accesslog

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/filterx"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

func (h *HTTPHandler) GetAccessLog(
	ctx context.Context,
	request httpapi.GetAccessLogRequestObject,
) (httpapi.GetAccessLogResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetAccessLog")

	query, err := NewAccessLogQuery(request.Params)
	if err != nil {
		if errors.Is(err, filterx.ErrInvalidFilter) {
			return httpapi.GetAccessLog400JSONResponse(errorMsgResponse(err.Error())), nil
		}
		h.logger.ErrorContext(ctx, "failed to build access log query", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetAccessLog500JSONResponse(errorMsgResponse("Failed to list access log")), nil
	}

	rows, total, err := h.repo.ListAccessLog(ctx, query)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to list access log", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetAccessLog500JSONResponse(errorMsgResponse("Failed to list access log")), nil
	}

	httpRows := make([]httpapi.AccessLogRow, len(rows))
	for i := range rows {
		httpRows[i] = toAccessLogRow(rows[i])
	}

	var nextCursor *string
	if len(rows) == query.Limit {
		last := rows[len(rows)-1]
		token, err := accessLogRegistry.EncodeCursor(query.Sort, query.Order, accessLogSortValue(last, query.Sort), last.ID)
		if err != nil {
			h.logger.ErrorContext(ctx, "failed to encode access log cursor", slog.Any(logging.AttrKeyError, err))
			return httpapi.GetAccessLog500JSONResponse(errorMsgResponse("Failed to list access log")), nil
		}
		nextCursor = &token
	}

	return httpapi.GetAccessLog200JSONResponse(httpapi.AccessLogResponse{
		Total:      total,
		NextCursor: nextCursor,
		Rows:       httpRows,
	}), nil
}

func (h *HTTPHandler) GetAccessLogEntry(
	ctx context.Context,
	request httpapi.GetAccessLogEntryRequestObject,
) (httpapi.GetAccessLogEntryResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetAccessLogEntry")

	entry, err := h.repo.GetAccessLogEntry(ctx, request.Id)
	if err != nil {
		if errors.Is(err, ErrEntryNotFound) {
			return httpapi.GetAccessLogEntry404JSONResponse(errorMsgResponse("Access log entry not found")), nil
		}
		h.logger.ErrorContext(ctx, "failed to get access log entry", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetAccessLogEntry500JSONResponse(errorMsgResponse("Failed to get access log entry")), nil
	}

	return httpapi.GetAccessLogEntry200JSONResponse(toAccessLogDetail(entry)), nil
}

func (h *HTTPHandler) GetAccessLogHistogram(
	ctx context.Context,
	request httpapi.GetAccessLogHistogramRequestObject,
) (httpapi.GetAccessLogHistogramResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetAccessLogHistogram")

	query, err := NewAccessLogHistogramQuery(request.Params)
	if err != nil {
		if errors.Is(err, filterx.ErrInvalidFilter) {
			return httpapi.GetAccessLogHistogram400JSONResponse(errorMsgResponse(err.Error())), nil
		}
		h.logger.ErrorContext(ctx, "failed to build access log histogram query", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetAccessLogHistogram500JSONResponse(errorMsgResponse("Failed to get access log histogram")), nil
	}

	response, err := h.repo.GetAccessLogHistogram(ctx, query)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to get access log histogram", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetAccessLogHistogram500JSONResponse(errorMsgResponse("Failed to get access log histogram")), nil
	}

	return httpapi.GetAccessLogHistogram200JSONResponse(response), nil
}

func (h *HTTPHandler) GetAccessLogByCountry(
	ctx context.Context,
	request httpapi.GetAccessLogByCountryRequestObject,
) (httpapi.GetAccessLogByCountryResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetAccessLogByCountry")

	now := time.Now().UTC()
	from := now.Add(-defaultWindow)
	to := now
	if request.Params.From != nil {
		from = request.Params.From.UTC()
	}
	if request.Params.To != nil {
		to = request.Params.To.UTC()
	}

	stats, err := h.repo.ListAccessLogStatsByCountry(ctx, from, to)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to list access log stats by country", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetAccessLogByCountry500JSONResponse(errorMsgResponse("Failed to list country stats")), nil
	}

	result := make([]httpapi.AccessLogCountryStats, len(stats))
	for i, s := range stats {
		result[i] = httpapi.AccessLogCountryStats{
			CountryCode: s.CountryCode,
			CountryName: &s.CountryName,
			Total:       int(s.Total),
			Allowed:     int(s.Allowed),
			Denied:      int(s.Denied),
		}
	}

	return httpapi.GetAccessLogByCountry200JSONResponse(result), nil
}

func toAccessLogRow(r AccessLogView) httpapi.AccessLogRow {
	contributors := make([]httpapi.AccessLogContributor, len(r.Contributors))
	for i, c := range r.Contributors {
		contributors[i] = toAccessLogContributor(c)
	}

	var denyReason *httpapi.PolicyDenyReason
	if r.DenyReason != nil {
		denyReason = new(httpapi.PolicyDenyReason(*r.DenyReason))
	}

	return httpapi.AccessLogRow{
		Id:                r.ID,
		CreatedAt:         httpapi.UTCTime(r.CreatedAt),
		Outcome:           r.Outcome,
		ClientIp:          r.ClientIP,
		DenyReason:        denyReason,
		Contributors:      contributors,
		ContributorCount:  r.ContributorCount,
		TargetHost:        r.TargetHost,
		TargetUri:         r.TargetURI,
		HttpMethod:        r.HTTPMethod,
		CountryCode:       r.CountryCode,
		DurationUs:        &r.DurationUs,
		NetworkPolicyId:   r.NetworkPolicyID,
		NetworkPolicyName: r.NetworkPolicyName,
	}
}

func toAccessLogDetail(d AccessLogDetailView) httpapi.AccessLogDetail {
	row := toAccessLogRow(d.AccessLogView)

	var asn *int
	if d.ASN != nil {
		asn = new(int(*d.ASN))
	}

	return httpapi.AccessLogDetail{
		Id:                row.Id,
		CreatedAt:         row.CreatedAt,
		Outcome:           row.Outcome,
		ClientIp:          row.ClientIp,
		DenyReason:        row.DenyReason,
		Contributors:      row.Contributors,
		ContributorCount:  row.ContributorCount,
		TargetHost:        row.TargetHost,
		TargetUri:         row.TargetUri,
		HttpMethod:        row.HttpMethod,
		CountryCode:       row.CountryCode,
		DurationUs:        row.DurationUs,
		NetworkPolicyId:   row.NetworkPolicyId,
		NetworkPolicyName: row.NetworkPolicyName,
		XffChain:          d.XFFChain,
		Headers:           d.Headers,
		CountryName:       d.CountryName,
		ContinentCode:     d.ContinentCode,
		Asn:               asn,
		AsnOrg:            d.ASNOrg,
	}
}

func toAccessLogContributor(c AccessLogContributor) httpapi.AccessLogContributor {
	var deviceID *httpapi.ID
	if c.DeviceID != nil {
		deviceID = new(c.DeviceID.Int64())
	}
	var userID *httpapi.ID
	if c.UserID != nil {
		userID = new(c.UserID.Int64())
	}
	var addressID *httpapi.ID
	if c.AddressID != nil {
		addressID = new(c.AddressID.Int64())
	}

	return httpapi.AccessLogContributor{
		DeviceId:   deviceID,
		DeviceName: c.DeviceName,
		UserId:     userID,
		UserName:   c.UserName,
		AddressId:  addressID,
	}
}
