//go:build test

package queries

import (
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/matryer/is"
)

// fakeGeoResolver returns canned results keyed by IP; unknown IPs resolve empty.
type fakeGeoResolver map[string]geoip.Result

func (f fakeGeoResolver) Resolve(ip string) geoip.Result {
	return f[ip]
}

func TestGeoInfoFromResult_EmptyReturnsNil(t *testing.T) {
	is := is.New(t)
	is.Equal(geoInfoFromResult(geoip.Result{}), (*httpapi.GeoInfo)(nil))
}

func TestGeoInfoFromResult_PopulatesSetFieldsOnly(t *testing.T) {
	is := is.New(t)

	// Country resolved but ASN absent: asn fields stay nil.
	info := geoInfoFromResult(geoip.Result{CountryCode: "DE", CountryName: "Germany", ContinentCode: "EU"})
	is.True(info != nil)
	is.Equal(*info.CountryCode, "DE")
	is.Equal(*info.CountryName, "Germany")
	is.Equal(*info.ContinentCode, "EU")
	is.Equal(info.Asn, (*int64)(nil))
	is.Equal(info.AsnOrg, (*string)(nil))

	// ASN-only result still yields a non-nil GeoInfo with just the ASN fields.
	asnOnly := geoInfoFromResult(geoip.Result{ASN: 13335, ASNOrg: "Cloudflare, Inc."})
	is.True(asnOnly != nil)
	is.Equal(asnOnly.CountryCode, (*string)(nil))
	is.Equal(*asnOnly.Asn, int64(13335))
	is.Equal(*asnOnly.AsnOrg, "Cloudflare, Inc.")
}

func TestToAddressViewResponse_EnrichesGeo(t *testing.T) {
	is := is.New(t)

	geo := fakeGeoResolver{
		"1.1.1.1": {CountryCode: "AU", CountryName: "Australia", ContinentCode: "OC", ASN: 13335, ASNOrg: "Cloudflare, Inc."},
	}
	view := &AddressView{
		ID:        ids.AddressID(1),
		DeviceID:  ids.DeviceID(2),
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

func TestEnrichGeo_SetsPerIP(t *testing.T) {
	is := is.New(t)

	geo := fakeGeoResolver{
		"1.1.1.1": {CountryCode: "AU", CountryName: "Australia"},
		// 10.0.0.1 absent → resolves empty → geo omitted.
	}
	ips := []httpapi.PolicyUserIP{
		{Ip: "1.1.1.1"},
		{Ip: "10.0.0.1"},
	}

	enrichGeo(ips, geo)

	is.True(ips[0].Geo != nil)
	is.Equal(*ips[0].Geo.CountryCode, "AU")
	is.Equal(ips[1].Geo, (*httpapi.GeoInfo)(nil))
}

func TestEnrichGeo_NilResolverLeavesGeoUnset(t *testing.T) {
	is := is.New(t)

	ips := []httpapi.PolicyUserIP{{Ip: "1.1.1.1"}}
	enrichGeo(ips, nil)
	is.Equal(ips[0].Geo, (*httpapi.GeoInfo)(nil))
}
