package anomaly

import (
	"context"
	"log/slog"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

// ScanState is the persisted incremental cursor. LastAccessLogID is the highest
// raw row already processed; LastBucketAt is the last complete hourly bucket
// evaluated (nil before the first bucket pass).
type ScanState struct {
	LastAccessLogID int64
	LastBucketAt    *time.Time
}

// ScanRepository is the persistence the job needs. It is deliberately narrow:
// the job orchestrates, the repository owns all SQL.
type ScanRepository interface {
	LoadScanState(ctx context.Context) (ScanState, error)
	MaxAccessLogID(ctx context.Context) (int64, error)
	LastAggregateBucketAt(ctx context.Context) (*time.Time, error)
	UpsertFinding(ctx context.Context, f Finding) error
	UpsertDeviceProfile(ctx context.Context, o ProfileObservation) error
	SaveScanState(ctx context.Context, s ScanState) error
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// ScanOptions carries the scan's tunables (resolved from ConfAnomaly by app.go)
// so the job depends on primitives, not the config struct.
type ScanOptions struct {
	Interval            time.Duration
	Sensitivity         string
	LearningDays        int
	DetectRules         bool
	DetectVolume        bool
	DetectNovelty       bool
	TravelSameContinent bool
}

func (o ScanOptions) familyEnabled(f Family) bool {
	switch f {
	case FamilyRules:
		return o.DetectRules
	case FamilyVolume:
		return o.DetectVolume
	case FamilyNovelty:
		return o.DetectNovelty
	default:
		return false
	}
}

// ScanJob is the scheduler.Job that drives one detection pass. It ticks with the
// rest of the scheduler at RULE_CHECK_INTERVAL but self-gates on opts.Interval so
// scans run at their own, coarser cadence.
type ScanJob struct {
	repo      ScanRepository
	detectors []Detector
	opts      ScanOptions
	lastRanAt time.Time
	logger    *slog.Logger
}

// NewScanJob builds the job. detectors may be empty — the job then persists only
// its watermark, which is the intended state until detector tasks land.
func NewScanJob(repo ScanRepository, detectors []Detector, opts ScanOptions, logger *slog.Logger) *ScanJob {
	return &ScanJob{
		repo:      repo,
		detectors: detectors,
		opts:      opts,
		logger:    logger.With(slog.String(logging.AttrKeyComponent, "anomaly_scan_job")),
	}
}

func (j *ScanJob) Run(ctx context.Context) error {
	now := time.Now()
	if !j.lastRanAt.IsZero() && now.Sub(j.lastRanAt) < j.opts.Interval {
		return nil
	}

	state, err := j.repo.LoadScanState(ctx)
	if err != nil {
		return err
	}

	// Snapshot the raw cursor and the complete-hour boundary once, so every
	// detector reads the same window even as new rows land during the pass.
	maxID, err := j.repo.MaxAccessLogID(ctx)
	if err != nil {
		return err
	}
	completeHour := now.Truncate(time.Hour)

	// The volume detectors read hourly_traffic_aggregates / hourly_attribution_
	// aggregates, built by the rollup job earlier in the same scheduler tick on
	// its own clock. If the hour boundary falls between rollup's Run and this
	// one, rollup may not yet have built the aggregate for the hour wall-clock
	// now considers complete: advancing the cursor past that gap would mark the
	// still-empty bucket observed, and once rollup catches up and populates it,
	// the next pass would classify it as history instead — silently losing any
	// spike inside it. Clamp to rollup's actual progress instead; a nil lastAgg
	// (no aggregates ever) keeps the wall-clock completeHour, since holding the
	// cursor at zero forever would re-classify all history on the first advance.
	// During zero-traffic hours MAX(bucket_at) lags harmlessly — there is
	// nothing to evaluate there, and the cursor catches up, one pass late, as
	// soon as traffic resumes, which the day-bucketed fingerprints absorb.
	lastAgg, err := j.repo.LastAggregateBucketAt(ctx)
	if err != nil {
		return err
	}
	if lastAgg != nil {
		if rollupComplete := lastAgg.Add(time.Hour); rollupComplete.Before(completeHour) {
			completeHour = rollupComplete
		}
	}

	scope := Scope{
		FromAccessLogID:     state.LastAccessLogID,
		ToAccessLogID:       maxID,
		FromBucket:          state.LastBucketAt,
		ToBucket:            completeHour,
		Now:                 now,
		Sensitivity:         j.opts.Sensitivity,
		LearningWindow:      time.Duration(j.opts.LearningDays) * 24 * time.Hour,
		TravelSameContinent: j.opts.TravelSameContinent,
	}

	var findings []Finding
	var observations []ProfileObservation
	detectorFailed := false
	for _, d := range j.detectors {
		if !j.opts.familyEnabled(d.Family()) {
			continue
		}
		found, err := d.Detect(ctx, scope)
		if err != nil {
			detectorFailed = true
			j.logger.ErrorContext(ctx, "anomaly detector failed",
				slog.String("family", string(d.Family())),
				slog.Any(logging.AttrKeyError, err),
			)
			continue
		}
		findings = append(findings, found...)
		// A profile-learning detector reports the sightings its pass observed
		// alongside its findings; only a clean pass contributes them, so a failed
		// detector's partial state never lands.
		if pl, ok := d.(ProfileLearner); ok {
			observations = append(observations, pl.ProfileObservations()...)
		}
	}

	// Upsert and watermark advance share one transaction: a crash mid-pass never
	// advances the cursor past unwritten findings. When any detector errored the
	// watermark is held so its window is rescanned next pass — findings dedupe by
	// fingerprint, so rescanning the clean detectors' rows is idempotent.
	err = j.repo.WithinTx(ctx, func(ctx context.Context) error {
		for _, f := range findings {
			if err := j.repo.UpsertFinding(ctx, f); err != nil {
				return err
			}
		}
		for _, o := range observations {
			if err := j.repo.UpsertDeviceProfile(ctx, o); err != nil {
				return err
			}
		}
		next := state
		if !detectorFailed {
			next.LastAccessLogID = maxID
			next.LastBucketAt = &completeHour
		}
		return j.repo.SaveScanState(ctx, next)
	})
	if err != nil {
		return err
	}

	j.lastRanAt = now
	return nil
}
