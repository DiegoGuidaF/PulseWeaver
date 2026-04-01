package queries

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

type HTTPHandler struct {
	repo   *Repository
	logger *slog.Logger
}

func NewHTTPHandler(repo *Repository, logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		repo:   repo,
		logger: logger.With(slog.String(logging.AttrKeyComponent, "queries")),
	}
}

func (h *HTTPHandler) GetDeviceAddresses(
	ctx context.Context,
	request httpapi.GetDeviceAddressesRequestObject,
) (httpapi.GetDeviceAddressesResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetDeviceAddresses")
	deviceID := device.DeviceID(request.DeviceId)
	logger := h.logger.With(slog.Int64(device.AttrKeyDeviceID, deviceID.Int64()))

	exists, err := h.repo.DeviceExists(ctx, deviceID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to check device existence", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetDeviceAddresses500JSONResponse(errorMsgResponse("Failed to list device IPs")), nil
	}
	if !exists {
		logger.WarnContext(ctx, "device not found")
		return httpapi.GetDeviceAddresses404JSONResponse(
			errorMsgResponse(fmt.Sprintf("Device with id %d not found", deviceID)),
		), nil
	}

	addresses, err := h.repo.GetDeviceAddresses(ctx, deviceID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to list device addresses", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetDeviceAddresses500JSONResponse(errorMsgResponse("Failed to list device IPs")), nil
	}

	response := make([]httpapi.Address, len(addresses))
	for i := range addresses {
		response[i] = toAddressViewResponse(&addresses[i])
	}
	return httpapi.GetDeviceAddresses200JSONResponse(response), nil
}

func (h *HTTPHandler) GetDevices(
	ctx context.Context,
	_ httpapi.GetDevicesRequestObject,
) (httpapi.GetDevicesResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetDevices")

	devices, err := h.repo.GetDevices(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to list devices", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetDevices500JSONResponse(errorMsgResponse("Failed to list devices")), nil
	}

	response := make([]httpapi.Device, len(devices))
	for i := range devices {
		response[i] = toDeviceViewResponse(&devices[i])
	}
	return httpapi.GetDevices200JSONResponse(response), nil
}

func (h *HTTPHandler) GetAccessLog(
	ctx context.Context,
	request httpapi.GetAccessLogRequestObject,
) (httpapi.GetAccessLogResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetAccessLog")
	params := request.Params

	query := NewAccessLogQuery(params)

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

	response := httpapi.AccessLogResponse{
		Total:      total,
		NextCursor: nextCursor,
		Rows:       httpRows,
	}

	return httpapi.GetAccessLog200JSONResponse(response), nil
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
		Id:            r.ID,
		CreatedAt:     httpapi.UTCTime(r.CreatedAt),
		Outcome:       r.Outcome,
		ClientIp:      r.ClientIP,
		DenyReason:    r.DenyReason,
		DeviceId:      deviceID,
		DeviceName:    r.DeviceName,
		AddressId:     addressID,
		XffChain:      r.XFFChain,
		TargetHost:    r.TargetHost,
		TargetUri:     r.TargetURI,
		HttpMethod:    r.HTTPMethod,
		Headers:       r.Headers,
		CountryCode:   r.CountryCode,
		CountryName:   r.CountryName,
		ContinentCode: r.ContinentCode,
		Asn:           asn,
		AsnOrg:        r.ASNOrg,
		DurationUs:    &r.DurationUs,
	}
}

func toAddressViewResponse(a *AddressView) httpapi.Address {
	address := httpapi.Address{
		Id:        a.ID.Int64(),
		DeviceId:  a.DeviceID.Int64(),
		Ip:        a.IP,
		IsEnabled: a.IsEnabled,
		CreatedAt: httpapi.UTCTime(a.CreatedAt),
		UpdatedAt: httpapi.UTCTime(a.UpdatedAt),
	}
	if a.ExpiresAt != nil {
		address.ExpiresAt = new(httpapi.UTCTime(*a.ExpiresAt))
	}

	return address
}

func toDeviceViewResponse(d *DeviceView) httpapi.Device {
	var lastSeenAt *httpapi.UTCTime
	if d.LastSeenAt != nil {
		lastSeenAt = new(httpapi.UTCTime(d.LastSeenAt.Time))
	}
	return httpapi.Device{
		Id:           d.ID.Int64(),
		Name:         d.Name,
		CreatedAt:    httpapi.UTCTime(d.CreatedAt),
		ApiKeyPrefix: d.KeyPrefix,
		AddressCount: new(d.AddressCount),
		LastSeenAt:   lastSeenAt,
	}
}

func errorMsgResponse(msg string) httpapi.ErrorResponse {
	return httpapi.ErrorResponse{Error: &msg}
}
