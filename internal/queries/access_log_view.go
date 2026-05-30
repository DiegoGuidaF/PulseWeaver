package queries

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

type AccessLogView struct {
	ID                int64
	ClientIP          string
	Outcome           bool
	DenyReason        *string
	DeviceID          *ids.DeviceID
	DeviceName        *string
	AddressID         *ids.AddressID
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

type AccessLogQuery struct {
	From            time.Time
	To              time.Time
	BeforeID        *int64 // cursor: return rows with id < BeforeID; nil for first page
	ClientIP        *string
	Outcome         *bool
	DenyReason      *string
	DeviceID        *ids.DeviceID
	NetworkPolicyID *ids.NetworkPolicyID
	TargetHost      *string
	CountryCode     *string
	ContinentCode   *string
	Limit           int
}

func NewAccessLogQuery(params httpapi.GetAccessLogParams) AccessLogQuery {
	// Defaults: from = 24h ago, to = now, limit = 50 if not provided.
	now := time.Now().UTC()
	from := now.Add(-24 * time.Hour)
	to := now

	if params.From != nil {
		from = *params.From
	}
	if params.To != nil {
		to = *params.To
	}

	var limit int
	if params.Limit != nil {
		limit = *params.Limit
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	return AccessLogQuery{
		DeviceID:        (*ids.DeviceID)(params.DeviceId),
		Outcome:         params.Outcome,
		DenyReason:      params.DenyReason,
		ClientIP:        params.Ip,
		TargetHost:      params.Host,
		CountryCode:     params.CountryCode,
		ContinentCode:   params.ContinentCode,
		NetworkPolicyID: (*ids.NetworkPolicyID)(params.NetworkPolicyId),
		From:            from,
		To:              to,
		Limit:           limit,
		BeforeID:        params.BeforeId,
	}

}

func (r *Repository) ListAccessLog(ctx context.Context, q AccessLogQuery) ([]AccessLogView, int, error) {
	// Shared filter conditions, applied to both the count and the page query so the
	// two can never drift. Columns are fixed constants (not caller-supplied), so this
	// set is the allowlist; richer per-column operators arrive with PW-24.
	cond := accessLogFilters(q)

	// Total count (no contributor join needed — device filtering uses an EXISTS subquery).
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

	// For display, expose the first contributor's device/address (lowest contributor id).
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
			"c.device_id  AS device_id",
			"c.address_id AS address_id",
			"d.name       AS device_name",
			"ral.headers_json",
			"ral.duration_us",
			"g.country_code",
			"g.country_name",
			"g.continent_code",
			"g.asn",
			"g.asn_org",
			"anpc.policy_id   AS network_policy_id",
			"anpc.policy_name AS network_policy_name",
		).
		From("access_log ral").
		LeftJoin("access_log_contributors c ON c.rowid = (SELECT c2.rowid FROM access_log_contributors c2 WHERE c2.access_log_id = ral.id LIMIT 1)").
		LeftJoin("devices d ON d.id = c.device_id").
		LeftJoin("access_log_geoip g ON g.access_log_id = ral.id").
		LeftJoin("access_log_network_policy_contributors anpc ON anpc.access_log_id = ral.id").
		Where(cond)

	// Cursor pagination: rows with id < BeforeID (nil for the first page).
	if q.BeforeID != nil {
		page = page.Where(sq.Lt{"ral.id": *q.BeforeID})
	}
	page = page.OrderBy("ral.id DESC").Limit(uint64(q.Limit))

	selectSQL, selectArgs, err := page.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build access log query: %w", err)
	}

	var dbRows []dbAccessLogRow
	if err := r.db.SelectContext(ctx, &dbRows, selectSQL, selectArgs...); err != nil {
		return nil, 0, fmt.Errorf("list access log: %w", err)
	}

	rows := make([]AccessLogView, len(dbRows))
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
			DeviceID:          rRow.DeviceID,
			DeviceName:        rRow.DeviceName,
			AddressID:         rRow.AddressID,
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
		}
	}

	if len(rows) == 0 {
		rows = []AccessLogView{}
	}

	return rows, total, nil
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
func (r *Repository) ListAccessLogStatsByCountry(ctx context.Context, from, to time.Time) ([]AccessLogCountryStat, error) {
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

	stats := make([]AccessLogCountryStat, len(rows))
	for i, row := range rows {
		stats[i] = AccessLogCountryStat(row)
	}

	return stats, nil
}

// Page of rows.
type dbAccessLogRow struct {
	ID                int64          `db:"id"`
	ClientIP          string         `db:"client_ip"`
	Outcome           bool           `db:"outcome"`
	DenyReason        *string        `db:"deny_reason"`
	DeviceID          *ids.DeviceID  `db:"device_id"`
	DeviceName        *string        `db:"device_name"`
	AddressID         *ids.AddressID `db:"address_id"`
	CreatedAt         time.Time      `db:"created_at"`
	DurationUs        int64          `db:"duration_us"`
	XFFChain          *string        `db:"xff_chain"`
	TargetHost        *string        `db:"target_host"`
	TargetURI         *string        `db:"target_uri"`
	HTTPMethod        *string        `db:"http_method"`
	HeadersRaw        string         `db:"headers_json"`
	CountryCode       *string        `db:"country_code"`
	CountryName       *string        `db:"country_name"`
	ContinentCode     *string        `db:"continent_code"`
	ASN               *int64         `db:"asn"`
	ASNOrg            *string        `db:"asn_org"`
	NetworkPolicyID   *int64         `db:"network_policy_id"`
	NetworkPolicyName *string        `db:"network_policy_name"`
}

type dbCountryStatsRow struct {
	CountryCode   string `db:"country_code"`
	CountryName   string `db:"country_name"`
	ContinentCode string `db:"continent_code"`
	Total         int64  `db:"total"`
	Allowed       int64  `db:"allowed"`
	Denied        int64  `db:"denied"`
}

// accessLogFilters builds the shared WHERE conditions for the access log list and
// count queries. Column references are fixed constants — callers supply only values,
// which squirrel parameterises. An empty set renders to no WHERE clause.
func accessLogFilters(q AccessLogQuery) sq.And {
	cond := sq.And{}

	if q.DeviceID != nil {
		// Any contributor row for this access_log entry matches the device.
		cond = append(cond, sq.Expr(
			"EXISTS (SELECT 1 FROM access_log_contributors c WHERE c.access_log_id = ral.id AND c.device_id = ?)",
			*q.DeviceID,
		))
	}
	if q.Outcome != nil {
		cond = append(cond, sq.Eq{"ral.outcome": *q.Outcome})
	}
	if q.DenyReason != nil {
		cond = append(cond, sq.Eq{"ral.deny_reason": *q.DenyReason})
	}
	if q.ClientIP != nil {
		// Substring match; wildcards in the input are escaped so they match literally.
		cond = append(cond, sq.Expr(`ral.client_ip LIKE ? ESCAPE '\'`, "%"+database.EscapeLIKE(*q.ClientIP)+"%"))
	}
	if q.TargetHost != nil {
		cond = append(cond, sq.Eq{"ral.target_host": *q.TargetHost})
	}
	if q.CountryCode != nil {
		cond = append(cond, sq.Eq{"g.country_code": *q.CountryCode})
	}
	if q.ContinentCode != nil {
		cond = append(cond, sq.Eq{"g.continent_code": *q.ContinentCode})
	}
	if q.NetworkPolicyID != nil {
		cond = append(cond, sq.Eq{"anpc.policy_id": *q.NetworkPolicyID})
	}
	if !q.From.IsZero() {
		cond = append(cond, sq.GtOrEq{"ral.created_at": q.From})
	}
	if !q.To.IsZero() {
		cond = append(cond, sq.LtOrEq{"ral.created_at": q.To})
	}

	return cond
}
