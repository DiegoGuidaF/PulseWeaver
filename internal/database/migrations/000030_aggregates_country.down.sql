BEGIN TRANSACTION;

ALTER TABLE hourly_traffic_aggregates DROP COLUMN country_code;
ALTER TABLE hourly_traffic_aggregates DROP COLUMN country_name;
ALTER TABLE hourly_traffic_aggregates DROP COLUMN continent_code;

COMMIT;
