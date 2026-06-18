package rollup

import (
	"context"
	"fmt"
	"time"
)

// GetAttributionSplit returns per-entity allow/deny counts for the given kind.
// Uses access_log + the link table directly for windows ≤ 24h;
// hourly_attribution_aggregates for longer windows. The two paths agree for the
// same window (the F18 invariant), but per-entity totals deliberately do NOT
// reconcile to global traffic: fan-out and the attributed-only subset mean a
// per-user split can sum above total traffic, which is correct.
func (r *Repository) GetAttributionSplit(ctx context.Context, kind AttributionKind, from, to time.Time) ([]AttributionCount, error) {
	spec, ok := attributionSpecs[kind]
	if !ok {
		return nil, fmt.Errorf("get attribution split: unknown kind %q", kind)
	}
	if to.Sub(from) <= RawWindowThreshold {
		return r.getRawAttributionSplit(ctx, spec, from, to)
	}
	return r.getAggregateAttributionSplit(ctx, kind, from, to)
}

// getRawAttributionSplit aggregates allow/deny per entity straight from
// access_log + the link table. allow/deny dedup by access_log.id so the 1:N
// fan-out on the IP side counts a shared-IP request once per entity.
func (r *Repository) getRawAttributionSplit(ctx context.Context, spec attributionSpec, from, to time.Time) ([]AttributionCount, error) {
	query := fmt.Sprintf(`
	SELECT
		%s                                                                   AS entity_id,
		%s                                                                   AS entity_name,
		COUNT(DISTINCT CASE WHEN al.outcome = 1 THEN al.id END)              AS allow_count,
		COUNT(DISTINCT CASE WHEN al.outcome = 0 THEN al.id END)              AS deny_count
	FROM %s
	WHERE al.created_at >= ? AND al.created_at < ?
	GROUP BY %s
	ORDER BY (allow_count + deny_count) DESC
	`, spec.idExpr, spec.nameExpr, spec.source, spec.nameExpr)

	var counts []AttributionCount
	if err := r.db.SelectContext(ctx, &counts, query, from.UTC(), to.UTC()); err != nil {
		return nil, fmt.Errorf("get raw attribution split: %w", err)
	}
	if counts == nil {
		counts = []AttributionCount{}
	}
	return counts, nil
}

// getAggregateAttributionSplit reads the pre-rolled counts for one entity_kind.
// The query is identical across kinds — only the entity_kind filter differs.
func (r *Repository) getAggregateAttributionSplit(ctx context.Context, kind AttributionKind, from, to time.Time) ([]AttributionCount, error) {
	const query = `
	SELECT
		MAX(entity_id)                                                       AS entity_id,
		entity_name                                                          AS entity_name,
		COALESCE(SUM(CASE WHEN outcome = 1 THEN request_count ELSE 0 END), 0) AS allow_count,
		COALESCE(SUM(CASE WHEN outcome = 0 THEN request_count ELSE 0 END), 0) AS deny_count
	FROM hourly_attribution_aggregates
	WHERE entity_kind = ? AND bucket_at >= ? AND bucket_at < ?
	GROUP BY entity_name
	ORDER BY (allow_count + deny_count) DESC
	`
	var counts []AttributionCount
	if err := r.db.SelectContext(ctx, &counts, query, string(kind), from.UTC(), to.UTC()); err != nil {
		return nil, fmt.Errorf("get aggregate attribution split: %w", err)
	}
	if counts == nil {
		counts = []AttributionCount{}
	}
	return counts, nil
}
