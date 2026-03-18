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

type RequestAuditLogView struct {
	ID         int64
	ClientIP   string
	Outcome    bool
	DenyReason *string
	DeviceID   *device.DeviceID
	DeviceName *string
	AddressID  *device.AddressID
	CreatedAt  time.Time
	XFFChain   *string
	TargetHost *string
	TargetURI  *string
	HTTPMethod *string
	Headers    map[string][]string
}

type RequestAuditLogQuery struct {
	From       time.Time
	To         time.Time
	BeforeID   *int64 // cursor: return rows with id < BeforeID; nil for first page
	ClientIP   *string
	Outcome    *bool
	DenyReason *string
	DeviceID   *device.DeviceID
	TargetHost *string
	Limit      int
}

func NewRequestAuditLogQuery(params httpapi.GetRequestAuditLogParams) RequestAuditLogQuery {
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

	return RequestAuditLogQuery{
		DeviceID:   (*device.DeviceID)(params.DeviceId),
		Outcome:    params.Outcome,
		DenyReason: params.DenyReason,
		ClientIP:   params.Ip,
		TargetHost: params.Host,
		From:       from,
		To:         to,
		Limit:      limit,
		BeforeID:   params.BeforeId,
	}

}

func (r *Repository) ListRequestAuditLog(ctx context.Context, q RequestAuditLogQuery) ([]RequestAuditLogView, int, error) {

	whereFilters := []string{"1=1"}
	var countArgs []any

	if q.DeviceID != nil {
		whereFilters = append(whereFilters, "ral.device_id = ?")
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

	if !q.From.IsZero() {
		whereFilters = append(whereFilters, "ral.created_at >= ?")
		countArgs = append(countArgs, q.From)
	}

	if !q.To.IsZero() {
		whereFilters = append(whereFilters, "ral.created_at <= ?")
		countArgs = append(countArgs, q.To)
	}

	// Total count
	var total int
	countQuery := `
		SELECT COUNT(*) FROM request_audit_log ral
		LEFT JOIN devices d ON d.id = ral.device_id
	` + buildWhere(whereFilters)
	if err := r.db.GetContext(ctx, &total, countQuery, countArgs...); err != nil {
		return nil, 0, fmt.Errorf("count audit log: %w", err)
	}

	selectArgs := countArgs

	// Append Cursor for pagination: rows with id < BeforeID
	if q.BeforeID != nil {
		whereFilters = append(whereFilters, "ral.id < ?")
		selectArgs = append(selectArgs, *q.BeforeID)
	}
	//selectQuery := baseSelect + " " + whereClause +
	selectArgs = append(selectArgs, q.Limit)

	var dbRows []dbRequestAuditLogRow
	// Base SELECT query with LEFT JOIN to devices for device_name.
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
			ral.device_id as device_id,
			ral.address_id as address_id,
			d.name as device_name,
			ral.headers_json
		FROM request_audit_log ral
		LEFT JOIN devices d ON d.id = ral.device_id
	` + buildWhere(whereFilters) + ` ORDER BY ral.id DESC LIMIT ?`
	if err := r.db.SelectContext(ctx, &dbRows, selectQuery, selectArgs...); err != nil {
		return nil, 0, fmt.Errorf("list audit log: %w", err)
	}

	rows := make([]RequestAuditLogView, len(dbRows))
	for i, rRow := range dbRows {
		var headers map[string][]string
		if err := json.Unmarshal([]byte(rRow.HeadersRaw), &headers); err != nil {
			// Malformed JSON should not break the endpoint; fall back to empty map.
			headers = map[string][]string{}
		}

		rows[i] = RequestAuditLogView{
			ID:         rRow.ID,
			ClientIP:   rRow.ClientIP,
			Outcome:    rRow.Outcome,
			DenyReason: rRow.DenyReason,
			DeviceID:   rRow.DeviceID,
			DeviceName: rRow.DeviceName,
			AddressID:  rRow.AddressID,
			CreatedAt:  rRow.CreatedAt,
			XFFChain:   rRow.XFFChain,
			TargetHost: rRow.TargetHost,
			TargetURI:  rRow.TargetURI,
			HTTPMethod: rRow.HTTPMethod,
			Headers:    headers,
		}
	}

	if len(rows) == 0 {
		rows = []RequestAuditLogView{}
	}

	return rows, total, nil
}

// Page of rows.
type dbRequestAuditLogRow struct {
	ID         int64             `db:"id"`
	ClientIP   string            `db:"client_ip"`
	Outcome    bool              `db:"outcome"`
	DenyReason *string           `db:"deny_reason"`
	DeviceID   *device.DeviceID  `db:"device_id"`
	DeviceName *string           `db:"device_name"`
	AddressID  *device.AddressID `db:"address_id"`
	CreatedAt  time.Time         `db:"created_at"`
	XFFChain   *string           `db:"xff_chain"`
	TargetHost *string           `db:"target_host"`
	TargetURI  *string           `db:"target_uri"`
	HTTPMethod *string           `db:"http_method"`
	HeadersRaw string            `db:"headers_json"`
}

// Helper to format WHERE clause from a list of filters
func buildWhere(w []string) string {
	if len(w) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(w, " AND ")
}
