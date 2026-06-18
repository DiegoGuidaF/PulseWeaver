package rollup

import (
	"context"
	"fmt"
	"time"
)

// attributionSpec is the only irreducible per-kind code: everything else in the
// attribution family is generic. It carries the source join, the entity id/name
// expressions, and the dedup mode for the rollup's request_count.
//
// Dedup matters because of the 1:N fan-out on the IP side:
// access_log_contributors lists ≥1 contributor per request, so a shared-IP
// request that names two of a user's devices must count once for that user. The
// raw and aggregate split queries always dedup by access_log.id; only the rollup
// pre-aggregates, so rollupCount distinguishes COUNT(*) (1:1 policy link) from
// COUNT(DISTINCT al.id) (1:N user/device links).
type attributionSpec struct {
	// source is the FROM clause: access_log aliased al, joined to the link table
	// and (for the IP kinds) the entity table that supplies the name.
	source string
	// idExpr selects the entity id, MAX-picked so a deleted entity's NULL never
	// displaces the live id when one name briefly carries both.
	idExpr string
	// nameExpr selects the denormalized entity name (also the GROUP BY key).
	nameExpr string
	// rollupCount is the request_count expression for RunAttributionRollup.
	rollupCount string
}

// attributionSpecs holds one spec per kind. Adding a fourth dimension is a new
// entry here, not a new table or flow.
var attributionSpecs = map[AttributionKind]attributionSpec{
	AttributionKindPolicy: {
		source:      "access_log al JOIN access_log_network_policy_contributors anpc ON anpc.access_log_id = al.id",
		idExpr:      "MAX(anpc.policy_id)",
		nameExpr:    "anpc.policy_name",
		rollupCount: "COUNT(*)",
	},
	AttributionKindUser: {
		source:      "access_log al JOIN access_log_contributors alc ON alc.access_log_id = al.id JOIN users u ON u.id = alc.user_id",
		idExpr:      "MAX(alc.user_id)",
		nameExpr:    "u.display_name",
		rollupCount: "COUNT(DISTINCT al.id)",
	},
	AttributionKindDevice: {
		source:      "access_log al JOIN access_log_contributors alc ON alc.access_log_id = al.id JOIN devices d ON d.id = alc.device_id",
		idExpr:      "MAX(alc.device_id)",
		nameExpr:    "d.name",
		rollupCount: "COUNT(DISTINCT al.id)",
	},
}

// RunAttributionRollup aggregates attribution-linked access_log rows in
// [from, to) into hourly_attribution_aggregates, one entity_kind per spec.
// Idempotent via INSERT OR REPLACE on the unique index. Populated from the same
// catch-up pass as RunRollup (see RollupJob.Run), so it shares one lastRollupAt
// cursor — no second scheduler.
//
// Grouping matches the unique index columns (bucket_at, entity_kind,
// entity_name, outcome).
func (r *Repository) RunAttributionRollup(ctx context.Context, from, to time.Time) error {
	for kind, spec := range attributionSpecs {
		query := fmt.Sprintf(`
			INSERT OR REPLACE INTO hourly_attribution_aggregates
				(bucket_at, entity_kind, entity_id, entity_name, outcome, request_count)
			SELECT
				strftime('%%Y-%%m-%%d %%H:00:00', al.created_at) || '+00:00' AS bucket_at,
				'%s'                                                         AS entity_kind,
				%s                                                           AS entity_id,
				%s                                                           AS entity_name,
				al.outcome                                                   AS outcome,
				%s                                                           AS request_count
			FROM %s
			WHERE al.created_at >= ?
			  AND al.created_at <  ?
			  AND strftime('%%Y-%%m-%%d %%H:00:00', al.created_at) IS NOT NULL
			GROUP BY bucket_at, %s, al.outcome
			`, kind, spec.idExpr, spec.nameExpr, spec.rollupCount, spec.source, spec.nameExpr)
		if _, err := r.db.ExecContext(ctx, query, from.UTC(), to.UTC()); err != nil {
			return fmt.Errorf("run attribution rollup (%s): %w", kind, err)
		}
	}
	return nil
}

// DeleteAttributionAggregatesOlderThan prunes hourly_attribution_aggregates
// buckets that start before the given cutoff and returns the number of deleted
// rows. Pruned at the same horizon as DeleteAggregatesOlderThan, across all
// entity kinds.
func (r *Repository) DeleteAttributionAggregatesOlderThan(ctx context.Context, before time.Time) (int64, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM hourly_attribution_aggregates WHERE bucket_at < ?`, before.UTC())
	if err != nil {
		return 0, fmt.Errorf("delete attribution aggregates older than: %w", err)
	}
	return res.RowsAffected()
}
