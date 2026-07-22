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
	repo       *Repository
	ipProvider EnabledIPProvider
	geo        GeoResolver
	fleet      *FleetComposer
	logger     *slog.Logger
}

func NewHTTPHandler(repo *Repository, ipProvider EnabledIPProvider, geo GeoResolver, fleet *FleetComposer, logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		repo:       repo,
		ipProvider: ipProvider,
		geo:        geo,
		fleet:      fleet,
		logger:     logger.With(slog.String(logging.AttrKeyComponent, "queries")),
	}
}

func errorMsgResponse(msg string) httpapi.ErrorResponse {
	return httpapi.ErrorResponse{Error: &msg}
}
