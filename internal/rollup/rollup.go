package rollup

import (
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
)

// GeoResolver resolves an IP to geographic and ASN data. Declared on the
// consumer side (Go convention); *geoip.Lookup satisfies it. A nil resolver is
// valid — enrichment is skipped.
type GeoResolver interface {
	Resolve(ip string) geoip.Result
}

// RawWindowThreshold is the maximum window size for which queries run directly
// against access_log. Windows wider than this use hourly_traffic_aggregates
// instead. The current in-flight hour is always absent from aggregates (rollup
// covers only complete hours), so any window ≤ 24h benefits from the raw path.
// Every widget that answers from traffic data must dispatch on this same
// threshold so that all widgets agree for a given window (see also
// queries.ListAccessLogStatsByCountry).
const RawWindowThreshold = 24 * time.Hour

// Repository provides both read and write access to traffic aggregates.
type Repository struct {
	db  *database.DB
	geo GeoResolver
}

func NewRepository(db *database.DB, geo GeoResolver) *Repository {
	return &Repository{
		db:  db,
		geo: geo,
	}
}
