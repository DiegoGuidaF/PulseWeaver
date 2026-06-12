package queries

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"

	"github.com/DiegoGuidaF/PulseWeaver/internal/dashboard"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/queries/filterx"
)

const (
	defaultSort  = "created_at"
	defaultOrder = "desc"

	// contributorCorrelated is the EXISTS body shared by the device and user
	// relational filters: any contributor row for the parent access_log entry.
	contributorCorrelated = "SELECT 1 FROM access_log_contributors c WHERE c.access_log_id = ral.id"
)

// accessLogRegistry is the column allowlist for the access log list query. SQL
// expressions are fixed here (ADR-007): callers supply only values. The next
// filter-rich views (network policies, devices) adopt this same component.
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
		"continent_code": {
			Expr:     "g.continent_code",
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
		"created_at":   {Expr: "ral.created_at", Kind: filterx.KindTime},
		"client_ip":    {Expr: "ral.client_ip", Kind: filterx.KindString},
		"target_host":  {Expr: "ral.target_host", Kind: filterx.KindString, Nullable: true},
		"http_method":  {Expr: "ral.http_method", Kind: filterx.KindString, Nullable: true},
		"country_code": {Expr: "g.country_code", Kind: filterx.KindString, Nullable: true},
		"deny_reason":  {Expr: "ral.deny_reason", Kind: filterx.KindString, Nullable: true},
		"duration_us":  {Expr: "ral.duration_us", Kind: filterx.KindInt},
		"outcome":      {Expr: "ral.outcome", Kind: filterx.KindInt},
	},
	"ral.id",
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

type AccessLogView struct {
	ID                int64
	ClientIP          string
	Outcome           bool
	DenyReason        *string
	Contributors      []AccessLogContributor
	ContributorCount  int
	CreatedAt         time.Time
	DurationUs        int64
	XFFChain          *string
	TargetHost        *string
	TargetURI         *string
	HTTPMethod        *string
	Headers           map[string][]string
	CountryCode       *string
	CountryName       *string
	ContinentCode     *string
	ASN               *int64
	ASNOrg            *string
	NetworkPolicyID   *int64
	NetworkPolicyName *string
}

// AccessLogQuery is the validated, normalized form of the list request. Sort and
// Order always hold effective values; Cursor is nil on the first page.
type AccessLogQuery struct {
	From      time.Time
	To        time.Time
	Outcome   *bool
	Ambiguous *bool
	Filters   []filterx.Filter
	Sort      string
	Order     string
	Cursor    *filterx.Cursor
	Limit     int
}

func stringsToAny(ss *[]string) []any {
	if ss == nil {
		return nil
	}
	out := make([]any, len(*ss))
	for i, v := range *ss {
		out[i] = v
	}
	return out
}

func idsToAny(idList *[]httpapi.ID) []any {
	if idList == nil {
		return nil
	}
	out := make([]any, len(*idList))
	for i, v := range *idList {
		out[i] = int64(v)
	}
	return out
}

// parseFilter resolves a value column's operator and decides whether it
// contributes a filter. A present operator with no values (and no null check) is
// treated as no filter; null operators apply with no values.
func parseFilter(column string, values []any, opPtr *httpapi.AccessLogFilterOperator) (filterx.Filter, bool, error) {
	op := filterx.OpIn
	if opPtr != nil {
		parsed, err := filterx.ParseOperator(string(*opPtr))
		if err != nil {
			return filterx.Filter{}, false, err
		}
		op = parsed
	}
	nullCheck := op == filterx.OpIsNull || op == filterx.OpNotNull
	if len(values) == 0 && !nullCheck {
		return filterx.Filter{}, false, nil
	}
	return filterx.Filter{Column: column, Op: op, Values: values}, true, nil
}

// NewAccessLogQuery validates and normalizes the request params. It returns an
// error wrapping filterx.ErrInvalidFilter for an unknown operator/column/sort,
// an over-cap value list, or a malformed cursor — the handler maps these to 400.
func NewAccessLogQuery(params httpapi.GetAccessLogParams) (AccessLogQuery, error) {
	now := time.Now().UTC()
	from := now.Add(-24 * time.Hour)
	to := now
	if params.From != nil {
		from = *params.From
	}
	if params.To != nil {
		to = *params.To
	}

	limit := 50
	if params.Limit != nil {
		limit = *params.Limit
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	q := AccessLogQuery{
		From:      from,
		To:        to,
		Limit:     limit,
		Outcome:   params.Outcome,
		Ambiguous: params.Ambiguous,
	}

	valueFilters := []struct {
		column string
		values []any
		op     *httpapi.AccessLogFilterOperator
	}{
		{"client_ip", stringsToAny(params.ClientIp), params.ClientIpOp},
		{"target_host", stringsToAny(params.TargetHost), params.TargetHostOp},
		{"target_uri", stringsToAny(params.TargetUri), params.TargetUriOp},
		{"http_method", stringsToAny(params.HttpMethod), params.HttpMethodOp},
		{"deny_reason", stringsToAny(params.DenyReason), params.DenyReasonOp},
		{"country_code", stringsToAny(params.CountryCode), params.CountryCodeOp},
		{"continent_code", stringsToAny(params.ContinentCode), params.ContinentCodeOp},
		{"device", idsToAny(params.DeviceId), params.DeviceIdOp},
		{"user", idsToAny(params.UserId), params.UserIdOp},
		{"network_policy", idsToAny(params.NetworkPolicyId), params.NetworkPolicyIdOp},
	}
	for _, vf := range valueFilters {
		filter, ok, err := parseFilter(vf.column, vf.values, vf.op)
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

// accessLogConditions assembles the shared WHERE set fed to both the count and
// page query so the two can never drift. Cursor and limit attach to the page
// builder only.
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
	if q.Ambiguous != nil && *q.Ambiguous {
		// Reads the denormalized count directly — no join.
		cond = append(cond, sq.Expr("ral.contributor_count > 1"))
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

func (r *Repository) ListAccessLog(ctx context.Context, q AccessLogQuery) ([]AccessLogView, int, error) {
	if q.Sort == "" {
		q.Sort = defaultSort
	}
	if q.Order == "" {
		q.Order = defaultOrder
	}

	cond, err := accessLogConditions(q)
	if err != nil {
		return nil, 0, err
	}

	// Total count. The geoip and network-policy joins are 1:1 (PK on access_log_id),
	// so they cannot inflate the count; device/user filters use EXISTS subqueries.
	var total int
	countSQL, countArgs, err := sq.
		Select("COUNT(*)").
		From("access_log ral").
		LeftJoin("access_log_geoip g ON g.access_log_id = ral.id").
		LeftJoin("access_log_network_policy_contributors anpc ON anpc.access_log_id = ral.id").
		Where(cond).
		ToSql()
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
		Select(
			"ral.id",
			"ral.created_at",
			"ral.outcome",
			"ral.deny_reason",
			"ral.client_ip",
			"ral.xff_chain",
			"ral.target_host",
			"ral.target_uri",
			"ral.http_method",
			"ral.headers_json",
			"ral.duration_us",
			"ral.contributor_count",
			"g.country_code",
			"g.country_name",
			"g.continent_code",
			"g.asn",
			"g.asn_org",
			"anpc.policy_id   AS network_policy_id",
			"anpc.policy_name AS network_policy_name",
		).
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
		var headers map[string][]string
		if err := json.Unmarshal([]byte(rRow.HeadersRaw), &headers); err != nil {
			// Malformed JSON should not break the endpoint; fall back to empty map.
			headers = map[string][]string{}
		}

		rows[i] = AccessLogView{
			ID:                rRow.ID,
			ClientIP:          rRow.ClientIP,
			Outcome:           rRow.Outcome,
			DenyReason:        rRow.DenyReason,
			ContributorCount:  rRow.ContributorCount,
			CreatedAt:         rRow.CreatedAt,
			DurationUs:        rRow.DurationUs,
			XFFChain:          rRow.XFFChain,
			TargetHost:        rRow.TargetHost,
			TargetURI:         rRow.TargetURI,
			HTTPMethod:        rRow.HTTPMethod,
			Headers:           headers,
			CountryCode:       rRow.CountryCode,
			CountryName:       rRow.CountryName,
			ContinentCode:     rRow.ContinentCode,
			ASN:               rRow.ASN,
			ASNOrg:            rRow.ASNOrg,
			NetworkPolicyID:   rRow.NetworkPolicyID,
			NetworkPolicyName: rRow.NetworkPolicyName,
			Contributors:      []AccessLogContributor{},
		}
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
		deviceID := cr.DeviceID
		deviceName := cr.DeviceName
		userID := cr.UserID
		userName := cr.UserName
		addressID := cr.AddressID
		result[cr.AccessLogID] = append(result[cr.AccessLogID], AccessLogContributor{
			DeviceID:   &deviceID,
			DeviceName: &deviceName,
			UserID:     &userID,
			UserName:   &userName,
			AddressID:  &addressID,
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
	case "country_code":
		return strPtrValue(row.CountryCode)
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

// AccessLogCountryStat holds aggregated request counts for a single country.
type AccessLogCountryStat struct {
	CountryCode   string
	CountryName   string
	ContinentCode string
	Total         int64
	Allowed       int64
	Denied        int64
}

// ListAccessLogStatsByCountry returns request counts grouped by country for all rows
// within the [from, to] time window. Only rows with GeoIP data are included.
//
// Dispatches on dashboard.RawWindowThreshold like every other traffic widget,
// so the map/country tables answer from the same source as the stat cards and
// charts for a given window.
func (r *Repository) ListAccessLogStatsByCountry(ctx context.Context, from, to time.Time) ([]AccessLogCountryStat, error) {
	if to.Sub(from) <= dashboard.RawWindowThreshold {
		return r.listRawAccessLogStatsByCountry(ctx, from, to)
	}
	return r.listAggregateAccessLogStatsByCountry(ctx, from, to)
}

func (r *Repository) listRawAccessLogStatsByCountry(ctx context.Context, from, to time.Time) ([]AccessLogCountryStat, error) {
	const query = `
		SELECT
			g.country_code,
			COALESCE(g.country_name, '')  AS country_name,
			COALESCE(g.continent_code, '') AS continent_code,
			COUNT(*) AS total,
			SUM(CASE WHEN ral.outcome = 1 THEN 1 ELSE 0 END) AS allowed,
			SUM(CASE WHEN ral.outcome = 0 THEN 1 ELSE 0 END) AS denied
		FROM access_log_geoip g
		JOIN access_log ral ON ral.id = g.access_log_id
		WHERE ral.created_at >= ? AND ral.created_at <= ?
		GROUP BY g.country_code, g.country_name, g.continent_code
		ORDER BY total DESC
	`

	var rows []dbCountryStatsRow
	if err := r.db.SelectContext(ctx, &rows, query, from, to); err != nil {
		return nil, fmt.Errorf("list access log stats by country: %w", err)
	}

	return countryStatsFromRows(rows), nil
}

// listAggregateAccessLogStatsByCountry answers from hourly_traffic_aggregates.
// Buckets without country attribution (empty country_code: no GeoIP at rollup
// time, or rolled up before country columns existed) are excluded, matching
// the raw path's inner join on access_log_geoip.
func (r *Repository) listAggregateAccessLogStatsByCountry(ctx context.Context, from, to time.Time) ([]AccessLogCountryStat, error) {
	const query = `
		SELECT
			country_code,
			country_name,
			continent_code,
			SUM(request_count) AS total,
			SUM(CASE WHEN outcome = 1 THEN request_count ELSE 0 END) AS allowed,
			SUM(CASE WHEN outcome = 0 THEN request_count ELSE 0 END) AS denied
		FROM hourly_traffic_aggregates
		WHERE country_code != ''
		  AND bucket_at >= ? AND bucket_at < ?
		GROUP BY country_code, country_name, continent_code
		ORDER BY total DESC
	`

	var rows []dbCountryStatsRow
	if err := r.db.SelectContext(ctx, &rows, query, from.UTC(), to.UTC()); err != nil {
		return nil, fmt.Errorf("list aggregate access log stats by country: %w", err)
	}

	return countryStatsFromRows(rows), nil
}

func countryStatsFromRows(rows []dbCountryStatsRow) []AccessLogCountryStat {
	stats := make([]AccessLogCountryStat, len(rows))
	for i, row := range rows {
		stats[i] = AccessLogCountryStat(row)
	}
	return stats
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
	XFFChain          *string   `db:"xff_chain"`
	TargetHost        *string   `db:"target_host"`
	TargetURI         *string   `db:"target_uri"`
	HTTPMethod        *string   `db:"http_method"`
	HeadersRaw        string    `db:"headers_json"`
	CountryCode       *string   `db:"country_code"`
	CountryName       *string   `db:"country_name"`
	ContinentCode     *string   `db:"continent_code"`
	ASN               *int64    `db:"asn"`
	ASNOrg            *string   `db:"asn_org"`
	NetworkPolicyID   *int64    `db:"network_policy_id"`
	NetworkPolicyName *string   `db:"network_policy_name"`
}

type dbContributorRow struct {
	AccessLogID int64         `db:"access_log_id"`
	DeviceID    ids.DeviceID  `db:"device_id"`
	DeviceName  string        `db:"device_name"`
	UserID      ids.UserID    `db:"user_id"`
	UserName    string        `db:"user_name"`
	AddressID   ids.AddressID `db:"address_id"`
}

type dbCountryStatsRow struct {
	CountryCode   string `db:"country_code"`
	CountryName   string `db:"country_name"`
	ContinentCode string `db:"continent_code"`
	Total         int64  `db:"total"`
	Allowed       int64  `db:"allowed"`
	Denied        int64  `db:"denied"`
}
