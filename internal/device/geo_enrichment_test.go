//go:build test

package device

import (
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/matryer/is"
)

var baseTime = time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

// fakeGeoResolver returns canned results keyed by IP; unknown IPs resolve empty.
type fakeGeoResolver map[string]geoip.Result

func (f fakeGeoResolver) Resolve(ip string) geoip.Result {
	return f[ip]
}

func TestToAddressViewResponse_EnrichesGeo(t *testing.T) {
	is := is.New(t)

	geo := fakeGeoResolver{
		"1.1.1.1": {CountryCode: "AU", CountryName: "Australia", ContinentCode: "OC", ASN: 13335, ASNOrg: "Cloudflare, Inc."},
	}
	view := &AddressView{
		ID:        1,
		DeviceID:  2,
		IP:        "1.1.1.1",
		IsEnabled: true,
		Source:    "heartbeat",
		CreatedAt: baseTime,
		UpdatedAt: baseTime,
	}

	addr := toAddressViewResponse(view, geo)
	is.True(addr.Geo != nil)
	is.Equal(*addr.Geo.CountryCode, "AU")
	is.Equal(*addr.Geo.AsnOrg, "Cloudflare, Inc.")
}

func TestToAddressViewResponse_PrivateIPOmitsGeo(t *testing.T) {
	is := is.New(t)

	// Resolver returns empty for an unknown/private IP — geo is omitted.
	geo := fakeGeoResolver{}
	view := &AddressView{IP: "10.0.0.1", Source: "manual", CreatedAt: baseTime, UpdatedAt: baseTime}

	addr := toAddressViewResponse(view, geo)
	is.Equal(addr.Geo, (*httpapi.GeoInfo)(nil))
}

func TestToAddressViewResponse_NilResolverOmitsGeo(t *testing.T) {
	is := is.New(t)

	view := &AddressView{IP: "1.1.1.1", Source: "manual", CreatedAt: baseTime, UpdatedAt: baseTime}
	addr := toAddressViewResponse(view, nil)
	is.Equal(addr.Geo, (*httpapi.GeoInfo)(nil))
}
