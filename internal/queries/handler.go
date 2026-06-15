package queries

import (
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

// GeoResolver resolves an IP to geographic and ASN data. Declared on the
// consumer side (Go convention); *geoip.Lookup satisfies it. A nil resolver is
// valid — enrichment is skipped.
type GeoResolver interface {
	Resolve(ip string) geoip.Result
}

type HTTPHandler struct {
	repo         *Repository
	policyReader PolicyMapReader
	npProvider   AuditNetworkPoliciesProvider
	geo          GeoResolver
	logger       *slog.Logger
}

func NewHTTPHandler(repo *Repository, policyReader PolicyMapReader, npProvider AuditNetworkPoliciesProvider, geo GeoResolver, logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		repo:         repo,
		policyReader: policyReader,
		npProvider:   npProvider,
		geo:          geo,
		logger:       logger.With(slog.String(logging.AttrKeyComponent, "queries")),
	}
}

func errorMsgResponse(msg string) httpapi.ErrorResponse {
	return httpapi.ErrorResponse{Error: &msg}
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
