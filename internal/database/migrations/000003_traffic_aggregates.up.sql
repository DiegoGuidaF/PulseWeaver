CREATE TABLE IF NOT EXISTS hourly_traffic_aggregates (
    id            INTEGER PRIMARY KEY,
    bucket_at     DATETIME NOT NULL,   -- truncated to the hour, UTC
    client_ip     TEXT     NOT NULL,
    target_host   TEXT     NOT NULL DEFAULT '',
    outcome       INTEGER  NOT NULL CHECK (outcome IN (0, 1)), -- 1=allow, 0=deny
    deny_reason   TEXT     NOT NULL DEFAULT '',
    request_count INTEGER  NOT NULL DEFAULT 0
);

-- Unique key ensures the rollup job is idempotent (INSERT OR REPLACE)
CREATE UNIQUE INDEX IF NOT EXISTS idx_hourly_aggregates_bucket
    ON hourly_traffic_aggregates (bucket_at, client_ip, target_host, outcome, deny_reason);

CREATE INDEX IF NOT EXISTS idx_hourly_aggregates_bucket_at
    ON hourly_traffic_aggregates (bucket_at DESC);
