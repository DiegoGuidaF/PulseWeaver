package device

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

// addressHistoryTuningLimit bounds the tuning candidate ranking.
const addressHistoryTuningLimit = 5

// tuningMinRenewals is the sample floor a device's renewal count must clear
// to qualify for the tuning ranking — below it, a p95 gap is just the
// maximum observed and not a reliable signal that the TTL is wrong.
const tuningMinRenewals = 10

// AddressHistoryTuningQuery is the validated window behind
// GET /address-history/tuning. Unlike AddressHistoryQuery it carries no
// filterx filters — the endpoint is window-scoped by contract, so there is
// nothing here for a filter to narrow.
type AddressHistoryTuningQuery struct {
	From     time.Time
	To       time.Time
	DeviceID *ids.DeviceID
}

// NewAddressHistoryTuningQuery normalizes GET /address-history/tuning params,
// defaulting to the same window as the other address-history endpoints.
func NewAddressHistoryTuningQuery(params httpapi.GetAddressHistoryTuningParams) AddressHistoryTuningQuery {
	now := time.Now().UTC()
	q := AddressHistoryTuningQuery{
		From: now.Add(-defaultHistoryRange),
		To:   now,
	}
	if params.From != nil {
		q.From = *params.From
	}
	if params.To != nil {
		q.To = *params.To
	}
	if params.DeviceId != nil {
		q.DeviceID = new(ids.DeviceID(*params.DeviceId))
	}
	return q
}

// tuningCandidateRow is one device's renewal-timing aggregates for the tuning
// window. Total is the count of every row that qualified, repeated identically
// on each row, so it survives the LIMIT that trims the rest.
type tuningCandidateRow struct {
	DeviceID         ids.DeviceID `db:"device_id"`
	DeviceName       string       `db:"device_name"`
	TTLSeconds       int64        `db:"ttl_seconds"`
	RenewalCount     int          `db:"renewal_count"`
	LateRenewalCount int          `db:"late_renewal_count"`
	P95GapSeconds    int64        `db:"p95_gap_seconds"`
	Total            int          `db:"total"`
}

// addressHistoryTuningPerDevice builds one row per device with a renewal in the
// window, carrying every aggregate the caller can filter or order on.
//
// `rn * 100 >= n * 95` picks the smallest gap at or above the 95th percentile
// without CEIL or integer-division rounding. Below ~20 renewals it resolves to
// the maximum gap, which is still the number to size a TTL against.
//
// Excluding unknown ttl_risk is what makes ttl_seconds safe to scan into a
// non-pointer: ttlRiskCase never classifies a row as anything else while the
// device has no lease rule.
func addressHistoryTuningPerDevice(q AddressHistoryTuningQuery) sq.SelectBuilder {
	cond := sq.And{
		sq.GtOrEq{"e.created_at": q.From},
		sq.LtOrEq{"e.created_at": q.To},
		sq.NotEq{"e.ttl_risk": string(TTLRiskUnknown)},
	}
	if q.DeviceID != nil {
		cond = append(cond, sq.Eq{"e.device_id": *q.DeviceID})
	}

	renewals := sq.Select(
		"e.device_id AS device_id",
		"e.device_name AS device_name",
		"e.ttl_seconds AS ttl_seconds",
		"e.renewal_gap_seconds AS renewal_gap_seconds",
		"CASE WHEN e.ttl_risk = 'breached' THEN 1 ELSE 0 END AS is_late",
	).FromSelect(addressHistoryEnriched(), "e").Where(cond)

	ranked := sq.Select(
		"g.device_id AS device_id",
		"g.device_name AS device_name",
		"g.ttl_seconds AS ttl_seconds",
		"g.renewal_gap_seconds AS renewal_gap_seconds",
		"g.is_late AS is_late",
		"ROW_NUMBER() OVER (PARTITION BY g.device_id ORDER BY g.renewal_gap_seconds) AS rn",
		"COUNT(*) OVER (PARTITION BY g.device_id) AS n",
	).FromSelect(renewals, "g")

	return sq.Select(
		"k.device_id AS device_id",
		"MAX(k.device_name) AS device_name",
		"MAX(k.ttl_seconds) AS ttl_seconds",
		"COUNT(*) AS renewal_count",
		"SUM(k.is_late) AS late_renewal_count",
		"MIN(CASE WHEN k.rn * 100 >= k.n * 95 THEN k.renewal_gap_seconds END) AS p95_gap_seconds",
	).FromSelect(ranked, "k").GroupBy("k.device_id")
}

// toTuningCandidates is always non-nil, so an empty result serializes as []
// rather than null.
func toTuningCandidates(rows []tuningCandidateRow) []httpapi.AddressHistoryTuningCandidate {
	candidates := make([]httpapi.AddressHistoryTuningCandidate, len(rows))
	for i, d := range rows {
		candidates[i] = httpapi.AddressHistoryTuningCandidate{
			DeviceId:         d.DeviceID.Int64(),
			DeviceName:       d.DeviceName,
			RenewalCount:     d.RenewalCount,
			LateRenewalCount: d.LateRenewalCount,
			TtlSeconds:       d.TTLSeconds,
			P95GapSeconds:    d.P95GapSeconds,
		}
	}
	return candidates
}

// GetAddressHistoryTuning answers the tuning endpoint: with no device_id, a
// top-5 ranking of every device whose TTL is genuinely too short in the window;
// with device_id, that one device's readout regardless of whether it meets the
// ranking threshold.
func (r *Repository) GetAddressHistoryTuning(ctx context.Context, q AddressHistoryTuningQuery) (httpapi.AddressHistoryTuningResponse, error) {
	// COUNT(*) OVER () counts the rows surviving the WHERE before the LIMIT
	// trims them, so the fleet-wide total and the page come from one read and
	// cannot disagree about who qualified.
	candidates := sq.Select(
		"x.device_id", "x.device_name", "x.ttl_seconds", "x.renewal_count",
		"x.late_renewal_count", "x.p95_gap_seconds", "COUNT(*) OVER () AS total",
	).FromSelect(addressHistoryTuningPerDevice(q), "x")

	// Scoped to one device there is nothing to rank, and a device the caller is
	// already looking at needs no threshold to justify showing it. p95 > ttl is
	// itself the ">5% of renewals arrived late" test, so no second late-ratio
	// condition sits beside it to drift from.
	if q.DeviceID == nil {
		candidates = candidates.
			Where(sq.Expr("x.p95_gap_seconds > x.ttl_seconds")).
			Where(sq.GtOrEq{"x.renewal_count": tuningMinRenewals}).
			OrderBy("(x.p95_gap_seconds * 1.0 / x.ttl_seconds) DESC", "x.late_renewal_count DESC", "x.device_id ASC").
			Limit(addressHistoryTuningLimit)
	}

	tuningSQL, tuningArgs, err := candidates.ToSql()
	if err != nil {
		return httpapi.AddressHistoryTuningResponse{}, fmt.Errorf("build address history tuning query: %w", err)
	}

	var rows []tuningCandidateRow
	if err := r.db.SelectContext(ctx, &rows, tuningSQL, tuningArgs...); err != nil {
		return httpapi.AddressHistoryTuningResponse{}, fmt.Errorf("get address history tuning: %w", err)
	}

	total := 0
	if len(rows) > 0 {
		total = rows[0].Total
	}

	return httpapi.AddressHistoryTuningResponse{
		Devices:     toTuningCandidates(rows),
		Total:       total,
		MinRenewals: tuningMinRenewals,
	}, nil
}
