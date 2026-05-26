BEGIN TRANSACTION;

-- Legacy pairings with no linked device, or with a nulled code (already claimed), cannot be migrated.
DELETE FROM pending_registrations WHERE created_device_id IS NULL OR registration_code IS NULL;

CREATE TABLE device_pairings
(
    id                         INTEGER  PRIMARY KEY,
    device_id                  INTEGER  NOT NULL REFERENCES devices (id),
    pairing_code               TEXT     NOT NULL UNIQUE,
    heartbeat_server_url       TEXT     NOT NULL,
    heartbeat_interval_seconds INTEGER  NOT NULL,
    app_biometric_enabled      INTEGER  NOT NULL DEFAULT 0,
    app_settings_locked        INTEGER  NOT NULL DEFAULT 0,
    expires_at                 DATETIME NOT NULL,
    created_at                 DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at                 DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    status                     TEXT     NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'used', 'invalidated', 'replaced'))
);

INSERT INTO device_pairings (id, device_id, pairing_code,
                              heartbeat_server_url, heartbeat_interval_seconds,
                              app_biometric_enabled, app_settings_locked,
                              expires_at, created_at, updated_at, status)
SELECT id, created_device_id, registration_code,
       heartbeat_server_url, heartbeat_interval_seconds,
       app_biometric_enabled, app_settings_locked,
       expires_at, created_at,
       CASE
           WHEN used_at IS NOT NULL        THEN used_at
           WHEN invalidated_at IS NOT NULL THEN invalidated_at
           ELSE created_at
       END,
       CASE
           WHEN used_at IS NOT NULL        THEN 'used'
           WHEN invalidated_at IS NOT NULL THEN 'invalidated'
           ELSE 'pending'
       END
FROM pending_registrations;

DROP TABLE pending_registrations;

PRAGMA foreign_key_check;

COMMIT;
