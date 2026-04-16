PRAGMA foreign_keys = OFF;
BEGIN TRANSACTION;

-- Remove device_api_key (no longer pre-generated at invite creation; key is generated at claim time).
-- Remove device_api_key_prefix (redundant after claim — accessible via created_device_id → device_api_keys).
-- Add invalidated_at for soft-delete on InvalidateInvite (replaces hard DELETE).
CREATE TABLE pending_registrations_new (
    id                          TEXT     PRIMARY KEY,
    device_name                 TEXT     NOT NULL,
    owner_id                    INTEGER  NOT NULL DEFAULT 1 REFERENCES users(id),
    registration_code           TEXT     UNIQUE,
    heartbeat_server_url        TEXT     NOT NULL,
    heartbeat_interval_seconds  INTEGER  NOT NULL,
    app_biometric_enabled       INTEGER  NOT NULL DEFAULT 0,
    app_settings_locked         INTEGER  NOT NULL DEFAULT 0,
    expires_at                  DATETIME NOT NULL,
    created_at                  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    used_at                     DATETIME,
    invalidated_at              DATETIME,
    created_device_id           INTEGER  REFERENCES devices(id)
);

INSERT INTO pending_registrations_new
SELECT id, device_name, owner_id, registration_code,
       heartbeat_server_url, heartbeat_interval_seconds,
       app_biometric_enabled, app_settings_locked,
       expires_at, created_at, used_at, NULL, created_device_id
FROM pending_registrations;

DROP TABLE pending_registrations;
ALTER TABLE pending_registrations_new RENAME TO pending_registrations;

PRAGMA foreign_key_check;
COMMIT;
PRAGMA foreign_keys = ON;
