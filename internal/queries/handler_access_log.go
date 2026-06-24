package queries

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
	"github.com/DiegoGuidaF/PulseWeaver/internal/queries/filterx"
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
	var asn *int
	if r.ASN != nil {
		asn = new(int(*r.ASN))
	}

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
