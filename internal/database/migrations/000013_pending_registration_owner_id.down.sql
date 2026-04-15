PRAGMA foreign_keys = OFF;
BEGIN TRANSACTION;

CREATE TABLE pending_registrations_without_owner (
    id                          TEXT     PRIMARY KEY,
    device_name                 TEXT     NOT NULL,
    registration_code           TEXT     UNIQUE,
    device_api_key              TEXT,
    device_api_key_prefix       TEXT     NOT NULL,
    heartbeat_server_url        TEXT     NOT NULL,
    heartbeat_interval_seconds  INTEGER  NOT NULL,
    biometric_enabled           INTEGER  NOT NULL DEFAULT 0,
    biometric_user_can_toggle   INTEGER  NOT NULL DEFAULT 1,
    expires_at                  DATETIME NOT NULL,
    created_at                  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    used_at                     DATETIME,
    created_device_id           INTEGER  REFERENCES devices(id)
);

INSERT INTO pending_registrations_without_owner
SELECT id, device_name, registration_code, device_api_key, device_api_key_prefix,
       heartbeat_server_url, heartbeat_interval_seconds, biometric_enabled,
       biometric_user_can_toggle, expires_at, created_at, used_at, created_device_id
FROM pending_registrations;

DROP TABLE pending_registrations;

ALTER TABLE pending_registrations_without_owner RENAME TO pending_registrations;

PRAGMA foreign_key_check;
COMMIT;
PRAGMA foreign_keys = ON;
