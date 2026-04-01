ALTER TABLE hourly_traffic_aggregates ADD COLUMN sum_duration_us INTEGER NOT NULL DEFAULT 0;
ALTER TABLE hourly_traffic_aggregates ADD COLUMN max_duration_us INTEGER NOT NULL DEFAULT 0;
