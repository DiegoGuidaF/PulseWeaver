BEGIN TRANSACTION;

DROP INDEX IF EXISTS idx_hourly_aggregates_bucket_at;
DROP INDEX IF EXISTS idx_hourly_aggregates_bucket;
DROP TABLE IF EXISTS hourly_traffic_aggregates;

COMMIT;
