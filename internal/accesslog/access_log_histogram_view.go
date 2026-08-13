package accesslog

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/timebucket"
)

// trafficBucket is one aggregated row of the histogram query.
type trafficBucket struct {
	Timestamp  database.DBTime `db:"timestamp"`
	AllowCount int64           `db:"allow_count"`
	DenyCount  int64           `db:"deny_count"`
}

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

	rows, err := r.trafficBuckets(ctx, q, granularity.BucketExpr("ral.created_at"))
	if err != nil {
		return httpapi.AccessLogHistogramResponse{}, err
	}

	return httpapi.AccessLogHistogramResponse{
		Buckets: foldTrafficBuckets(granularity.Sequence(q.From, q.To), rows),
	}, nil
}

// trafficBuckets aggregates access_log under the shared filter set. The bucket
// expression is only a projection and grouping key: the bare created_at range in
// the shared conditions is what drives the index.
func (r *Repository) trafficBuckets(ctx context.Context, q AccessLogQuery, bucketExpr string) ([]trafficBucket, error) {
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

	var buckets []trafficBucket
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
func foldTrafficBuckets(sequence []time.Time, rows []trafficBucket) []httpapi.AccessLogHistogramBucket {
	byBucket := make(map[int64]trafficBucket, len(rows))
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
