BEGIN TRANSACTION;

-- SQLite does not support DROP COLUMN on older versions; recreate the table.
CREATE TABLE access_log_new (
    id           INTEGER PRIMARY KEY,
    client_ip    TEXT     NOT NULL,
    outcome      INTEGER  NOT NULL CHECK (outcome IN (0, 1)),
    deny_reason  TEXT,
    device_id    INTEGER  REFERENCES devices(id) ON DELETE SET NULL,
    address_id   INTEGER  REFERENCES addresses(id) ON DELETE SET NULL,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    xff_chain    TEXT,
    target_host  TEXT,
    target_uri   TEXT,
    http_method  TEXT,
    headers_json TEXT     NOT NULL DEFAULT '{}'
);

INSERT INTO access_log_new
    SELECT id, client_ip, outcome, deny_reason, device_id, address_id,
           created_at, xff_chain, target_host, target_uri, http_method, headers_json
    FROM access_log;

DROP TABLE access_log;
ALTER TABLE access_log_new RENAME TO access_log;

CREATE INDEX IF NOT EXISTS idx_access_log_created_at ON access_log (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_access_log_client_ip  ON access_log (client_ip);
CREATE INDEX IF NOT EXISTS idx_access_log_device_id  ON access_log (device_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_access_log_outcome    ON access_log (outcome, created_at DESC);

COMMIT;
