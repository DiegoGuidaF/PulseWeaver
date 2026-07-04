BEGIN TRANSACTION;

-- Detection findings produced by the background anomaly scan. Attribution
-- (device/user) is nullable and denormalized: the row is history and must stay
-- readable after its device or user is deleted, so ids null out (ON DELETE SET
-- NULL) while the names remain (same rationale as hourly_attribution_aggregates).
CREATE TABLE anomalies (
    id            INTEGER PRIMARY KEY,
    kind          TEXT     NOT NULL,
    severity      TEXT     NOT NULL,
    status        TEXT     NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'acknowledged')),
    fingerprint   TEXT     NOT NULL,
    first_seen_at DATETIME NOT NULL,
    last_seen_at  DATETIME NOT NULL,
    device_id     INTEGER  REFERENCES devices (id) ON DELETE SET NULL,
    device_name   TEXT     NOT NULL DEFAULT '',
    user_id       INTEGER  REFERENCES users (id) ON DELETE SET NULL,
    user_name     TEXT     NOT NULL DEFAULT '',
    client_ip     TEXT,
    target_host   TEXT,
    country_code  TEXT,
    evidence_json TEXT     NOT NULL DEFAULT '{}'
);
-- Dedup target: at most one OPEN row per fingerprint; acknowledged history may
-- repeat, so the uniqueness is partial. Re-detection upserts the open row.
CREATE UNIQUE INDEX idx_anomalies_open_fingerprint ON anomalies (fingerprint) WHERE status = 'open';
CREATE INDEX idx_anomalies_status_last_seen ON anomalies (status, last_seen_at DESC);

-- Per-device learned baselines for the novelty family: one row per
-- (device, dimension, value) seen. A fingerprint absent for a device's
-- dimension is the novelty signal; seen_count/last_seen_at track recurrence.
CREATE TABLE device_profiles (
    id            INTEGER PRIMARY KEY,
    device_id     INTEGER  NOT NULL REFERENCES devices (id) ON DELETE CASCADE,
    dimension     TEXT     NOT NULL CHECK (dimension IN ('user_agent', 'country')),
    fingerprint   TEXT     NOT NULL,
    first_seen_at DATETIME NOT NULL,
    last_seen_at  DATETIME NOT NULL,
    seen_count    INTEGER  NOT NULL DEFAULT 1
);
CREATE UNIQUE INDEX idx_device_profiles_key ON device_profiles (device_id, dimension, fingerprint);

-- Single-row incremental scan cursor. The rollup derives its cursor from
-- MAX(bucket_at) of its own output, but anomaly findings are sparse, so this
-- watermark must be persisted explicitly and advanced in the scan transaction.
CREATE TABLE anomaly_scan_state (
    id                 INTEGER PRIMARY KEY CHECK (id = 1),
    last_access_log_id INTEGER  NOT NULL DEFAULT 0,
    last_bucket_at     DATETIME
);

COMMIT;
