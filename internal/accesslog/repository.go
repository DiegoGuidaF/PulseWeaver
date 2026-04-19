package accesslog

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
)

// Repository owns the access log write path and simple single-table reads.
// Cross-domain reads (e.g. joining devices for device_name) live in internal/queries.
type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) BatchInsert(ctx context.Context, events []policy.DecisionEvent) error {
	if len(events) == 0 {
		return nil
	}

	return r.db.WithinTx(ctx, func(ctx context.Context) error {
		const insertAccessLog = `
            INSERT INTO access_log (
                client_ip, outcome, deny_reason, contributor_count,
                created_at, xff_chain, target_host, target_uri, http_method, headers_json,
                duration_us
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            RETURNING id
        `
		const insertGeoIP = `
            INSERT INTO access_log_geoip
                (access_log_id, country_code, country_name, continent_code, asn, asn_org)
            VALUES (?, ?, ?, ?, ?, ?)
        `
		const insertContributor = `
            INSERT INTO access_log_contributors (access_log_id, device_id, address_id, user_id)
            VALUES (?, ?, ?, ?)
        `

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
			if err := r.db.GetContext(ctx, &accessID, insertAccessLog,
				e.ClientIP, e.Outcome, e.DenyReason, contributorCount,
				e.CreatedAt, e.XFFChain, e.TargetHost, e.TargetURI, e.HTTPMethod,
				string(headersJSON), e.DurationUs,
			); err != nil {
				return fmt.Errorf("insert access event: %w", err)
			}

			for _, c := range e.IPContributors {
				if _, err := r.db.ExecContext(ctx, insertContributor,
					accessID, c.DeviceID, c.AddressID, c.UserID,
				); err != nil {
					return fmt.Errorf("insert contributor row: %w", err)
				}
			}

			if e.GeoIP.IsEmpty() {
				continue
			}
			if _, err := r.db.ExecContext(ctx, insertGeoIP,
				accessID, e.GeoIP.CountryCode, e.GeoIP.CountryName, e.GeoIP.ContinentCode, e.GeoIP.ASN, e.GeoIP.ASNOrg,
			); err != nil {
				return fmt.Errorf("insert geoip row: %w", err)
			}
		}
		return nil
	})
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
