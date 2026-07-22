//go:build test

package httpapi_test

import (
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/matryer/is"
)

func TestGeoInfoFromResult_EmptyReturnsNil(t *testing.T) {
	is := is.New(t)
	is.Equal(httpapi.GeoInfoFromResult(geoip.Result{}), (*httpapi.GeoInfo)(nil))
}

func TestGeoInfoFromResult_PopulatesSetFieldsOnly(t *testing.T) {
	is := is.New(t)

	// Country resolved but ASN absent: asn fields stay nil.
	info := httpapi.GeoInfoFromResult(geoip.Result{CountryCode: "DE", CountryName: "Germany", ContinentCode: "EU"})
	is.True(info != nil)
	is.Equal(*info.CountryCode, "DE")
	is.Equal(*info.CountryName, "Germany")
	is.Equal(*info.ContinentCode, "EU")
	is.Equal(info.Asn, (*int64)(nil))
	is.Equal(info.AsnOrg, (*string)(nil))

	// ASN-only result still yields a non-nil GeoInfo with just the ASN fields.
	asnOnly := httpapi.GeoInfoFromResult(geoip.Result{ASN: 13335, ASNOrg: "Cloudflare, Inc."})
	is.True(asnOnly != nil)
	is.Equal(asnOnly.CountryCode, (*string)(nil))
	is.Equal(*asnOnly.Asn, int64(13335))
	is.Equal(*asnOnly.AsnOrg, "Cloudflare, Inc.")
}
