package dashboard

import (
	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
)

// SummaryStats holds aggregate counts for the stat cards. The three deny-reason
// fields partition DeniedCount: DenyIPNotRegistered + DenyHostNotAllowed +
// DenyOther always equals DeniedCount.
type SummaryStats struct {
	TotalRequests       int64 `db:"total_requests"`
	AllowedCount        int64 `db:"allowed_count"`
	DeniedCount         int64 `db:"denied_count"`
	DenyIPNotRegistered int64 `db:"deny_ip_not_registered"`
	DenyHostNotAllowed  int64 `db:"deny_host_not_allowed"`
	DenyOther           int64 `db:"deny_other"`
	UniqueIPs           int64 `db:"unique_ips"`
	AvgDurationUs       int64 `db:"avg_duration_us"`
}

// TrafficBucket holds allow/deny counts for a single time bucket.
type TrafficBucket struct {
	Timestamp  database.DBTime `db:"timestamp"`
	AllowCount int64           `db:"allow_count"`
	DenyCount  int64           `db:"deny_count"`
}

// IPCount pairs an IP address with its request count. Geo is resolved on read
// (not from the database) and is empty when the IP is unresolvable or GeoIP is
// disabled.
type IPCount struct {
	IP    string `db:"ip"`
	Count int64  `db:"count"`
	Geo   geoip.Result
}

// ServiceCount holds per-host allow/deny counts.
type ServiceCount struct {
	Host       string `db:"host"`
	AllowCount int64  `db:"allow_count"`
	DenyCount  int64  `db:"deny_count"`
}

// AttributionKind identifies which entity an attribution split is grouped by.
type AttributionKind string

const (
	AttributionKindPolicy AttributionKind = "policy"
	AttributionKindUser   AttributionKind = "user"
	AttributionKindDevice AttributionKind = "device"
)

// AttributionCount holds per-entity allow/deny counts for one attribution kind.
// EntityID is nil when the entity has been hard-deleted; EntityName is
// denormalized at rollup time and survives deletion.
type AttributionCount struct {
	EntityID   *int64 `db:"entity_id"`
	EntityName string `db:"entity_name"`
	AllowCount int64  `db:"allow_count"`
	DenyCount  int64  `db:"deny_count"`
}
