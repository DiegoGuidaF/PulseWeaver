package accesslog

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	sq "github.com/Masterminds/squirrel"

	"github.com/DiegoGuidaF/PulseWeaver/internal/filterx"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

const (
	defaultSort  = "created_at"
	defaultOrder = "desc"

	// defaultWindow is how far back a request with no explicit from/to looks.
	defaultWindow = 24 * time.Hour

	// defaultPageSize applies when the caller names no limit; maxPageSize caps
	// what it can ask for, since every row carries its contributors.
	defaultPageSize = 50
	maxPageSize     = 200

	// contributorCorrelated is the EXISTS body shared by the device and user
	// relational filters: any contributor row for the parent access_log entry.
	contributorCorrelated = "SELECT 1 FROM access_log_contributors c WHERE c.access_log_id = ral.id"
)

// ErrEntryNotFound reports that no access_log row carries the requested id.
// Entries are pruned by retention (DeleteOlderThan), so a link to an old
// request resolving to this is an ordinary outcome, not a failure.
var ErrEntryNotFound = errors.New("access log entry not found")

// accessLogRegistry is the column allowlist for the access log queries. SQL
// expressions are fixed here (ADR-007): callers supply only values. The list,
// its count, and the histogram all filter through this one registry.
var accessLogRegistry = filterx.NewRegistry(
	map[string]filterx.ColumnSpec{
		"client_ip": {
			Expr: "ral.client_ip",
			Ops:  []filterx.Operator{filterx.OpIn, filterx.OpNotIn, filterx.OpContains, filterx.OpNotContains},
		},
		"target_host": {
			Expr:     "ral.target_host",
			Nullable: true,
			Ops:      []filterx.Operator{filterx.OpIn, filterx.OpNotIn, filterx.OpContains, filterx.OpNotContains, filterx.OpIsNull, filterx.OpNotNull},
		},
		"target_uri": {
			Expr:     "ral.target_uri",
			Nullable: true,
			Ops:      []filterx.Operator{filterx.OpIn, filterx.OpNotIn, filterx.OpContains, filterx.OpNotContains, filterx.OpIsNull, filterx.OpNotNull},
		},
		"http_method": {
			Expr:     "ral.http_method",
			Nullable: true,
			Ops:      []filterx.Operator{filterx.OpIn, filterx.OpNotIn},
		},
		"deny_reason": {
			Expr:     "ral.deny_reason",
			Nullable: true,
			Ops:      []filterx.Operator{filterx.OpIn, filterx.OpNotIn, filterx.OpIsNull, filterx.OpNotNull},
		},
		"country_code": {
			Expr:     "g.country_code",
			Nullable: true,
			Ops:      []filterx.Operator{filterx.OpIn, filterx.OpNotIn, filterx.OpIsNull, filterx.OpNotNull},
		},
		"network_policy": {
			Expr:     "anpc.policy_id",
			Nullable: true,
			Ops:      []filterx.Operator{filterx.OpIn, filterx.OpNotIn, filterx.OpIsNull, filterx.OpNotNull},
		},
		"device": {
			Rel: &filterx.Relational{Correlated: contributorCorrelated, ValueCol: "c.device_id"},
			Ops: []filterx.Operator{filterx.OpIn, filterx.OpNotIn, filterx.OpIsNull, filterx.OpNotNull},
		},
		"user": {
			Rel: &filterx.Relational{Correlated: contributorCorrelated, ValueCol: "c.user_id"},
			Ops: []filterx.Operator{filterx.OpIn, filterx.OpNotIn},
		},
	},
	map[string]filterx.SortSpec{
		"created_at":  {Expr: "ral.created_at", Kind: filterx.KindTime},
		"client_ip":   {Expr: "ral.client_ip", Kind: filterx.KindString},
		"target_host": {Expr: "ral.target_host", Kind: filterx.KindString, Nullable: true},
		"http_method": {Expr: "ral.http_method", Kind: filterx.KindString, Nullable: true},
		"deny_reason": {Expr: "ral.deny_reason", Kind: filterx.KindString, Nullable: true},
		"duration_us": {Expr: "ral.duration_us", Kind: filterx.KindInt},
		"outcome":     {Expr: "ral.outcome", Kind: filterx.KindInt},
	},
	"ral.id",
)

// accessLogRowColumns are the columns the table scans. Detail-only columns —
// the headers blob, the forwarded-for chain, the full geolocation — are read by
// GetAccessLogEntry alone, so paging the list never pays for them.
var accessLogRowColumns = []string{
	"ral.id",
	"ral.created_at",
	"ral.outcome",
	"ral.deny_reason",
	"ral.client_ip",
	"ral.target_host",
	"ral.target_uri",
	"ral.http_method",
	"ral.duration_us",
	"ral.contributor_count",
	"g.country_code",
	"anpc.policy_id   AS network_policy_id",
	"anpc.policy_name AS network_policy_name",
}

// accessLogDetailColumns extend the row columns rather than restating them, so
// the two shapes cannot report different values for a field they share.
var accessLogDetailColumns = append(slices.Clone(accessLogRowColumns),
	"ral.xff_chain",
	"ral.headers_json",
	"g.country_name",
	"g.continent_code",
	"g.asn",
	"g.asn_org",
)

// AccessLogContributor is one device/user/address a request's client IP resolved
// to. Fields are always populated (contributor rows are fully constrained) but
// carried as pointers to mirror the API shape.
type AccessLogContributor struct {
	DeviceID   *ids.DeviceID
	DeviceName *string
	UserID     *ids.UserID
	UserName   *string
	AddressID  *ids.AddressID
}

// AccessLogView is one row of the access-log table.
type AccessLogView struct {
	ID                int64
	ClientIP          string
	Outcome           bool
	DenyReason        *string
	Contributors      []AccessLogContributor
	ContributorCount  int
	CreatedAt         time.Time
	DurationUs        int64
	TargetHost        *string
	TargetURI         *string
	HTTPMethod        *string
	CountryCode       *string
	NetworkPolicyID   *int64
	NetworkPolicyName *string
}

// AccessLogDetailView is the whole record of one request: the table row plus
// the fields only the detail view shows.
type AccessLogDetailView struct {
	AccessLogView
	XFFChain      *string
	Headers       map[string][]string
	CountryName   *string
	ContinentCode *string
	ASN           *int64
	ASNOrg        *string
}

// AccessLogQuery is the validated, normalized form of the list request. Sort and
// Order always hold effective values; Cursor is nil on the first page. The
// histogram shares Filters, Outcome and the window, and leaves the rest zero.
type AccessLogQuery struct {
	From    time.Time
	To      time.Time
	Outcome *bool
	Filters []filterx.Filter
	Sort    string
	Order   string
	Cursor  *filterx.Cursor
	Limit   int
}

// accessLogFilterParams collects the fields the list and histogram OpenAPI
// params structs share, so the filter parsing underneath is written once.
type accessLogFilterParams struct {
	From              *time.Time
	To                *time.Time
	Outcome           *bool
	ClientIP          *[]string
	ClientIPOp        *httpapi.AccessLogFilterOperator
	TargetHost        *[]string
	TargetHostOp      *httpapi.AccessLogFilterOperator
	TargetURI         *[]string
	TargetURIOp       *httpapi.AccessLogFilterOperator
	HTTPMethod        *[]string
	HTTPMethodOp      *httpapi.AccessLogFilterOperator
	DenyReason        *[]httpapi.PolicyDenyReason
	DenyReasonOp      *httpapi.AccessLogFilterOperator
	CountryCode       *[]string
	CountryCodeOp     *httpapi.AccessLogFilterOperator
	DeviceID          *[]httpapi.ID
	DeviceIDOp        *httpapi.AccessLogFilterOperator
	UserID            *[]httpapi.ID
	UserIDOp          *httpapi.AccessLogFilterOperator
	NetworkPolicyID   *[]httpapi.ID
	NetworkPolicyIDOp *httpapi.AccessLogFilterOperator
}

// newAccessLogQuery validates and normalizes the filter/window portion shared by
// the list and histogram endpoints. It returns an error wrapping
// filterx.ErrInvalidFilter for an unknown operator/column or an over-cap value
// list — the handler maps these to 400.
func newAccessLogQuery(p accessLogFilterParams) (AccessLogQuery, error) {
	now := time.Now().UTC()
	q := AccessLogQuery{
		From:    now.Add(-defaultWindow),
		To:      now,
		Outcome: p.Outcome,
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
		op     *httpapi.AccessLogFilterOperator
	}{
		{"client_ip", filterx.StringValues(p.ClientIP), p.ClientIPOp},
		{"target_host", filterx.StringValues(p.TargetHost), p.TargetHostOp},
		{"target_uri", filterx.StringValues(p.TargetURI), p.TargetURIOp},
		{"http_method", filterx.StringValues(p.HTTPMethod), p.HTTPMethodOp},
		{"deny_reason", filterx.StringValues(p.DenyReason), p.DenyReasonOp},
		{"country_code", filterx.StringValues(p.CountryCode), p.CountryCodeOp},
		{"device", filterx.Int64Values(p.DeviceID), p.DeviceIDOp},
		{"user", filterx.Int64Values(p.UserID), p.UserIDOp},
		{"network_policy", filterx.Int64Values(p.NetworkPolicyID), p.NetworkPolicyIDOp},
	}
	for _, vf := range valueFilters {
		filter, ok, err := filterx.ParseFilter(vf.column, vf.values, vf.op)
		if err != nil {
			return AccessLogQuery{}, err
		}
		if !ok {
			continue
		}
		if err := accessLogRegistry.Validate(filter); err != nil {
			return AccessLogQuery{}, err
		}
		q.Filters = append(q.Filters, filter)
	}

	return q, nil
}

// NewAccessLogQuery validates and normalizes GET /access-log params, resolving
// the page's sort, order, cursor and limit on top of the shared filter set.
func NewAccessLogQuery(params httpapi.GetAccessLogParams) (AccessLogQuery, error) {
	q, err := newAccessLogQuery(accessLogFilterParams{
		From: params.From, To: params.To, Outcome: params.Outcome,
		ClientIP: params.ClientIp, ClientIPOp: params.ClientIpOp,
		TargetHost: params.TargetHost, TargetHostOp: params.TargetHostOp,
		TargetURI: params.TargetUri, TargetURIOp: params.TargetUriOp,
		HTTPMethod: params.HttpMethod, HTTPMethodOp: params.HttpMethodOp,
		DenyReason: params.DenyReason, DenyReasonOp: params.DenyReasonOp,
		CountryCode: params.CountryCode, CountryCodeOp: params.CountryCodeOp,
		DeviceID: params.DeviceId, DeviceIDOp: params.DeviceIdOp,
		UserID: params.UserId, UserIDOp: params.UserIdOp,
		NetworkPolicyID: params.NetworkPolicyId, NetworkPolicyIDOp: params.NetworkPolicyIdOp,
	})
	if err != nil {
		return AccessLogQuery{}, err
	}

	limit := defaultPageSize
	if params.Limit != nil {
		limit = *params.Limit
	}
	if limit <= 0 {
		limit = defaultPageSize
	}
	if limit > maxPageSize {
		limit = maxPageSize
	}
	q.Limit = limit

	// A cursor is authoritative for sort/order — it embeds the sort it was issued
	// under. Otherwise resolve from params, falling back to the defaults.
	if params.Cursor != nil && *params.Cursor != "" {
		cur, err := accessLogRegistry.DecodeCursor(*params.Cursor)
		if err != nil {
			return AccessLogQuery{}, err
		}
		q.Cursor = &cur
		q.Sort = cur.Sort
		q.Order = cur.Order
	} else {
		sort := defaultSort
		order := defaultOrder
		if params.Sort != nil {
			sort = string(*params.Sort)
		}
		if params.Order != nil {
			order = string(*params.Order)
		}
		if _, err := accessLogRegistry.OrderBy(sort, order); err != nil {
			return AccessLogQuery{}, err
		}
		q.Sort = sort
		q.Order = order
	}

	return q, nil
}

// NewAccessLogHistogramQuery validates and normalizes GET /access-log/histogram
// params. A histogram has no page and no ordering, so sort, order, limit and
// cursor are absent by design.
func NewAccessLogHistogramQuery(params httpapi.GetAccessLogHistogramParams) (AccessLogQuery, error) {
	return newAccessLogQuery(accessLogFilterParams{
		From: params.From, To: params.To, Outcome: params.Outcome,
		ClientIP: params.ClientIp, ClientIPOp: params.ClientIpOp,
		TargetHost: params.TargetHost, TargetHostOp: params.TargetHostOp,
		TargetURI: params.TargetUri, TargetURIOp: params.TargetUriOp,
		HTTPMethod: params.HttpMethod, HTTPMethodOp: params.HttpMethodOp,
		DenyReason: params.DenyReason, DenyReasonOp: params.DenyReasonOp,
		CountryCode: params.CountryCode, CountryCodeOp: params.CountryCodeOp,
		DeviceID: params.DeviceId, DeviceIDOp: params.DeviceIdOp,
		UserID: params.UserId, UserIDOp: params.UserIdOp,
		NetworkPolicyID: params.NetworkPolicyId, NetworkPolicyIDOp: params.NetworkPolicyIdOp,
	})
}

// accessLogConditions assembles the shared WHERE set fed to the count, page and
// histogram queries so none of them can drift. Cursor and limit attach to the
// page builder only.
func accessLogConditions(q AccessLogQuery) (sq.And, error) {
	cond := sq.And{}
	if !q.From.IsZero() {
		cond = append(cond, sq.GtOrEq{"ral.created_at": q.From})
	}
	if !q.To.IsZero() {
		cond = append(cond, sq.LtOrEq{"ral.created_at": q.To})
	}
	if q.Outcome != nil {
		cond = append(cond, sq.Eq{"ral.outcome": *q.Outcome})
	}
	for _, f := range q.Filters {
		c, err := accessLogRegistry.Condition(f)
		if err != nil {
			return nil, err
		}
		cond = append(cond, c)
	}
	return cond, nil
}

// accessLogFilterJoins reports which 1:1 child tables the query's filters reference,
// so an aggregate query can join them only when a WHERE term depends on their columns.
func accessLogFilterJoins(q AccessLogQuery) (geoip, policy bool) {
	for _, f := range q.Filters {
		switch f.Column {
		case "country_code":
			geoip = true
		case "network_policy":
			policy = true
		}
	}
	return geoip, policy
}

func (r *Repository) ListAccessLog(ctx context.Context, q AccessLogQuery) ([]AccessLogView, int, error) {
	cond, err := accessLogConditions(q)
	if err != nil {
		return nil, 0, err
	}

	// Total count. The geoip and network-policy joins are PK-keyed 1:1 children, so
	// they can never change COUNT — include them only when a filter constrains them.
	// Without that filter they would scan ~one PK lookup per matched row for nothing
	// (an order-of-magnitude cost on a full-table window). Device/user filters use
	// EXISTS subqueries and need no join.
	countBuilder := accessLogFilteredFrom(sq.Select("COUNT(*)"), q).Where(cond)
	var total int
	countSQL, countArgs, err := countBuilder.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build access log count query: %w", err)
	}
	if err := r.db.GetContext(ctx, &total, countSQL, countArgs...); err != nil {
		return nil, 0, fmt.Errorf("count access log: %w", err)
	}

	orderBy, err := accessLogRegistry.OrderBy(q.Sort, q.Order)
	if err != nil {
		return nil, 0, fmt.Errorf("build access log order: %w", err)
	}

	// One row per entry: contributors are fetched separately and assembled in Go,
	// never via a fan-out join (which would break LIMIT, the keyset cursor, and COUNT).
	page := sq.
		Select(accessLogRowColumns...).
		From("access_log ral").
		LeftJoin("access_log_geoip g ON g.access_log_id = ral.id").
		LeftJoin("access_log_network_policy_contributors anpc ON anpc.access_log_id = ral.id").
		Where(cond)

	if q.Cursor != nil {
		pred, err := accessLogRegistry.Keyset(*q.Cursor)
		if err != nil {
			return nil, 0, fmt.Errorf("build access log cursor: %w", err)
		}
		page = page.Where(pred)
	}
	page = page.OrderBy(orderBy).Limit(uint64(q.Limit))

	selectSQL, selectArgs, err := page.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build access log query: %w", err)
	}

	var dbRows []dbAccessLogRow
	if err := r.db.SelectContext(ctx, &dbRows, selectSQL, selectArgs...); err != nil {
		return nil, 0, fmt.Errorf("list access log: %w", err)
	}

	rows := make([]AccessLogView, len(dbRows))
	pageIDs := make([]int64, len(dbRows))
	for i, rRow := range dbRows {
		rows[i] = rRow.toView()
		pageIDs[i] = rRow.ID
	}

	contributorsByLog, err := r.fetchAccessLogContributors(ctx, pageIDs)
	if err != nil {
		return nil, 0, err
	}
	for i := range rows {
		if c := contributorsByLog[rows[i].ID]; c != nil {
			rows[i].Contributors = c
		}
	}

	return rows, total, nil
}

// GetAccessLogEntry returns the full record of one request. It reports
// ErrEntryNotFound when the id is unknown, which retention pruning makes an
// ordinary answer for an old link rather than a failure.
func (r *Repository) GetAccessLogEntry(ctx context.Context, id int64) (AccessLogDetailView, error) {
	query, args, err := sq.
		Select(accessLogDetailColumns...).
		From("access_log ral").
		LeftJoin("access_log_geoip g ON g.access_log_id = ral.id").
		LeftJoin("access_log_network_policy_contributors anpc ON anpc.access_log_id = ral.id").
		Where(sq.Eq{"ral.id": id}).
		ToSql()
	if err != nil {
		return AccessLogDetailView{}, fmt.Errorf("build access log entry query: %w", err)
	}

	var row dbAccessLogDetailRow
	if err := r.db.GetContext(ctx, &row, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AccessLogDetailView{}, ErrEntryNotFound
		}
		return AccessLogDetailView{}, fmt.Errorf("get access log entry: %w", err)
	}

	detail := AccessLogDetailView{
		AccessLogView: row.toView(),
		XFFChain:      row.XFFChain,
		CountryName:   row.CountryName,
		ContinentCode: row.ContinentCode,
		ASN:           row.ASN,
		ASNOrg:        row.ASNOrg,
	}
	if err := json.Unmarshal([]byte(row.HeadersRaw), &detail.Headers); err != nil {
		// The row is already persisted, so failing the read would only deny the
		// caller the rest of a record it can still use. Report the bad blob and
		// serve the entry without headers.
		slog.Default().WarnContext(ctx, "access log entry has malformed headers_json",
			slog.Int64("access_log_id", row.ID), slog.Any(logging.AttrKeyError, err))
		detail.Headers = map[string][]string{}
	}

	contributorsByLog, err := r.fetchAccessLogContributors(ctx, []int64{row.ID})
	if err != nil {
		return AccessLogDetailView{}, err
	}
	if c := contributorsByLog[row.ID]; c != nil {
		detail.Contributors = c
	}

	return detail, nil
}

// accessLogFilteredFrom attaches the FROM and only the 1:1 child joins the
// query's filters actually reference. Aggregates (COUNT, the histogram) share
// it; the row-returning queries always join both, since they project from them.
func accessLogFilteredFrom(b sq.SelectBuilder, q AccessLogQuery) sq.SelectBuilder {
	geoipFilter, policyFilter := accessLogFilterJoins(q)
	b = b.From("access_log ral")
	if geoipFilter {
		b = b.LeftJoin("access_log_geoip g ON g.access_log_id = ral.id")
	}
	if policyFilter {
		b = b.LeftJoin("access_log_network_policy_contributors anpc ON anpc.access_log_id = ral.id")
	}
	return b
}

// fetchAccessLogContributors loads every contributor for the given page of
// access_log ids in one bounded query (IN over ≤ limit ids), assembled into a
// map keyed by access_log_id. One query per relationship, assembled in Go.
func (r *Repository) fetchAccessLogContributors(ctx context.Context, logIDs []int64) (map[int64][]AccessLogContributor, error) {
	result := make(map[int64][]AccessLogContributor, len(logIDs))
	if len(logIDs) == 0 {
		return result, nil
	}

	query, args, err := sq.
		Select(
			"c.access_log_id",
			"c.device_id",
			"d.name AS device_name",
			"c.user_id",
			"u.display_name AS user_name",
			"c.address_id",
		).
		From("access_log_contributors c").
		Join("devices d ON d.id = c.device_id").
		Join("users u ON u.id = c.user_id").
		Where(sq.Eq{"c.access_log_id": logIDs}).
		OrderBy("c.access_log_id", "d.name").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build access log contributors query: %w", err)
	}

	var dbRows []dbContributorRow
	if err := r.db.SelectContext(ctx, &dbRows, query, args...); err != nil {
		return nil, fmt.Errorf("list access log contributors: %w", err)
	}

	for _, cr := range dbRows {
		result[cr.AccessLogID] = append(result[cr.AccessLogID], AccessLogContributor{
			DeviceID:   &cr.DeviceID,
			DeviceName: &cr.DeviceName,
			UserID:     &cr.UserID,
			UserName:   &cr.UserName,
			AddressID:  &cr.AddressID,
		})
	}

	return result, nil
}

// accessLogSortValue returns the value of the active sort column for a row, used
// to mint the next-page cursor. The type matches the column's SortSpec kind so
// the cursor round-trips it correctly.
func accessLogSortValue(row AccessLogView, sortKey string) any {
	switch sortKey {
	case "client_ip":
		return row.ClientIP
	case "target_host":
		return strPtrValue(row.TargetHost)
	case "http_method":
		return strPtrValue(row.HTTPMethod)
	case "deny_reason":
		return strPtrValue(row.DenyReason)
	case "duration_us":
		return row.DurationUs
	case "outcome":
		if row.Outcome {
			return int64(1)
		}
		return int64(0)
	default:
		return row.CreatedAt
	}
}

func strPtrValue(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

// Page of rows.
type dbAccessLogRow struct {
	ID                int64     `db:"id"`
	ClientIP          string    `db:"client_ip"`
	Outcome           bool      `db:"outcome"`
	DenyReason        *string   `db:"deny_reason"`
	ContributorCount  int       `db:"contributor_count"`
	CreatedAt         time.Time `db:"created_at"`
	DurationUs        int64     `db:"duration_us"`
	TargetHost        *string   `db:"target_host"`
	TargetURI         *string   `db:"target_uri"`
	HTTPMethod        *string   `db:"http_method"`
	CountryCode       *string   `db:"country_code"`
	NetworkPolicyID   *int64    `db:"network_policy_id"`
	NetworkPolicyName *string   `db:"network_policy_name"`
}

func (r dbAccessLogRow) toView() AccessLogView {
	return AccessLogView{
		ID:                r.ID,
		ClientIP:          r.ClientIP,
		Outcome:           r.Outcome,
		DenyReason:        r.DenyReason,
		ContributorCount:  r.ContributorCount,
		CreatedAt:         r.CreatedAt,
		DurationUs:        r.DurationUs,
		TargetHost:        r.TargetHost,
		TargetURI:         r.TargetURI,
		HTTPMethod:        r.HTTPMethod,
		CountryCode:       r.CountryCode,
		NetworkPolicyID:   r.NetworkPolicyID,
		NetworkPolicyName: r.NetworkPolicyName,
		Contributors:      []AccessLogContributor{},
	}
}

type dbAccessLogDetailRow struct {
	dbAccessLogRow
	XFFChain      *string `db:"xff_chain"`
	HeadersRaw    string  `db:"headers_json"`
	CountryName   *string `db:"country_name"`
	ContinentCode *string `db:"continent_code"`
	ASN           *int64  `db:"asn"`
	ASNOrg        *string `db:"asn_org"`
}

type dbContributorRow struct {
	AccessLogID int64         `db:"access_log_id"`
	DeviceID    ids.DeviceID  `db:"device_id"`
	DeviceName  string        `db:"device_name"`
	UserID      ids.UserID    `db:"user_id"`
	UserName    string        `db:"user_name"`
	AddressID   ids.AddressID `db:"address_id"`
}
