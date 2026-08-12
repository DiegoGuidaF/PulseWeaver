package accesslog

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
)

// Repository owns the access log write path plus the read models over it — the
// list, one entry's detail, the histogram and the country rollup, each isolated
// in a *_view.go file (ADR-009).
//
// traffic serves only the histogram's wide unfiltered window; a nil reader
// makes that case raw-scan like every other, which is what a write-path test
// wants and never what production does.
type Repository struct {
	db      *database.DB
	traffic TrafficSeriesReader
}

func NewRepository(db *database.DB, traffic TrafficSeriesReader) *Repository {
	return &Repository{
		db:      db,
		traffic: traffic,
	}
}

const insertAccessLogSQL = `
	INSERT INTO access_log (
		client_ip, outcome, deny_reason, contributor_count,
		created_at, xff_chain, target_host, target_uri, http_method, headers_json,
		duration_us
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id
`

func (r *Repository) BatchInsert(ctx context.Context, events []policy.DecisionEvent) error {
	if len(events) == 0 {
		return nil
	}

	return r.db.WithinTx(ctx, func(ctx context.Context) error {
		// The access_log insert runs once per row; preparing it once per batch
		// reuses the compiled statement across all rows instead of recompiling it
		// each iteration. Sparse child inserts stay per-row — they are absent on
		// the common (deny) path, so they are not worth keeping a prepared stmt for.
		insertStmt, err := r.db.PreparexContext(ctx, insertAccessLogSQL)
		if err != nil {
			return fmt.Errorf("prepare access event insert: %w", err)
		}
		defer func() { _ = insertStmt.Close() }()

		for _, e := range events {
			headers := e.Headers
			if headers == nil {
				headers = make(map[string][]string)
			}
			headersJSON, err := json.Marshal(headers)
			if err != nil {
				return fmt.Errorf("marshal headers_json: %w", err)
			}

			contributorCount := len(e.IPContributors)

			var accessID int64
			if err := insertStmt.GetContext(ctx, &accessID,
				e.ClientIP, e.Outcome, e.DenyReason, contributorCount,
				e.CreatedAt, e.XFFChain, e.TargetHost, e.TargetURI, e.HTTPMethod,
				string(headersJSON), e.DurationUs,
			); err != nil {
				return fmt.Errorf("insert access event: %w", err)
			}

			switch e.MatchSource {
			case policy.MatchSourceNetworkPolicy:
				if e.NetworkPolicyID != nil {
					if _, err := r.db.ExecContext(ctx,
						`
						INSERT INTO access_log_network_policy_contributors (access_log_id, policy_id, policy_name)
						VALUES (?, ?, ?)
						`, accessID, *e.NetworkPolicyID, new(e.NetworkPolicyName),
					); err != nil {
						return fmt.Errorf("insert network policy contributor: %w", err)
					}
				}
			// MatchSource is also its zero value ("") for denied requests, which fall
			// into this case alongside MatchSourceDevice: both record IP contributors.
			case policy.MatchSourceDevice, "":
				for _, c := range e.IPContributors {
					if _, err := r.db.ExecContext(ctx,
						`
						INSERT INTO access_log_contributors (access_log_id, device_id, address_id, user_id)
						VALUES (?, ?, ?, ?)
					`, accessID, c.DeviceID, c.AddressID, c.UserID,
					); err != nil {
						return fmt.Errorf("insert contributor row: %w", err)
					}
				}
			}

			if e.GeoIP.IsEmpty() {
				continue
			}
			if _, err := r.db.ExecContext(ctx,
				`
            	INSERT INTO access_log_geoip (access_log_id, country_code, country_name, continent_code, asn, asn_org)
            	VALUES (?, ?, ?, ?, ?, ?)
            `, accessID, e.GeoIP.CountryCode, e.GeoIP.CountryName, e.GeoIP.ContinentCode, e.GeoIP.ASN, e.GeoIP.ASNOrg,
			); err != nil {
				return fmt.Errorf("insert geoip row: %w", err)
			}
		}
		return nil
	})
}

// DeleteOlderThan removes access_log rows (and their children via CASCADE) with created_at before the given time.
// Returns the number of rows deleted from access_log.
func (r *Repository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	result, err := r.db.ExecContext(ctx, `DELETE FROM access_log WHERE created_at < ?`, before)
	if err != nil {
		return 0, fmt.Errorf("delete access_log older than %s: %w", before.Format(time.RFC3339), err)
	}
	return result.RowsAffected()
}

func (r *Repository) ListDenyReasons(ctx context.Context) ([]string, error) {
	const query = `
		SELECT DISTINCT deny_reason
		FROM access_log
		WHERE deny_reason IS NOT NULL
		ORDER BY deny_reason
	`
	var reasons []string
	if err := r.db.SelectContext(ctx, &reasons, query); err != nil {
		return nil, fmt.Errorf("list deny reasons: %w", err)
	}
	if reasons == nil {
		reasons = []string{}
	}
	return reasons, nil
}
