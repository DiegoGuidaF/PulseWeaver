package device

import (
	"context"
	"errors"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

func (h *HTTPHandler) GetDeviceAddresses(
	ctx context.Context,
	request httpapi.GetDeviceAddressesRequestObject,
) (httpapi.GetDeviceAddressesResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetDeviceAddresses")
	deviceID := ids.DeviceID(request.DeviceId)
	logger := h.logger.With(slog.Int64(AttrKeyDeviceID, deviceID.Int64()))

	addresses, err := h.repo.GetDeviceAddresses(ctx, deviceID)
	if err != nil {
		switch {
		case errors.Is(err, ErrDeviceNotFound):
			return httpapi.GetDeviceAddresses404JSONResponse(errorMsgResponse("Device not found")), nil
		default:
			logger.ErrorContext(ctx, "failed to list device addresses", slog.Any(AttrKeyError, err))
			return httpapi.GetDeviceAddresses500JSONResponse(errorMsgResponse("Failed to list device IPs")), nil
		}
	}

	response := make([]httpapi.Address, len(addresses))
	for i := range addresses {
		response[i] = toAddressViewResponse(&addresses[i], h.geo)
	}
	return httpapi.GetDeviceAddresses200JSONResponse(response), nil
}

func toAddressViewResponse(a *AddressView, geo GeoResolver) httpapi.Address {
	address := httpapi.Address{
		Id:        a.ID.Int64(),
		DeviceId:  a.DeviceID.Int64(),
		Ip:        a.IP,
		IsEnabled: a.IsEnabled,
		Source:    httpapi.AddressEventSource(a.Source),
		CreatedAt: httpapi.UTCTime(a.CreatedAt),
		UpdatedAt: httpapi.UTCTime(a.UpdatedAt),
	}
	if a.ExpiresAt != nil {
		address.ExpiresAt = new(httpapi.UTCTime(*a.ExpiresAt))
	}
	if geo != nil {
		address.Geo = geoInfoFromResult(geo.Resolve(a.IP))
	}
	return address
}

// geoInfoFromResult maps a geoip.Result to the API GeoInfo DTO, returning nil
// when the lookup found nothing so the field is omitted from the response.
func geoInfoFromResult(r geoip.Result) *httpapi.GeoInfo {
	if r.IsEmpty() {
		return nil
	}
	info := &httpapi.GeoInfo{}
	if r.CountryCode != "" {
		info.CountryCode = &r.CountryCode
	}
	if r.CountryName != "" {
		info.CountryName = &r.CountryName
	}
	if r.ContinentCode != "" {
		info.ContinentCode = &r.ContinentCode
	}
	if r.ASN != 0 {
		asn := int64(r.ASN)
		info.Asn = &asn
	}
	if r.ASNOrg != "" {
		info.AsnOrg = &r.ASNOrg
	}
	return info
}
