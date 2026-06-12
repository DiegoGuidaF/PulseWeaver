BEGIN TRANSACTION;

ALTER TABLE hourly_traffic_aggregates ADD COLUMN country_code TEXT NOT NULL DEFAULT '';
ALTER TABLE hourly_traffic_aggregates ADD COLUMN country_name TEXT NOT NULL DEFAULT '';
ALTER TABLE hourly_traffic_aggregates ADD COLUMN continent_code TEXT NOT NULL DEFAULT '';

-- Drop aggregate rows that the retained raw data can rebuild, so the next
-- rollup catch-up re-creates them with country attribution. The bucket of the
-- earliest raw row is kept: that hour may be partially pruned, so its existing
-- aggregate (computed when the hour was complete) is more accurate than a
-- re-roll. With an empty access_log the subquery is NULL and nothing is deleted.
DELETE FROM hourly_traffic_aggregates
WHERE bucket_at > (SELECT strftime('%Y-%m-%d %H:00:00', MIN(created_at)) || '+00:00' FROM access_log);

COMMIT;
