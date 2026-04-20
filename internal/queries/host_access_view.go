package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/hostaccess"
)

type KnownHostStats struct {
	ID        hostaccess.KnownHostID `db:"id"`
	FQDN      string                 `db:"fqdn"`
	Icon      *string                `db:"icon"`
	CreatedAt time.Time              `db:"created_at"`
	LastSeen  *time.Time             `db:"last_seen"`
	UserCount int                    `db:"user_count"`
}

type HostSuggestion struct {
	FQDN        string    `db:"fqdn"`
	FirstSeen   time.Time `db:"first_seen"`
	AllowedHits int       `db:"allowed_hits"`
	DeniedHits  int       `db:"denied_hits"`
}

func (r *Repository) GetKnownHostsWithStats(ctx context.Context) ([]KnownHostStats, error) {
	const query = `
		SELECT
			kh.id,
			kh.fqdn,
			kh.icon,
			kh.created_at,
			MAX(al.created_at) AS last_seen,
			(
				SELECT COUNT(DISTINCT user_id) FROM (
					SELECT uah.user_id FROM user_allowed_hosts uah WHERE uah.known_host_id = kh.id
					UNION
					SELECT uahg.user_id FROM user_allowed_host_groups uahg
					JOIN host_group_members hgm ON hgm.host_group_id = uahg.host_group_id
					WHERE hgm.known_host_id = kh.id
				)
			) AS user_count
		FROM known_hosts kh
		LEFT JOIN access_log al ON LOWER(al.target_host) = kh.fqdn
		GROUP BY kh.id, kh.fqdn, kh.icon, kh.created_at
		ORDER BY kh.fqdn
	`
	var hosts []KnownHostStats
	if err := r.db.SelectContext(ctx, &hosts, query); err != nil {
		return nil, fmt.Errorf("get known hosts with stats: %w", err)
	}
	if hosts == nil {
		hosts = []KnownHostStats{}
	}
	return hosts, nil
}

func (r *Repository) GetHostSuggestions(ctx context.Context) ([]HostSuggestion, error) {
	const query = `
		SELECT
			LOWER(al.target_host) AS fqdn,
			MIN(al.created_at)    AS first_seen,
			SUM(CASE WHEN al.outcome = 1 THEN 1 ELSE 0 END) AS allowed_hits,
			SUM(CASE WHEN al.outcome = 0 THEN 1 ELSE 0 END) AS denied_hits
		FROM access_log al
		WHERE al.target_host IS NOT NULL
		  AND LOWER(al.target_host) NOT IN (SELECT fqdn FROM known_hosts)
		  AND LOWER(al.target_host) NOT IN (SELECT fqdn FROM ignored_host_suggestions)
		GROUP BY LOWER(al.target_host)
		ORDER BY denied_hits DESC, allowed_hits DESC
	`
	var suggestions []HostSuggestion
	if err := r.db.SelectContext(ctx, &suggestions, query); err != nil {
		return nil, fmt.Errorf("get host suggestions: %w", err)
	}
	if suggestions == nil {
		suggestions = []HostSuggestion{}
	}
	return suggestions, nil
}
