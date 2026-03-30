package geoip

// Result holds all resolved geo/ASN data for a single IP.
// All fields are empty/zero if the IP is not in the database
// (private ranges, IPv6 gaps, unknown).
type Result struct {
	CountryCode   string // ISO 3166-1 alpha-2, e.g. "DE"
	CountryName   string // e.g. "Germany"
	ContinentCode string // e.g. "EU"
	ASN           uint   // Autonomous System Number, e.g. 13335
	ASNOrg        string // e.g. "Cloudflare, Inc."
}

// IsEmpty returns true when the lookup found no data.
func (r Result) IsEmpty() bool {
	return r.CountryCode == "" && r.CountryName == "" && r.ContinentCode == "" && r.ASN == 0 && r.ASNOrg == ""
}
