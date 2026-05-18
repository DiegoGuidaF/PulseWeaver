package queries

import (
	"context"
	"log/slog"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

func (h *HTTPHandler) GetAccessLog(
	ctx context.Context,
	request httpapi.GetAccessLogRequestObject,
) (httpapi.GetAccessLogResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetAccessLog")

	query := NewAccessLogQuery(request.Params)

	rows, total, err := h.repo.ListAccessLog(ctx, query)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to list access log", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetAccessLog500JSONResponse(errorMsgResponse("Failed to list access log")), nil
	}

	httpRows := make([]httpapi.AccessLogRow, len(rows))
	for i := range rows {
		httpRows[i] = toAccessLogRow(rows[i])
	}

	var nextCursor *int64
	if len(rows) == query.Limit {
		nextCursor = &rows[len(rows)-1].ID
	}

	return httpapi.GetAccessLog200JSONResponse(httpapi.AccessLogResponse{
		Total:      total,
		NextCursor: nextCursor,
		Rows:       httpRows,
	}), nil
}

func (h *HTTPHandler) GetAccessLogByCountry(
	ctx context.Context,
	request httpapi.GetAccessLogByCountryRequestObject,
) (httpapi.GetAccessLogByCountryResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetAccessLogByCountry")

	now := time.Now().UTC()
	from := now.Add(-24 * time.Hour)
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
			CountryCode:   s.CountryCode,
			CountryName:   &s.CountryName,
			ContinentCode: &s.ContinentCode,
			Total:         int(s.Total),
			Allowed:       int(s.Allowed),
			Denied:        int(s.Denied),
		}
	}

	return httpapi.GetAccessLogByCountry200JSONResponse(result), nil
}

func toAccessLogRow(r AccessLogView) httpapi.AccessLogRow {
	var deviceID *int64
	if r.DeviceID != nil {
		deviceID = new(r.DeviceID.Int64())
	}
	var addressID *int64
	if r.AddressID != nil {
		addressID = new(r.AddressID.Int64())
	}
	var asn *int
	if r.ASN != nil {
		asn = new(int(*r.ASN))
	}

	return httpapi.AccessLogRow{
		Id:                r.ID,
		CreatedAt:         httpapi.UTCTime(r.CreatedAt),
		Outcome:           r.Outcome,
		ClientIp:          r.ClientIP,
		DenyReason:        r.DenyReason,
		DeviceId:          deviceID,
		DeviceName:        r.DeviceName,
		AddressId:         addressID,
		XffChain:          r.XFFChain,
		TargetHost:        r.TargetHost,
		TargetUri:         r.TargetURI,
		HttpMethod:        r.HTTPMethod,
		Headers:           r.Headers,
		CountryCode:       r.CountryCode,
		CountryName:       r.CountryName,
		ContinentCode:     r.ContinentCode,
		Asn:               asn,
		AsnOrg:            r.ASNOrg,
		DurationUs:        &r.DurationUs,
		NetworkPolicyId:   r.NetworkPolicyID,
		NetworkPolicyName: r.NetworkPolicyName,
	}
}
