// Package anomaly runs a periodic background scan over the access log and
// traffic aggregates, producing deduplicated findings in the anomalies table
// for an operator to review. Detection is observation only — it never touches
// the verify hot path.
package anomaly

import (
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

// Kind identifies a detector's output, grouped by the family that produces it.
type Kind string

const (
	// Rules family — deterministic, no statistics.
	KindExpiredAccess Kind = "expired_access"
	KindInvalidToken  Kind = "invalid_token"

	// Volume family — statistical baselines and windowed thresholds.
	KindDenySpike    Kind = "deny_spike"
	KindEntityDrift  Kind = "entity_drift"
	KindGeoDenied    Kind = "geo_denied"
	KindHostProbing  Kind = "host_probing"
	KindAddressChurn Kind = "address_churn"

	// Novelty family — per-device profiles and geo-velocity.
	KindNewUserAgent     Kind = "new_user_agent"
	KindNewCountry       Kind = "new_country"
	KindImpossibleTravel Kind = "impossible_travel"
)

// Severity ranks how much an operator should care about a finding.
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

// Status is the lifecycle of a persisted anomaly. Dedup keeps at most one open
// row per fingerprint; acknowledging a row lets a later recurrence open a new one.
type Status string

const (
	StatusOpen         Status = "open"
	StatusAcknowledged Status = "acknowledged"
)

// Family groups detectors so a single config toggle enables or disables all
// detectors that share a data source and failure mode.
type Family string

const (
	FamilyRules   Family = "rules"
	FamilyVolume  Family = "volume"
	FamilyNovelty Family = "novelty"
)

// Anomaly is a persisted detection finding. Attribution ids are nullable and
// their names denormalized so the row stays readable after the device or user
// is deleted.
type Anomaly struct {
	ID          int64
	Kind        Kind
	Severity    Severity
	Status      Status
	Fingerprint string
	FirstSeenAt time.Time
	LastSeenAt  time.Time
	DeviceID    *ids.DeviceID
	DeviceName  string
	UserID      *ids.UserID
	UserName    string
	ClientIP    *string
	TargetHost  *string
	CountryCode *string
	Evidence    map[string]any
}

// Finding is what a detector emits for one abnormal condition. The detector owns
// the fingerprint (its composition is kind-specific — some kinds bucket by UTC
// day, others key on device + IP) but never sees the dedup upsert itself: the
// job turns findings into inserts-or-updates against the open-row uniqueness.
type Finding struct {
	Kind        Kind
	Severity    Severity
	Fingerprint string
	DeviceID    *ids.DeviceID
	DeviceName  string
	UserID      *ids.UserID
	UserName    string
	ClientIP    *string
	TargetHost  *string
	CountryCode *string
	Evidence    map[string]any
	// ObservedAt is the point in the scan window the condition was seen; it
	// seeds first_seen_at on insert and advances last_seen_at on recurrence.
	ObservedAt time.Time
}
