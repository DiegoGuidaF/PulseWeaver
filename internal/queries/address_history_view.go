package queries

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/timebucket"
)

// AddressEventBucket holds aggregated activity data for one time period.
// Timestamp uses DBTime because SQLite's strftime returns TEXT even for
// DATETIME columns, and DBTime handles the multi-format scanning.
type AddressEventBucket struct {
	Timestamp   database.DBTime `db:"bucket"`
	ActiveCount int             `db:"active_count"`
	GapCount    int             `db:"gap_count"`
	EventCount  int             `db:"event_count"`
}

// AddressEventView represents a single recorded address event enriched with
// comparisons against the immediately preceding event for the same device
// (across all of that device's addresses), plus the device's configured
// address-lease TTL. These comparisons let the frontend surface whether a
// device's heartbeat cadence is keeping pace with its lease TTL, and whether
// an event represents real movement (IP/state change) or a plain refresh.
type AddressEventView struct {
	ID             int64                      `db:"id"`
	CreatedAt      time.Time                  `db:"created_at"`
	IP             string                     `db:"ip"`
	IsEnabled      bool                       `db:"is_enabled"`
	Source         httpapi.AddressEventSource `db:"source"`
	DeviceID       ids.DeviceID               `db:"device_id"`
	DeviceName     string                     `db:"device_name"`
	TimeGapSeconds *int64                     `db:"time_gap_seconds"`
	IPChanged      bool                       `db:"ip_changed"`
	IsRefresh      bool                       `db:"is_refresh"`
	TTLSeconds     *int64                     `db:"ttl_seconds"`
}

// AddressHistory holds the complete history response.
type AddressHistory struct {
	Buckets     []AddressEventBucket
	Events      []AddressEventView
	TotalEvents int
	QueryLimit  int // effective limit used for the query, needed for cursor logic
}

// AddressHistoryQuery encapsulates all filters and pagination for history queries.
type AddressHistoryQuery struct {
	From        time.Time
	To          time.Time
	Granularity timebucket.Granularity
	DeviceIDs   []ids.DeviceID // empty = all devices
	Source      *string
	IsEnabled   *bool
	IP          *string
	BeforeID    *int64 // cursor for events pagination
	Limit       int    // events limit (default 50, max 200)
	IncludeAll  bool   // when false (default), only state-change events are returned
}

const (
	defaultHistoryLimit = 50
	maxHistoryLimit     = 200
	defaultHistoryRange = 24 * time.Hour
)

// Validate normalizes defaults and validates business rules on the query.
// Must be called before passing the query to the repository.
func (q *AddressHistoryQuery) Validate() error {
	g, err := timebucket.ParseGranularity(string(q.Granularity))
	if err != nil {
		return err
	}
	q.Granularity = g

	now := time.Now().UTC()
	if q.From.IsZero() {
		q.From = now.Add(-defaultHistoryRange)
	}
	if q.To.IsZero() {
		q.To = now
	}

	if q.Limit <= 0 {
		q.Limit = defaultHistoryLimit
	}
	if q.Limit > maxHistoryLimit {
		q.Limit = maxHistoryLimit
	}

	return nil
}

// addressHistoryFilters builds the shared WHERE conditions for the buckets, count, and
// events queries so the three can never drift. Column references are fixed constants;
// callers supply only values, which squirrel parameterises. A device-ID slice expands
// to an IN clause (empty slice is omitted).
func addressHistoryFilters(q AddressHistoryQuery) sq.And {
	cond := sq.And{sq.Expr("d.deleted_at IS NULL")}

	if len(q.DeviceIDs) > 0 {
		cond = append(cond, sq.Eq{"a.device_id": q.DeviceIDs})
	}
	if !q.From.IsZero() {
		cond = append(cond, sq.GtOrEq{"aev.created_at": q.From})
	}
	if !q.To.IsZero() {
		cond = append(cond, sq.LtOrEq{"aev.created_at": q.To})
	}
	if q.Source != nil {
		cond = append(cond, sq.Eq{"aev.source": *q.Source})
	}
	if q.IsEnabled != nil {
		cond = append(cond, sq.Eq{"aev.is_enabled": *q.IsEnabled})
	}
	if q.IP != nil {
		cond = append(cond, sq.Expr(`a.ip LIKE ? ESCAPE '\'`, "%"+database.EscapeLIKE(*q.IP)+"%"))
	}

	return cond
}

// stateChangeCond keeps only state-change events (creation, enable↔disable
// transitions) by comparing each event's is_enabled with the immediately preceding
// event for the same address. Added as a WHERE condition when IncludeAll is false.
// References the enriched outer query's columns (e.id / e.address_id), which carry
// the same values as the underlying address_events row.
const stateChangeCond = `(
	NOT EXISTS (
		SELECT 1 FROM address_events prev
		WHERE prev.address_id = e.address_id AND prev.id < e.id
	)
	OR e.is_enabled != (
		SELECT prev.is_enabled FROM address_events prev
		WHERE prev.address_id = e.address_id AND prev.id < e.id
		ORDER BY prev.id DESC LIMIT 1
	)
)`

// deviceEventWindow orders and partitions address_events by device so LAG can
// compare each event against the immediately preceding one for the same device,
// across all of that device's addresses.
const deviceEventWindow = `WINDOW w AS (PARTITION BY a.device_id ORDER BY aev.created_at ASC, aev.id ASC)`

func (r *Repository) GetAddressHistory(ctx context.Context, q AddressHistoryQuery) (AddressHistory, error) {
	cond := addressHistoryFilters(q)

	// ── Buckets ──────────────────────────────────────────────────────────
	// active_count = addresses whose last event in the bucket was is_enabled=1.
	//   "Last event" = no later event exists for the same address in the same bucket.
	//   Detected via NOT EXISTS on address_events directly (avoids CTE self-ref).
	// gap_count = addresses that had any is_enabled=0 (expiry) event in the bucket.
	//
	// The strftime format string is a column-level placeholder, so squirrel emits its
	// args before the WHERE args automatically — no manual arg layout required.
	bucketFmt := q.Granularity.StrftimeISO()
	bucketsSQL, bucketArgs, err := sq.
		Select().
		Column("strftime(?, aev.created_at) AS bucket", bucketFmt).
		Column(`COUNT(DISTINCT CASE
			WHEN aev.is_enabled = 1
			 AND NOT EXISTS (
				 SELECT 1 FROM address_events later
				 WHERE later.address_id = aev.address_id
				   AND later.id > aev.id
				   AND strftime(?, later.created_at) = strftime(?, aev.created_at)
			 )
			THEN aev.address_id
		END) AS active_count`, bucketFmt, bucketFmt).
		Column("COUNT(DISTINCT CASE WHEN aev.is_enabled = 0 THEN aev.address_id END) AS gap_count").
		Column("COUNT(*) AS event_count").
		From("address_events aev").
		Join("addresses a ON a.id = aev.address_id").
		Join("devices d ON d.id = a.device_id").
		Where(cond).
		GroupBy("bucket").
		OrderBy("bucket ASC").
		ToSql()
	if err != nil {
		return AddressHistory{}, fmt.Errorf("build history buckets query: %w", err)
	}

	var buckets []AddressEventBucket
	if err := r.db.SelectContext(ctx, &buckets, bucketsSQL, bucketArgs...); err != nil {
		return AddressHistory{}, fmt.Errorf("get history buckets: %w", err)
	}
	if buckets == nil {
		buckets = []AddressEventBucket{}
	}

	// ── Events (enriched + paginated) ────────────────────────────────────
	// The inner query computes, per device (partitioning by device_id so a
	// device's events are compared across all of its addresses, not just one):
	//   - the previous event's timestamp/IP/enabled state (via LAG), used to
	//     derive time_gap_seconds / ip_changed / is_refresh in the outer query
	//   - the device's configured lease TTL, if an auto-expiry rule is enabled
	//
	// Comparisons are computed within the filtered (from/to-bounded) result set:
	// the earliest event for a device in range has nothing to compare against,
	// so it reports a null gap and false change flags — consistent with how the
	// bucket query already treats range edges.
	inner := sq.Select(
		"aev.id", "aev.address_id", "aev.created_at", "a.ip", "aev.is_enabled", "aev.source",
		"a.device_id", "d.name AS device_name",
		"LAG(aev.created_at) OVER w AS prev_created_at",
		"LAG(a.ip) OVER w AS prev_ip",
		"LAG(aev.is_enabled) OVER w AS prev_is_enabled",
		"json_extract(dr.config, '$.ttl_seconds') AS ttl_seconds",
	).
		From("address_events aev").
		Join("addresses a ON a.id = aev.address_id").
		Join("devices d ON d.id = a.device_id").
		LeftJoin("device_rules dr ON dr.device_id = a.device_id AND dr.rule_type = 'device_lease' AND dr.enabled = 1").
		Where(cond).
		Suffix(deviceEventWindow)

	// Shared base for the count and the page query, built on the enriched derived
	// table so both branches see the same filtered, comparison-enriched rows. When
	// IncludeAll is false, restrict to state-change events.
	base := sq.Select().FromSelect(inner, "e")
	if !q.IncludeAll {
		base = base.Where(sq.Expr(stateChangeCond))
	}

	// Count (without cursor).
	countSQL, countArgs, err := base.Column("COUNT(*)").ToSql()
	if err != nil {
		return AddressHistory{}, fmt.Errorf("build history count query: %w", err)
	}
	var totalEvents int
	if err := r.db.GetContext(ctx, &totalEvents, countSQL, countArgs...); err != nil {
		return AddressHistory{}, fmt.Errorf("count history events: %w", err)
	}

	// Events page (cursor + limit).
	eventsB := base.Columns(
		"e.id", "e.created_at", "e.ip", "e.is_enabled", "e.source", "e.device_id", "e.device_name", "e.ttl_seconds",
		"CAST((julianday(e.created_at) - julianday(e.prev_created_at)) * 86400 AS INTEGER) AS time_gap_seconds",
		"CASE WHEN e.prev_ip IS NOT NULL AND e.prev_ip != e.ip THEN 1 ELSE 0 END AS ip_changed",
		`CASE WHEN e.prev_created_at IS NOT NULL
		       AND e.prev_ip = e.ip
		       AND e.prev_is_enabled = e.is_enabled
		      THEN 1 ELSE 0 END AS is_refresh`,
	)
	if q.BeforeID != nil {
		eventsB = eventsB.Where(sq.Lt{"e.id": *q.BeforeID})
	}
	eventsB = eventsB.OrderBy("e.id DESC").Limit(uint64(q.Limit))

	eventsSQL, eventArgs, err := eventsB.ToSql()
	if err != nil {
		return AddressHistory{}, fmt.Errorf("build history events query: %w", err)
	}

	var events []AddressEventView
	if err := r.db.SelectContext(ctx, &events, eventsSQL, eventArgs...); err != nil {
		return AddressHistory{}, fmt.Errorf("get history events: %w", err)
	}
	if events == nil {
		events = []AddressEventView{}
	}

	return AddressHistory{
		Buckets:     buckets,
		Events:      events,
		TotalEvents: totalEvents,
	}, nil
}
