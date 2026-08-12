package accesslog

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/rollup"
	"github.com/DiegoGuidaF/PulseWeaver/internal/timebucket"
)

// TrafficSeriesReader answers the *unfiltered* allow/deny series over a window,
// already dispatching raw vs hourly aggregates on its own threshold. Declared
// consumer-side (Go convention); *rollup.Repository satisfies it.
type TrafficSeriesReader interface {
	GetTrafficSeries(ctx context.Context, from, to time.Time) ([]rollup.TrafficBucket, error)
}

// GetAccessLogHistogram answers the histogram endpoint: allow/deny counts per
// time bucket over exactly the rows GET /access-log lists for the same filters,
// contiguous across the whole window. Granularity follows the window size.
//
// An unfiltered request over a window wider than rollup.RawWindowThreshold is
// delegated to the traffic series, which reads the hourly aggregates instead of
// scanning months of access_log. The delegation is all-or-nothing: the
// aggregates carry no user, device or policy attribution, so any filter at all
// sends the query back to the raw scan rather than answering part of it from a
// second, divergent filter dialect.
func (r *Repository) GetAccessLogHistogram(ctx context.Context, q AccessLogQuery) (httpapi.AccessLogHistogramResponse, error) {
	granularity := timebucket.GranularityForWindow(q.To.Sub(q.From))

	var (
		rows []rollup.TrafficBucket
		err  error
	)
	if r.traffic != nil && len(q.Filters) == 0 && q.Outcome == nil && q.To.Sub(q.From) > rollup.RawWindowThreshold {
		rows, err = r.traffic.GetTrafficSeries(ctx, q.From, q.To)
	} else {
		rows, err = r.rawTrafficBuckets(ctx, q, granularity.BucketExpr("ral.created_at"))
	}
	if err != nil {
		return httpapi.AccessLogHistogramResponse{}, err
	}

	return httpapi.AccessLogHistogramResponse{
		Buckets: foldTrafficBuckets(granularity.Sequence(q.From, q.To), rows),
	}, nil
}

// rawTrafficBuckets aggregates access_log itself under the shared filter set.
// The bucket expression is only a projection and grouping key: the bare
// created_at range in the shared conditions is what drives the index.
func (r *Repository) rawTrafficBuckets(ctx context.Context, q AccessLogQuery, bucketExpr string) ([]rollup.TrafficBucket, error) {
	cond, err := accessLogConditions(q)
	if err != nil {
		return nil, err
	}

	query, args, err := accessLogFilteredFrom(sq.Select(), q).
		Column(bucketExpr + " AS timestamp").
		Column("COALESCE(SUM(CASE WHEN ral.outcome = 1 THEN 1 ELSE 0 END), 0) AS allow_count").
		Column("COALESCE(SUM(CASE WHEN ral.outcome = 0 THEN 1 ELSE 0 END), 0) AS deny_count").
		Where(cond).
		GroupBy("timestamp").
		OrderBy("timestamp").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build access log histogram query: %w", err)
	}

	var buckets []rollup.TrafficBucket
	if err := r.db.SelectContext(ctx, &buckets, query, args...); err != nil {
		return nil, fmt.Errorf("get access log histogram: %w", err)
	}
	return buckets, nil
}

// foldTrafficBuckets projects the aggregated rows onto the bucket sequence. The
// sequence drives the result, not the rows: a bucket nothing matched in still
// appears with both counts at zero. Under a filter most buckets are empty, and
// a series that simply omitted them would draw as one unbroken run of traffic.
// Rows outside the sequence are dropped rather than extending the window.
func foldTrafficBuckets(sequence []time.Time, rows []rollup.TrafficBucket) []httpapi.AccessLogHistogramBucket {
	byBucket := make(map[int64]rollup.TrafficBucket, len(rows))
	for _, b := range rows {
		byBucket[b.Timestamp.UTC().Unix()] = b
	}

	buckets := make([]httpapi.AccessLogHistogramBucket, len(sequence))
	for i, ts := range sequence {
		// A bucket with no row folds to the zero value, which reads as empty.
		row := byBucket[ts.Unix()]
		buckets[i] = httpapi.AccessLogHistogramBucket{
			Timestamp:  httpapi.UTCTime(ts),
			AllowCount: row.AllowCount,
			DenyCount:  row.DenyCount,
		}
	}
	return buckets
}
