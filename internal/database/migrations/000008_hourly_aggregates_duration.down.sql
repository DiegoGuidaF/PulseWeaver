CREATE TABLE hourly_traffic_aggregates_new (
    id            INTEGER PRIMARY KEY,
    bucket_at     DATETIME NOT NULL,
    client_ip     TEXT     NOT NULL,
    target_host   TEXT     NOT NULL DEFAULT '',
    outcome       INTEGER  NOT NULL CHECK (outcome IN (0, 1)),
    deny_reason   TEXT     NOT NULL DEFAULT '',
    request_count INTEGER  NOT NULL DEFAULT 0
);

INSERT INTO hourly_traffic_aggregates_new
    SELECT id, bucket_at, client_ip, target_host, outcome, deny_reason, request_count
    FROM hourly_traffic_aggregates;

DROP TABLE hourly_traffic_aggregates;
ALTER TABLE hourly_traffic_aggregates_new RENAME TO hourly_traffic_aggregates;

CREATE UNIQUE INDEX IF NOT EXISTS idx_hourly_aggregates_bucket
    ON hourly_traffic_aggregates (bucket_at, client_ip, target_host, outcome, deny_reason);

CREATE INDEX IF NOT EXISTS idx_hourly_aggregates_bucket_at
    ON hourly_traffic_aggregates (bucket_at DESC);
