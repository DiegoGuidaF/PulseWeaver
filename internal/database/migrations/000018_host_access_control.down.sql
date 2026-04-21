PRAGMA foreign_keys = OFF;
BEGIN TRANSACTION;

-- ── 1. Restore access_log with device_id/address_id ──────────────────────────

CREATE TABLE access_log_old
(
    id         INTEGER PRIMARY KEY,
    client_ip  TEXT    NOT NULL,
    outcome    INTEGER NOT NULL CHECK (outcome IN (0, 1)),
    deny_reason TEXT,
    device_id  INTEGER REFERENCES devices (id) ON DELETE SET NULL,
    address_id INTEGER REFERENCES addresses (id) ON DELETE SET NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    xff_chain  TEXT,
    target_host TEXT,
    target_uri  TEXT,
    http_method TEXT,
    headers_json TEXT   NOT NULL DEFAULT '{}',
    duration_us INTEGER NOT NULL DEFAULT 0
);

-- Restore device_id/address_id from the first contributor row (single-device case).
-- Multi-contributor rows (contributor_count > 1) get NULL device/address — acceptable on rollback.
INSERT INTO access_log_old
    (id, client_ip, outcome, deny_reason, device_id, address_id,
     created_at, xff_chain, target_host, target_uri, http_method, headers_json, duration_us)
SELECT
    al.id,
    al.client_ip,
    al.outcome,
    al.deny_reason,
    c.device_id,
    c.address_id,
    al.created_at,
    al.xff_chain,
    al.target_host,
    al.target_uri,
    al.http_method,
    al.headers_json,
    al.duration_us
FROM access_log al
LEFT JOIN (
    SELECT access_log_id, device_id, address_id
    FROM access_log_contributors
    GROUP BY access_log_id
    HAVING id = MIN(id)
) c ON c.access_log_id = al.id;

DROP TABLE access_log;
DROP TABLE access_log_contributors;
ALTER TABLE access_log_old RENAME TO access_log;

-- Restore original indexes.
CREATE INDEX idx_request_audit_log_created_at ON access_log (created_at DESC);
CREATE INDEX idx_request_audit_log_client_ip  ON access_log (client_ip);
CREATE INDEX idx_request_audit_log_device_id  ON access_log (device_id, created_at DESC);
CREATE INDEX idx_request_audit_log_outcome    ON access_log (outcome, created_at DESC);

-- ── 2. Drop host-access domain tables and user settings ──────────────────────

DROP TABLE IF EXISTS user_allowed_host_groups;
DROP TABLE IF EXISTS user_allowed_hosts;
DROP TABLE IF EXISTS user_host_settings;
DROP TABLE IF EXISTS host_group_members;
DROP TABLE IF EXISTS host_groups;
DROP TABLE IF EXISTS known_hosts;
DROP TABLE IF EXISTS ignored_host_suggestions;

PRAGMA foreign_key_check;
COMMIT;
PRAGMA foreign_keys = ON;
