package accesslog

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/timebucket"
)

// GetAccessLogHistogram answers the histogram endpoint: allow/deny counts per
// time bucket over exactly the rows GET /access-log lists for the same filters,
// contiguous across the whole window. Granularity follows the window size.
//
// It always aggregates access_log itself, at every window width. The hourly
// rollups cannot serve it: they carry no user, device or policy attribution, so
// they can only answer the unfiltered case, and they exclude the in-flight hour
// — which would make the chart quietly disagree with the table beside it.
func (r *Repository) GetAccessLogHistogram(ctx context.Context, q AccessLogQuery) (httpapi.AccessLogHistogramResponse, error) {
	granularity := timebucket.GranularityForWindow(q.To.Sub(q.From))

	buckets, err := r.trafficBuckets(ctx, q, granularity)
	if err != nil {
		return httpapi.AccessLogHistogramResponse{}, err
	}

	return httpapi.AccessLogHistogramResponse{Buckets: buckets}, nil
}

// trafficBuckets aggregates access_log once per bucket in the sequence, each
// query a bare created_at range under the shared filter set.
//
// Deriving the bucket in SQL instead — strftime(created_at) as the GROUP BY key
// — costs a date parse and a temp-B-tree insert for every row in the window to
// produce one output row per bucket; at month width that is ~85% of the query,
// 1.4M rows of work to return 31 numbers. The boundaries are already known here,
// so the same answer falls out of one indexed range aggregate per bucket, and
// GranularityForWindow bounds the sequence (≤168 buckets) so the fan-out stays
// small. See docs/patterns/backend/sargable-predicates.md.
//
// The sequence drives the result, not the rows: a bucket nothing matched in
// still appears with both counts at zero, since COALESCE(SUM(…), 0) over an
// empty range is 0. Under a filter most buckets are empty, and a series that
// simply omitted them would draw as one unbroken run of traffic.
//
// The buckets are read one statement at a time rather than in a snapshot: rows
// are only ever appended, at created_at ≈ now, so an insert racing the loop can
// only land in the in-flight bucket the loop has yet to reach — which is the
// same row a single grouped query would have counted.
func (r *Repository) trafficBuckets(
	ctx context.Context,
	q AccessLogQuery,
	granularity timebucket.Granularity,
) ([]httpapi.AccessLogHistogramBucket, error) {
	cond, err := accessLogFilterConditions(q)
	if err != nil {
		return nil, err
	}

	sequence := granularity.Sequence(q.From, q.To)
	buckets := make([]httpapi.AccessLogHistogramBucket, len(sequence))
	if len(sequence) == 0 {
		return buckets, nil
	}

	// One lower and one upper bound per bucket, already intersected with the
	// window — see accessLogWindowConditions for why a second pair would cost the
	// whole window per bucket. The intersection is also what clips the first and
	// last buckets, so a bucket only reports the part of itself the caller asked
	// for.
	bucketQuery := func(start time.Time) sq.SelectBuilder {
		lower := start
		if !q.From.IsZero() && q.From.After(lower) {
			lower = q.From
		}
		next := granularity.Next(start)
		var upper sq.Sqlizer = sq.Lt{"ral.created_at": next}
		if !q.To.IsZero() && q.To.Before(next) {
			upper = sq.LtOrEq{"ral.created_at": q.To}
		}

		return accessLogFilteredFrom(sq.Select(), q).
			Column("COALESCE(SUM(CASE WHEN ral.outcome = 1 THEN 1 ELSE 0 END), 0)").
			Column("COALESCE(SUM(CASE WHEN ral.outcome = 0 THEN 1 ELSE 0 END), 0)").
			Where(sq.GtOrEq{"ral.created_at": lower}).
			Where(upper).
			Where(cond)
	}

	// Only the closing bucket's bound differs in shape (it takes the window's
	// inclusive end), so the sequence renders at most two distinct statements —
	// each compiled once and reused across every bucket that shares it. Args come
	// back through the builder rather than a hand-ordered []any, per
	// docs/patterns/backend/dynamic-query-filtering.md.
	stmts := make(map[string]*sqlx.Stmt, 2)
	defer func() {
		for _, stmt := range stmts {
			_ = stmt.Close()
		}
	}()

	for i, start := range sequence {
		query, args, err := bucketQuery(start).ToSql()
		if err != nil {
			return nil, fmt.Errorf("build access log histogram query: %w", err)
		}

		stmt, ok := stmts[query]
		if !ok {
			if stmt, err = r.db.PreparexContext(ctx, query); err != nil {
				return nil, fmt.Errorf("prepare access log histogram query: %w", err)
			}
			stmts[query] = stmt
		}

		var allow, deny int64
		if err := stmt.QueryRowxContext(ctx, args...).Scan(&allow, &deny); err != nil {
			return nil, fmt.Errorf("get access log histogram: %w", err)
		}

		buckets[i] = httpapi.AccessLogHistogramBucket{
			Timestamp:  httpapi.UTCTime(start),
			AllowCount: allow,
			DenyCount:  deny,
		}
	}
	return buckets, nil
}
