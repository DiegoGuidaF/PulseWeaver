PRAGMA foreign_keys = OFF;
BEGIN TRANSACTION;

CREATE TABLE pending_registrations_new (
    id                          TEXT     PRIMARY KEY,
    device_name                 TEXT     NOT NULL,
    owner_id                    INTEGER  NOT NULL DEFAULT 1 REFERENCES users(id),
    registration_code           TEXT     UNIQUE,
    device_api_key              TEXT,
    device_api_key_prefix       TEXT     NOT NULL,
    heartbeat_server_url        TEXT     NOT NULL,
    heartbeat_interval_seconds  INTEGER  NOT NULL,
    app_biometric_enabled       INTEGER  NOT NULL DEFAULT 0,
    app_settings_locked         INTEGER  NOT NULL DEFAULT 0,
    expires_at                  DATETIME NOT NULL,
    created_at                  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    used_at                     DATETIME,
    created_device_id           INTEGER  REFERENCES devices(id)
);

-- app_settings_locked is the inverse of biometric_user_can_toggle
INSERT INTO pending_registrations_new
SELECT id, device_name, owner_id, registration_code, device_api_key, device_api_key_prefix,
       heartbeat_server_url, heartbeat_interval_seconds,
       biometric_enabled,
       1 - biometric_user_can_toggle,
       expires_at, created_at, used_at, created_device_id
FROM pending_registrations;

DROP TABLE pending_registrations;

ALTER TABLE pending_registrations_new RENAME TO pending_registrations;

PRAGMA foreign_key_check;
COMMIT;
PRAGMA foreign_keys = ON;
