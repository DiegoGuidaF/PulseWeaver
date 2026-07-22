//go:build test

package queries

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

func TestToAddressHistoryResponse_EnrichesEventGeo(t *testing.T) {
	is := is.New(t)

	geo := fakeGeoResolver{
		"1.1.1.1": {CountryCode: "AU", CountryName: "Australia", ContinentCode: "OC", ASN: 13335, ASNOrg: "Cloudflare, Inc."},
	}
	history := AddressHistory{
		Events: []AddressEventView{
			{ID: 1, IP: "1.1.1.1", Source: "heartbeat", CreatedAt: baseTime},
			{ID: 2, IP: "10.0.0.1", Source: "manual", CreatedAt: baseTime},
		},
	}

	resp := toAddressHistoryResponse(history, geo)
	is.True(resp.Events[0].Geo != nil)
	is.Equal(*resp.Events[0].Geo.CountryCode, "AU")
	// Private/unresolvable IP leaves geo omitted.
	is.Equal(resp.Events[1].Geo, (*httpapi.GeoInfo)(nil))
}

func TestToAddressHistoryResponse_NilResolverOmitsGeo(t *testing.T) {
	is := is.New(t)

	history := AddressHistory{
		Events: []AddressEventView{{ID: 1, IP: "1.1.1.1", Source: "manual", CreatedAt: baseTime}},
	}
	resp := toAddressHistoryResponse(history, nil)
	is.Equal(resp.Events[0].Geo, (*httpapi.GeoInfo)(nil))
}
