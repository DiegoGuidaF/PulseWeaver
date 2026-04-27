package queries

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
)

type AccessLogView struct {
	ID            int64
	ClientIP      string
	Outcome       bool
	DenyReason    *string
	DeviceID      *device.DeviceID
	DeviceName    *string
	AddressID     *device.AddressID
	CreatedAt     time.Time
	DurationUs    int64
	XFFChain      *string
	TargetHost    *string
	TargetURI     *string
	HTTPMethod    *string
	Headers       map[string][]string
	CountryCode   *string
	CountryName   *string
	ContinentCode *string
	ASN           *int64
	ASNOrg        *string
}

type AccessLogQuery struct {
	From          time.Time
	To            time.Time
	BeforeID      *int64 // cursor: return rows with id < BeforeID; nil for first page
	ClientIP      *string
	Outcome       *bool
	DenyReason    *string
	DeviceID      *device.DeviceID
	TargetHost    *string
	CountryCode   *string
	ContinentCode *string
	Limit         int
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
		DeviceID:      (*device.DeviceID)(params.DeviceId),
		Outcome:       params.Outcome,
		DenyReason:    params.DenyReason,
		ClientIP:      params.Ip,
		TargetHost:    params.Host,
		CountryCode:   params.CountryCode,
		ContinentCode: params.ContinentCode,
		From:          from,
		To:            to,
		Limit:         limit,
		BeforeID:      params.BeforeId,
	}

}

func (r *Repository) ListAccessLog(ctx context.Context, q AccessLogQuery) ([]AccessLogView, int, error) {

	whereFilters := []string{"1=1"}
	var countArgs []any

	if q.DeviceID != nil {
		// Filter: any contributor row for this access_log entry matches the device.
		whereFilters = append(whereFilters, "EXISTS (SELECT 1 FROM access_log_contributors c WHERE c.access_log_id = ral.id AND c.device_id = ?)")
		countArgs = append(countArgs, *q.DeviceID)
	}

	if q.Outcome != nil {
		whereFilters = append(whereFilters, "ral.outcome = ?")
		countArgs = append(countArgs, *q.Outcome)
	}

	if q.DenyReason != nil {
		whereFilters = append(whereFilters, "ral.deny_reason = ?")
		countArgs = append(countArgs, *q.DenyReason)
	}

	if q.ClientIP != nil {
		whereFilters = append(whereFilters, "ral.client_ip LIKE ?")
		countArgs = append(countArgs, "%"+*q.ClientIP+"%")
	}

	if q.TargetHost != nil {
		whereFilters = append(whereFilters, "ral.target_host = ?")
		countArgs = append(countArgs, *q.TargetHost)
	}

	if q.CountryCode != nil {
		whereFilters = append(whereFilters, "g.country_code = ?")
		countArgs = append(countArgs, *q.CountryCode)
	}

	if q.ContinentCode != nil {
		whereFilters = append(whereFilters, "g.continent_code = ?")
		countArgs = append(countArgs, *q.ContinentCode)
	}

	if !q.From.IsZero() {
		whereFilters = append(whereFilters, "ral.created_at >= ?")
		countArgs = append(countArgs, q.From)
	}

	if !q.To.IsZero() {
		whereFilters = append(whereFilters, "ral.created_at <= ?")
		countArgs = append(countArgs, q.To)
	}

	// Total count (no contributor join needed for count — filtering uses EXISTS subquery).
	var total int
	countQuery := `
		SELECT COUNT(*) FROM access_log ral
		LEFT JOIN access_log_geoip g ON g.access_log_id = ral.id
	` + buildWhere(whereFilters)
	if err := r.db.GetContext(ctx, &total, countQuery, countArgs...); err != nil {
		return nil, 0, fmt.Errorf("count access log: %w", err)
	}

	selectArgs := countArgs

	// Append Cursor for pagination: rows with id < BeforeID
	if q.BeforeID != nil {
		whereFilters = append(whereFilters, "ral.id < ?")
		selectArgs = append(selectArgs, *q.BeforeID)
	}
	selectArgs = append(selectArgs, q.Limit)

	var dbRows []dbAccessLogRow
	// For display, expose the first contributor's device/address (lowest contributor id).
	selectQuery := `
		SELECT
			ral.id,
			ral.created_at,
			ral.outcome,
			ral.deny_reason,
			ral.client_ip,
			ral.xff_chain,
			ral.target_host,
			ral.target_uri,
			ral.http_method,
			c.device_id  AS device_id,
			c.address_id AS address_id,
			d.name       AS device_name,
			ral.headers_json,
			ral.duration_us,
			g.country_code,
			g.country_name,
			g.continent_code,
			g.asn,
			g.asn_org
		FROM access_log ral
		LEFT JOIN (
			SELECT access_log_id, min(user_id) AS first_user_id FROM access_log_contributors GROUP BY access_log_id
		) c_first ON c_first.access_log_id = ral.id
		LEFT JOIN access_log_contributors c ON c.user_id = c_first.first_user_id
		LEFT JOIN devices d ON d.id = c.device_id
		LEFT JOIN access_log_geoip g ON g.access_log_id = ral.id
	` + buildWhere(whereFilters) + ` ORDER BY ral.id DESC LIMIT ?`
	if err := r.db.SelectContext(ctx, &dbRows, selectQuery, selectArgs...); err != nil {
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
			ID:            rRow.ID,
			ClientIP:      rRow.ClientIP,
			Outcome:       rRow.Outcome,
			DenyReason:    rRow.DenyReason,
			DeviceID:      rRow.DeviceID,
			DeviceName:    rRow.DeviceName,
			AddressID:     rRow.AddressID,
			CreatedAt:     rRow.CreatedAt,
			DurationUs:    rRow.DurationUs,
			XFFChain:      rRow.XFFChain,
			TargetHost:    rRow.TargetHost,
			TargetURI:     rRow.TargetURI,
			HTTPMethod:    rRow.HTTPMethod,
			Headers:       headers,
			CountryCode:   rRow.CountryCode,
			CountryName:   rRow.CountryName,
			ContinentCode: rRow.ContinentCode,
			ASN:           rRow.ASN,
			ASNOrg:        rRow.ASNOrg,
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
	ID            int64             `db:"id"`
	ClientIP      string            `db:"client_ip"`
	Outcome       bool              `db:"outcome"`
	DenyReason    *string           `db:"deny_reason"`
	DeviceID      *device.DeviceID  `db:"device_id"`
	DeviceName    *string           `db:"device_name"`
	AddressID     *device.AddressID `db:"address_id"`
	CreatedAt     time.Time         `db:"created_at"`
	DurationUs    int64             `db:"duration_us"`
	XFFChain      *string           `db:"xff_chain"`
	TargetHost    *string           `db:"target_host"`
	TargetURI     *string           `db:"target_uri"`
	HTTPMethod    *string           `db:"http_method"`
	HeadersRaw    string            `db:"headers_json"`
	CountryCode   *string           `db:"country_code"`
	CountryName   *string           `db:"country_name"`
	ContinentCode *string           `db:"continent_code"`
	ASN           *int64            `db:"asn"`
	ASNOrg        *string           `db:"asn_org"`
}

type dbCountryStatsRow struct {
	CountryCode   string `db:"country_code"`
	CountryName   string `db:"country_name"`
	ContinentCode string `db:"continent_code"`
	Total         int64  `db:"total"`
	Allowed       int64  `db:"allowed"`
	Denied        int64  `db:"denied"`
}

// Helper to format WHERE clause from a list of filters
func buildWhere(w []string) string {
	if len(w) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(w, " AND ")
}
