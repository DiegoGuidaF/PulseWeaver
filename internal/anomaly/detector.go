package anomaly

import (
	"context"
	"time"
)

// Detector reads a slice of the world and reports abnormal conditions. Detectors
// are pure readers: they never write anomalies, never see the dedup
// fingerprinting, and never touch scan state. Adding a new kind means adding one
// detector and registering it with the job — nothing else changes.
type Detector interface {
	// Family drives the per-family config toggle that enables this detector.
	Family() Family
	// Detect returns the findings visible within sc. A detector that fails
	// returns an error; the job isolates it so other detectors still run.
	Detect(ctx context.Context, sc Scope) ([]Finding, error)
}

// AllDetectors returns every detector wired to the repository, in scan order.
// Registration lives here so adding a kind is a one-line change (the
// encapsulation contract): the job, dedup, and API stay untouched. A nil geo
// resolver silences only the geo detector.
func AllDetectors(r *Repository, geo GeoResolver) []Detector {
	return []Detector{
		expiredAccessDetector{reader: r},
		invalidTokenDetector{reader: r},
		hostProbingDetector{reader: r},
		addressChurnDetector{reader: r},
		denySpikeDetector{reader: r},
		entityDriftDetector{reader: r},
		geoDeniedDetector{reader: r, geo: geo},
	}
}

// Scope is the immutable slice of the world a single scan pass observes. The job
// builds it once per run and hands the same value to every detector so they all
// agree on the window and the current time.
type Scope struct {
	// FromAccessLogID (exclusive) / ToAccessLogID (inclusive) bound the raw
	// access_log rows a raw-row detector may read.
	FromAccessLogID int64
	ToAccessLogID   int64
	// FromBucket (exclusive, nil on first scan) / ToBucket (exclusive) bound the
	// complete hourly buckets a volume detector may evaluate. ToBucket is the
	// current hour boundary, so the in-flight hour is never evaluated.
	FromBucket *time.Time
	ToBucket   time.Time
	// Now is injected so detectors compute trailing windows against a single,
	// testable clock rather than calling time.Now themselves.
	Now time.Time
	// Sensitivity is the preset name (low|medium|high) the volume family resolves
	// into multiplier/floor pairs.
	Sensitivity string
}
