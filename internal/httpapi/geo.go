package httpapi

import "github.com/DiegoGuidaF/PulseWeaver/internal/geoip"

// GeoInfoFromResult maps a geo/ASN lookup onto its wire representation, returning
// nil for an unresolved address so the field is omitted rather than sent as an
// object full of empty strings. Every zero field stays nil for the same reason.
//
// It lives here rather than in geoip because it produces a generated API type:
// geoip is infrastructure and must not depend on the HTTP layer.
func GeoInfoFromResult(r geoip.Result) *GeoInfo {
	if r.IsEmpty() {
		return nil
	}
	info := &GeoInfo{}
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
