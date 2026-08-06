package device

import (
	"context"
	"fmt"
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

// AddressHistoryBucketRow holds a plain event count for one time bucket.
// Timestamp uses DBTime because SQLite's strftime returns TEXT even for
// DATETIME columns, and DBTime handles the multi-format scanning.
type AddressHistoryBucketRow struct {
	Timestamp  database.DBTime `db:"bucket"`
	EventCount int             `db:"event_count"`
}

// GetAddressHistoryHistogram returns a plain event count per time bucket,
// over the same filtered rows GetAddressHistoryEvents counts — never affected
// by that endpoint's cursor, since the histogram carries no cursor parameter.
// Granularity is chosen server-side from the window size.
func (r *Repository) GetAddressHistoryHistogram(ctx context.Context, q AddressHistoryQuery) ([]AddressHistoryBucketRow, error) {
	base, err := addressHistoryBase(q)
	if err != nil {
		return nil, err
	}

	bucketExpr := timebucket.GranularityForWindow(q.To.Sub(q.From)).BucketExpr("e.created_at")

	bucketsSQL, bucketArgs, err := base.
		Column(bucketExpr + " AS bucket").
		Column("COUNT(*) AS event_count").
		GroupBy("bucket").
		OrderBy("bucket ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build address history histogram query: %w", err)
	}

	var buckets []AddressHistoryBucketRow
	if err := r.db.SelectContext(ctx, &buckets, bucketsSQL, bucketArgs...); err != nil {
		return nil, fmt.Errorf("get address history histogram: %w", err)
	}
	if buckets == nil {
		buckets = []AddressHistoryBucketRow{}
	}
	return buckets, nil
}
