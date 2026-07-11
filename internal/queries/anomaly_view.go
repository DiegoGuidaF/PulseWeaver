package queries

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"

	"github.com/DiegoGuidaF/PulseWeaver/internal/anomaly"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
)

// AnomalyListQuery is the resolved, validated filter for the anomaly list. The
// handler builds it from request params; a nil/empty field means "no filter".
type AnomalyListQuery struct {
	Status   *string
	Severity *string
	Kinds    []string
	Limit    int
}

// ListAnomalies returns detected anomalies newest first (by last-seen), filtered
// by the optional status and kind set. The anomalies table is retention-bounded,
// so a plain limit replaces cursor pagination.
func (r *Repository) ListAnomalies(ctx context.Context, q AnomalyListQuery) ([]httpapi.Anomaly, error) {
	cond := sq.And{}
	if q.Status != nil {
		cond = append(cond, sq.Eq{"status": *q.Status})
	}
	if q.Severity != nil {
		cond = append(cond, sq.Eq{"severity": *q.Severity})
	}
	if len(q.Kinds) > 0 {
		cond = append(cond, sq.Eq{"kind": q.Kinds})
	}

	query, args, err := sq.
		Select("id", "kind", "severity", "status", "first_seen_at", "last_seen_at",
			"device_id", "device_name", "user_id", "user_name",
			"client_ip", "target_host", "country_code", "evidence_json").
		From("anomalies").
		Where(cond).
		OrderBy("last_seen_at DESC", "id DESC").
		Limit(uint64(q.Limit)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build anomaly list query: %w", err)
	}

	type anomalyRow struct {
		ID           int64     `db:"id"`
		Kind         string    `db:"kind"`
		Severity     string    `db:"severity"`
		Status       string    `db:"status"`
		FirstSeenAt  time.Time `db:"first_seen_at"`
		LastSeenAt   time.Time `db:"last_seen_at"`
		DeviceID     *int64    `db:"device_id"`
		DeviceName   string    `db:"device_name"`
		UserID       *int64    `db:"user_id"`
		UserName     string    `db:"user_name"`
		ClientIP     *string   `db:"client_ip"`
		TargetHost   *string   `db:"target_host"`
		CountryCode  *string   `db:"country_code"`
		EvidenceJSON string    `db:"evidence_json"`
	}

	var rows []anomalyRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("list anomalies: %w", err)
	}

	out := make([]httpapi.Anomaly, len(rows))
	for i, row := range rows {
		evidence := map[string]any{}
		if row.EvidenceJSON != "" {
			if err := json.Unmarshal([]byte(row.EvidenceJSON), &evidence); err != nil {
				return nil, fmt.Errorf("decode evidence for anomaly %d: %w", row.ID, err)
			}
		}
		out[i] = httpapi.Anomaly{
			Id:          row.ID,
			Kind:        httpapi.AnomalyKind(row.Kind),
			Severity:    httpapi.AnomalySeverity(row.Severity),
			Status:      httpapi.AnomalyStatus(row.Status),
			FirstSeenAt: httpapi.UTCTime(row.FirstSeenAt),
			LastSeenAt:  httpapi.UTCTime(row.LastSeenAt),
			DeviceId:    idPtr(row.DeviceID),
			DeviceName:  nonEmptyPtr(row.DeviceName),
			UserId:      idPtr(row.UserID),
			UserName:    nonEmptyPtr(row.UserName),
			ClientIp:    row.ClientIP,
			TargetHost:  row.TargetHost,
			CountryCode: row.CountryCode,
			Evidence:    evidence,
			Summary:     anomaly.Summarize(anomaly.Kind(row.Kind), evidence),
		}
	}
	return out, nil
}

// idPtr renders a nullable int64 FK as the API ID pointer.
func idPtr(v *int64) *httpapi.ID {
	if v == nil {
		return nil
	}
	return (*httpapi.ID)(v)
}

// nonEmptyPtr returns a pointer to s, or nil when s is empty, so a denormalized
// name that never applied is omitted rather than serialized as "".
func nonEmptyPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
