package device

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/queries/filterx"
	"github.com/DiegoGuidaF/PulseWeaver/internal/timebucket"
)

const (
	defaultHistoryLimit = 50
	maxHistoryLimit     = 200
	defaultHistoryRange = 24 * time.Hour
)

// addressHistoryTuningLimit bounds the tuning candidate ranking.
const addressHistoryTuningLimit = 5

// tuningMinRenewals is the sample floor a device's renewal count must clear
// to qualify for the tuning ranking — below it, a p95 gap is just the
// maximum observed and not a reliable signal that the TTL is wrong.
const tuningMinRenewals = 10

// ttlRisk ratio thresholds — see TTLRisk.
const (
	ttlRiskBreachedRatio    = 1.0
	ttlRiskCriticalRatio    = 0.9
	ttlRiskApproachingRatio = 0.7
)

// addressHistoryRegistry is the column allowlist (ADR-007) for the
// address-history events and histogram queries. Every expression references
// the outer alias "e" of the enriched derived table addressHistoryEnriched
// builds — event_kind and ttl_risk are derived columns, not base-table
// columns, so both endpoints must filter over that table, not the raw joins.
// No sortable columns: sort is fixed at id DESC by design, so the sort map is
// empty and the tiebreaker is unused.
var addressHistoryRegistry = filterx.NewRegistry(
	map[string]filterx.ColumnSpec{
		"device_id": {
			Expr: "e.device_id",
			Ops:  []filterx.Operator{filterx.OpIn, filterx.OpNotIn},
		},
		"user_id": {
			Expr: "e.owner_id",
			Ops:  []filterx.Operator{filterx.OpIn, filterx.OpNotIn},
		},
		"ip": {
			Expr: "e.ip",
			Ops:  []filterx.Operator{filterx.OpIn, filterx.OpNotIn, filterx.OpContains, filterx.OpNotContains},
		},
		"source": {
			Expr: "e.source",
			Ops:  []filterx.Operator{filterx.OpIn, filterx.OpNotIn},
		},
		"event_kind": {
			Expr: "e.event_kind",
			Ops:  []filterx.Operator{filterx.OpIn, filterx.OpNotIn},
		},
		"ttl_risk": {
			Expr: "e.ttl_risk",
			Ops:  []filterx.Operator{filterx.OpIn, filterx.OpNotIn},
		},
	},
	map[string]filterx.SortSpec{},
	"e.id",
)

// renewalGapCTE computes, per device, the gap since the previous *renewal*
// event only (any event except a server-generated expiry/limit_exceeded
// termination — those disable a lease, they don't renew one). SQLite cannot
// exclude rows from a window frame while keeping them in the result, so the
// LAG runs here over renewal rows only and addressHistoryEnriched LEFT JOINs
// it back onto the full event set by id: a non-renewal row then carries a
// NULL gap instead of being measured against a termination that didn't renew
// anything (which would misclassify every routine expiry as a TTL breach).
var renewalGapCTE = fmt.Sprintf(`WITH renewal_gap AS (
	SELECT
		aev.id AS event_id,
		LAG(aev.created_at) OVER (PARTITION BY a.device_id ORDER BY aev.created_at, aev.id) AS prev_renewal_at
	FROM address_events aev
	JOIN addresses a ON a.id = aev.address_id
	WHERE aev.source NOT IN ('%s', '%s')
)`, EventSourceExpiry, EventSourceLimitExceeded)

// eventKindCase classifies each event against the immediately preceding event
// for the *same address* (not device), independent of any time window — the
// same unbounded correlated-subquery form as the legacy state-change filter,
// just moved from a WHERE clause into a SELECT. Computing this from the
// renewal-gap LAG instead would reclassify rows at the window's leading edge
// as the caller pans from/to; keeping it a per-address comparison avoids that.
const eventKindCase = `CASE
	WHEN NOT EXISTS (
		SELECT 1 FROM address_events prev
		WHERE prev.address_id = aev.address_id AND prev.id < aev.id
	) THEN 'created'
	WHEN aev.is_enabled != (
		SELECT prev.is_enabled FROM address_events prev
		WHERE prev.address_id = aev.address_id AND prev.id < aev.id
		ORDER BY prev.id DESC LIMIT 1
	) THEN CASE WHEN aev.is_enabled THEN 'enabled' ELSE 'disabled' END
	ELSE 'refresh'
END`

// ttlRiskCase classifies renewal_gap_seconds against ttl_seconds. Both are
// NULL for a non-renewal row or one with no lease rule, which is exactly
// "unknown" — a row with no gap to measure is never at risk.
const ttlRiskCase = `CASE
	WHEN r.ttl_seconds IS NULL OR r.renewal_gap_seconds IS NULL THEN 'unknown'
	WHEN CAST(r.renewal_gap_seconds AS REAL) / r.ttl_seconds >= ? THEN 'breached'
	WHEN CAST(r.renewal_gap_seconds AS REAL) / r.ttl_seconds > ? THEN 'critical'
	WHEN CAST(r.renewal_gap_seconds AS REAL) / r.ttl_seconds > ? THEN 'approaching'
	ELSE 'ok'
END`

// ttlRiskOrder ranks the TTLRisk levels ttlRiskCase can emit, least to most
// severe. It is the single source the histogram's worst-risk-per-device
// aggregation and the at-risk device ranking build their SQL rank from — the
// rank must never be re-derived independently of this list, or it can drift
// from what ttlRiskCase actually classifies.
var ttlRiskOrder = []TTLRisk{TTLRiskUnknown, TTLRiskOK, TTLRiskApproaching, TTLRiskCritical, TTLRiskBreached}

// ttlRiskOKRank is the lowest rank the histogram buckets count — unknown
// stays excluded, since a row with no gap to measure is not a renewal and
// belongs in neither the numerator nor the denominator.
var ttlRiskOKRank = slices.Index(ttlRiskOrder, TTLRiskOK)

// ttlRiskRankExpr renders a CASE mapping col — an expression producing one of
// the TTLRisk string values — to its severity rank per ttlRiskOrder.
func ttlRiskRankExpr(col string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "CASE %s", col)
	for rank, risk := range ttlRiskOrder {
		fmt.Fprintf(&b, " WHEN '%s' THEN %d", risk, rank)
	}
	b.WriteString(" END")
	return b.String()
}

// addressHistoryEnriched returns the enriched derived table both the events
// and histogram queries read from and filter over. It is deliberately
// unbounded by time: event_kind and the renewal gap are both defined to be
// window-independent (see eventKindCase and renewalGapCTE), so baking a
// from/to bound into this FROM would make their classification shift as the
// caller pans the window. from/to apply only in the outer WHERE built by
// addressHistoryConditions. The table is event-scale but slow-growing
// (~11k rows); if a full scan here ever needs bounding, bound the CTE at
// from minus the largest configured TTL, not at from.
func addressHistoryEnriched() sq.SelectBuilder {
	raw := sq.Select(
		"aev.id",
		"aev.address_id",
		"aev.created_at",
		"a.ip",
		"aev.is_enabled",
		"aev.source",
		"a.device_id",
		"d.name AS device_name",
		"d.owner_id",
		"json_extract(dr.config, '$.ttl_seconds') AS ttl_seconds",
		"CAST((julianday(aev.created_at) - julianday(rg.prev_renewal_at)) * 86400 AS INTEGER) AS renewal_gap_seconds",
		eventKindCase+" AS event_kind",
	).
		From("address_events aev").
		Join("addresses a ON a.id = aev.address_id").
		Join("devices d ON d.id = a.device_id").
		LeftJoin("device_rules dr ON dr.device_id = a.device_id AND dr.rule_type = 'device_lease' AND dr.enabled = 1").
		LeftJoin("renewal_gap rg ON rg.event_id = aev.id").
		Where("d.deleted_at IS NULL").
		Prefix(renewalGapCTE)

	return sq.Select(
		"r.id", "r.address_id", "r.created_at", "r.ip", "r.is_enabled", "r.source",
		"r.device_id", "r.device_name", "r.owner_id", "r.ttl_seconds", "r.renewal_gap_seconds",
		"r.event_kind",
	).
		Column(ttlRiskCase+" AS ttl_risk", ttlRiskBreachedRatio, ttlRiskCriticalRatio, ttlRiskApproachingRatio).
		FromSelect(raw, "r")
}

// AddressHistoryQuery is the validated, normalized form of an address-history
// request — shared by the events and histogram endpoints. BeforeID and Limit
// are only meaningful for the events page.
type AddressHistoryQuery struct {
	From     time.Time
	To       time.Time
	Filters  []filterx.Filter
	BeforeID *int64
	Limit    int
}

// addressHistoryFilterParams collects the fields the events and histogram
// OpenAPI params structs share, so the filter-parsing logic underneath is
// written once for both.
type addressHistoryFilterParams struct {
	From        *time.Time
	To          *time.Time
	DeviceID    *[]httpapi.ID
	DeviceIDOp  *httpapi.AddressHistoryFilterOperator
	UserID      *[]httpapi.ID
	UserIDOp    *httpapi.AddressHistoryFilterOperator
	IP          *[]string
	IPOp        *httpapi.AddressHistoryFilterOperator
	Source      *[]httpapi.AddressEventSource
	SourceOp    *httpapi.AddressHistoryFilterOperator
	EventKind   *[]httpapi.AddressEventKind
	EventKindOp *httpapi.AddressHistoryFilterOperator
	TTLRisk     *[]httpapi.TTLRisk
	TTLRiskOp   *httpapi.AddressHistoryFilterOperator
}

// newAddressHistoryQuery validates and normalizes the filter/window portion
// shared by both address-history endpoints. Returns an error wrapping
// filterx.ErrInvalidFilter for an unknown operator/column or an over-cap
// value list — the handler maps this to a 400.
func newAddressHistoryQuery(p addressHistoryFilterParams) (AddressHistoryQuery, error) {
	now := time.Now().UTC()
	q := AddressHistoryQuery{
		From: now.Add(-defaultHistoryRange),
		To:   now,
	}
	if p.From != nil {
		q.From = *p.From
	}
	if p.To != nil {
		q.To = *p.To
	}

	valueFilters := []struct {
		column string
		values []any
		op     *httpapi.AddressHistoryFilterOperator
	}{
		{"device_id", filterx.Int64Values(p.DeviceID), p.DeviceIDOp},
		{"user_id", filterx.Int64Values(p.UserID), p.UserIDOp},
		{"ip", filterx.StringValues(p.IP), p.IPOp},
		{"source", filterx.StringValues(p.Source), p.SourceOp},
		{"event_kind", filterx.StringValues(p.EventKind), p.EventKindOp},
		{"ttl_risk", filterx.StringValues(p.TTLRisk), p.TTLRiskOp},
	}
	for _, vf := range valueFilters {
		filter, ok, err := filterx.ParseFilter(vf.column, vf.values, vf.op)
		if err != nil {
			return AddressHistoryQuery{}, err
		}
		if !ok {
			continue
		}
		if err := addressHistoryRegistry.Validate(filter); err != nil {
			return AddressHistoryQuery{}, err
		}
		q.Filters = append(q.Filters, filter)
	}

	return q, nil
}

// NewAddressHistoryQuery validates and normalizes GET /address-history params.
func NewAddressHistoryQuery(params httpapi.GetAddressHistoryParams) (AddressHistoryQuery, error) {
	q, err := newAddressHistoryQuery(addressHistoryFilterParams{
		From: params.From, To: params.To,
		DeviceID: params.DeviceId, DeviceIDOp: params.DeviceIdOp,
		UserID: params.UserId, UserIDOp: params.UserIdOp,
		IP: params.Ip, IPOp: params.IpOp,
		Source: params.Source, SourceOp: params.SourceOp,
		EventKind: params.EventKind, EventKindOp: params.EventKindOp,
		TTLRisk: params.TtlRisk, TTLRiskOp: params.TtlRiskOp,
	})
	if err != nil {
		return AddressHistoryQuery{}, err
	}

	limit := defaultHistoryLimit
	if params.Limit != nil {
		limit = *params.Limit
	}
	if limit <= 0 {
		limit = defaultHistoryLimit
	}
	if limit > maxHistoryLimit {
		limit = maxHistoryLimit
	}
	q.Limit = limit
	q.BeforeID = params.BeforeId

	return q, nil
}

// NewAddressHistoryHistogramQuery validates and normalizes
// GET /address-history/histogram params.
func NewAddressHistoryHistogramQuery(params httpapi.GetAddressHistoryHistogramParams) (AddressHistoryQuery, error) {
	return newAddressHistoryQuery(addressHistoryFilterParams{
		From: params.From, To: params.To,
		DeviceID: params.DeviceId, DeviceIDOp: params.DeviceIdOp,
		UserID: params.UserId, UserIDOp: params.UserIdOp,
		IP: params.Ip, IPOp: params.IpOp,
		Source: params.Source, SourceOp: params.SourceOp,
		EventKind: params.EventKind, EventKindOp: params.EventKindOp,
		TTLRisk: params.TtlRisk, TTLRiskOp: params.TtlRiskOp,
	})
}

// addressHistoryConditions builds the shared WHERE set — the from/to window
// (an ordinary bound, not a filterx column) plus every registry filter — fed
// to the count, page, and histogram queries so none of them can drift from
// the others.
func addressHistoryConditions(q AddressHistoryQuery) (sq.And, error) {
	cond := sq.And{
		sq.GtOrEq{"e.created_at": q.From},
		sq.LtOrEq{"e.created_at": q.To},
	}
	for _, f := range q.Filters {
		c, err := addressHistoryRegistry.Condition(f)
		if err != nil {
			return nil, err
		}
		cond = append(cond, c)
	}
	return cond, nil
}

// addressHistoryBase returns the base builder over the enriched, filtered
// derived table (aliased "e"): a FROM + WHERE, not just a WHERE, since
// event_kind and ttl_risk only exist on the enriched table. Callers attach
// SELECT columns, GROUP BY/ORDER BY, and LIMIT on top — the FROM and WHERE
// stay identical across the count, events page, and histogram queries.
func addressHistoryBase(q AddressHistoryQuery) (sq.SelectBuilder, error) {
	cond, err := addressHistoryConditions(q)
	if err != nil {
		return sq.SelectBuilder{}, err
	}
	return sq.Select().FromSelect(addressHistoryEnriched(), "e").Where(cond), nil
}

// AddressHistoryEventRow is one enriched address event.
type AddressHistoryEventRow struct {
	ID                int64            `db:"id"`
	CreatedAt         time.Time        `db:"created_at"`
	IP                string           `db:"ip"`
	IsEnabled         bool             `db:"is_enabled"`
	Source            EventSource      `db:"source"`
	DeviceID          ids.DeviceID     `db:"device_id"`
	DeviceName        string           `db:"device_name"`
	RenewalGapSeconds *int64           `db:"renewal_gap_seconds"`
	EventKind         AddressEventKind `db:"event_kind"`
	TTLRisk           TTLRisk          `db:"ttl_risk"`
	TTLSeconds        *int64           `db:"ttl_seconds"`
}

// AddressHistoryEvents is a page of enriched events plus the total matching
// the filters (ignoring the cursor), for pagination.
type AddressHistoryEvents struct {
	Events []AddressHistoryEventRow
	Total  int
}

// GetAddressHistoryEvents returns a keyset-paginated page of address events,
// newest first, plus the total matching the filters.
func (r *Repository) GetAddressHistoryEvents(ctx context.Context, q AddressHistoryQuery) (AddressHistoryEvents, error) {
	base, err := addressHistoryBase(q)
	if err != nil {
		return AddressHistoryEvents{}, err
	}

	countSQL, countArgs, err := base.Column("COUNT(*)").ToSql()
	if err != nil {
		return AddressHistoryEvents{}, fmt.Errorf("build address history count query: %w", err)
	}
	var total int
	if err := r.db.GetContext(ctx, &total, countSQL, countArgs...); err != nil {
		return AddressHistoryEvents{}, fmt.Errorf("count address history events: %w", err)
	}

	page := base.
		Columns(
			"e.id", "e.created_at", "e.ip", "e.is_enabled", "e.source",
			"e.device_id", "e.device_name", "e.ttl_seconds", "e.renewal_gap_seconds",
			"e.event_kind", "e.ttl_risk",
		)
	if q.BeforeID != nil {
		page = page.Where(sq.Lt{"e.id": *q.BeforeID})
	}
	page = page.OrderBy("e.id DESC").Limit(uint64(q.Limit))

	pageSQL, pageArgs, err := page.ToSql()
	if err != nil {
		return AddressHistoryEvents{}, fmt.Errorf("build address history events query: %w", err)
	}

	var events []AddressHistoryEventRow
	if err := r.db.SelectContext(ctx, &events, pageSQL, pageArgs...); err != nil {
		return AddressHistoryEvents{}, fmt.Errorf("get address history events: %w", err)
	}
	if events == nil {
		events = []AddressHistoryEventRow{}
	}

	return AddressHistoryEvents{Events: events, Total: total}, nil
}

// addressHistoryRiskBucketRow holds the distinct-device count at one risk
// rank within one time bucket. Timestamp uses DBTime because SQLite's
// strftime returns TEXT even for DATETIME columns, and DBTime handles the
// multi-format scanning.
type addressHistoryRiskBucketRow struct {
	Timestamp   database.DBTime `db:"bucket"`
	RiskRank    int             `db:"risk_rank"`
	DeviceCount int             `db:"device_count"`
}

// riskDeviceBuckets counts, per bucket and risk level, the distinct devices
// whose worst ttl_risk within that bucket is that level — so a device with
// both an approaching and a critical event in the same bucket is counted
// once, under critical, rather than in both bands. This needs two
// aggregation levels: MAX(risk_rank) per (bucket, device_id) first, then
// COUNT(*) grouped by (bucket, risk_rank) over that — a single-level
// COUNT(DISTINCT device_id) per (bucket, risk_rank) would still double-count
// a device that crossed bands within the bucket. Only unknown is excluded
// before the outer GROUP BY: a row with no gap to measure is not a renewal,
// so it belongs in neither band, but ok stays in — a bucket where every
// device renewed comfortably is exactly the denominator the chart needs.
// This result only carries buckets something happened in; the caller fills
// in the buckets it omits (nothing renewed at all) with all-zero counts to
// keep the response contiguous across the window.
func (r *Repository) riskDeviceBuckets(ctx context.Context, q AddressHistoryQuery, bucketExpr string) ([]addressHistoryRiskBucketRow, error) {
	base, err := addressHistoryBase(q)
	if err != nil {
		return nil, err
	}

	perDeviceBucket := base.
		Column(bucketExpr+" AS bucket").
		Column("e.device_id AS device_id").
		Column("MAX("+ttlRiskRankExpr("e.ttl_risk")+") AS risk_rank").
		GroupBy("bucket", "e.device_id")

	bucketsSQL, bucketArgs, err := sq.Select("x.bucket AS bucket", "x.risk_rank AS risk_rank", "COUNT(*) AS device_count").
		FromSelect(perDeviceBucket, "x").
		Where(sq.GtOrEq{"x.risk_rank": ttlRiskOKRank}).
		GroupBy("x.bucket", "x.risk_rank").
		OrderBy("x.bucket ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build address history risk buckets query: %w", err)
	}

	var buckets []addressHistoryRiskBucketRow
	if err := r.db.SelectContext(ctx, &buckets, bucketsSQL, bucketArgs...); err != nil {
		return nil, fmt.Errorf("get address history risk buckets: %w", err)
	}
	return buckets, nil
}

// foldRiskBuckets turns the (bucket, risk rank) rows riskDeviceBuckets
// returns into one response bucket per entry in sequence, each carrying all
// four bands. The sequence, not the rows, drives the result: a bucket the
// query never saw (nothing renewed in it) still appears, all-zero, so the
// response stays contiguous across the window and a caller never has to
// reconstruct a gap it cannot see. Rows outside sequence — which the shared
// from/to WHERE should already exclude — are dropped rather than extending
// the window.
func foldRiskBuckets(sequence []time.Time, rows []addressHistoryRiskBucketRow) []httpapi.AddressHistoryBucket {
	byBucket := make(map[int64]map[TTLRisk]int, len(rows))
	for _, rb := range rows {
		key := rb.Timestamp.UTC().Unix()
		bands, seen := byBucket[key]
		if !seen {
			bands = make(map[TTLRisk]int, 4)
			byBucket[key] = bands
		}
		bands[ttlRiskOrder[rb.RiskRank]] = rb.DeviceCount
	}

	buckets := make([]httpapi.AddressHistoryBucket, len(sequence))
	for i, ts := range sequence {
		// A bucket with no row folds to a nil map, which reads as zero.
		bands := byBucket[ts.Unix()]
		buckets[i] = httpapi.AddressHistoryBucket{
			Timestamp:              httpapi.UTCTime(ts),
			OkDeviceCount:          bands[TTLRiskOK],
			ApproachingDeviceCount: bands[TTLRiskApproaching],
			CriticalDeviceCount:    bands[TTLRiskCritical],
			BreachedDeviceCount:    bands[TTLRiskBreached],
		}
	}
	return buckets
}

// GetAddressHistoryHistogram answers the histogram endpoint: per-bucket
// device counts across all four TTL-risk bands, contiguous across the whole
// window. Granularity is chosen server-side from the window size.
func (r *Repository) GetAddressHistoryHistogram(ctx context.Context, q AddressHistoryQuery) (httpapi.AddressHistoryHistogramResponse, error) {
	granularity := timebucket.GranularityForWindow(q.To.Sub(q.From))
	bucketExpr := granularity.BucketExpr("e.created_at")

	riskBuckets, err := r.riskDeviceBuckets(ctx, q, bucketExpr)
	if err != nil {
		return httpapi.AddressHistoryHistogramResponse{}, err
	}

	buckets := foldRiskBuckets(granularity.Sequence(q.From, q.To), riskBuckets)

	return httpapi.AddressHistoryHistogramResponse{
		Buckets: buckets,
	}, nil
}

// AddressHistoryTuningQuery is the validated window behind
// GET /address-history/tuning. Unlike AddressHistoryQuery it carries no
// filterx filters — the endpoint is window-scoped only, by design, so there
// is nothing here for a filter to narrow. DeviceID, when set, scopes the
// response to that one device and bypasses the ranking threshold entirely.
type AddressHistoryTuningQuery struct {
	From     time.Time
	To       time.Time
	DeviceID *ids.DeviceID
}

// NewAddressHistoryTuningQuery validates and normalizes
// GET /address-history/tuning params. There is nothing here that can fail —
// from/to fall back to the same default window as the other address-history
// endpoints, and device_id needs no validation beyond its own type.
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
// window. Total is the window count of every row that qualified, repeated
// identically on each row, so it survives the LIMIT that trims the rest.
type tuningCandidateRow struct {
	DeviceID         ids.DeviceID `db:"device_id"`
	DeviceName       string       `db:"device_name"`
	TTLSeconds       int64        `db:"ttl_seconds"`
	RenewalCount     int          `db:"renewal_count"`
	LateRenewalCount int          `db:"late_renewal_count"`
	P95GapSeconds    int64        `db:"p95_gap_seconds"`
	Total            int          `db:"total"`
}

// addressHistoryTuningPerDevice returns the base builder over one row per
// device with a renewal in the window: its renewal/late-renewal counts, its
// configured TTL, and its nearest-rank p95 renewal gap. `rn * 100 >= n * 95`
// selects the smallest gap at or above the 95th percentile without CEIL or
// integer-division rounding; with few renewals this collapses to the
// maximum, which is the right number to size a TTL against. p95 has to be
// computed before the threshold can be applied — it is an input to
// selection, not a decoration on the output — so this builder computes
// every aggregate the caller might filter or order on in one pass; callers
// attach the qualifying WHERE, ORDER BY, and LIMIT on top. A device only
// appears here if it has at least one renewal (ttl_risk not unknown) in the
// window, which also guarantees ttl_seconds is non-null for every row
// (ttlRiskCase never classifies a row as non-unknown without one).
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

// toTuningCandidates maps ranked/device-scoped rows to the API shape. Always
// non-nil, so an empty result serializes as [] rather than null.
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
// top-5 ranking of every device whose TTL is genuinely too short in the
// window; with device_id, that one device's readout regardless of whether it
// meets the ranking threshold.
func (r *Repository) GetAddressHistoryTuning(ctx context.Context, q AddressHistoryTuningQuery) (httpapi.AddressHistoryTuningResponse, error) {
	// COUNT(*) OVER () counts the rows surviving the WHERE before the LIMIT
	// trims them, so the fleet-wide total and the page it returns come from one
	// read and cannot disagree about who qualified.
	candidates := sq.Select(
		"x.device_id", "x.device_name", "x.ttl_seconds", "x.renewal_count",
		"x.late_renewal_count", "x.p95_gap_seconds", "COUNT(*) OVER () AS total",
	).FromSelect(addressHistoryTuningPerDevice(q), "x")

	// Scoped to one device there is nothing to rank, and a device the caller is
	// already looking at needs no threshold to justify showing it — so the
	// qualifying test, the ordering, and the limit all apply to the fleet-wide
	// read only. p95 > ttl is exactly a >5% late rate (approached from above as
	// the sample grows), so there is deliberately no second late-ratio test
	// beside it that could drift.
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
