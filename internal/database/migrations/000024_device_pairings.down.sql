BEGIN TRANSACTION;

CREATE TABLE pending_registrations
(
    id                         INTEGER  PRIMARY KEY,
    device_name                TEXT     NOT NULL DEFAULT '',
    owner_id                   INTEGER  NOT NULL DEFAULT 1 REFERENCES users (id),
    registration_code          TEXT     UNIQUE,
    heartbeat_server_url       TEXT     NOT NULL,
    heartbeat_interval_seconds INTEGER  NOT NULL,
    app_biometric_enabled      INTEGER  NOT NULL DEFAULT 0,
    app_settings_locked        INTEGER  NOT NULL DEFAULT 0,
    expires_at                 DATETIME NOT NULL,
    created_at                 DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    used_at                    DATETIME,
    invalidated_at             DATETIME,
    created_device_id          INTEGER REFERENCES devices (id)
);

INSERT INTO pending_registrations (id, created_device_id, registration_code,
                                    heartbeat_server_url, heartbeat_interval_seconds,
                                    app_biometric_enabled, app_settings_locked,
                                    expires_at, created_at, used_at, invalidated_at)
SELECT id, device_id, pairing_code,
       heartbeat_server_url, heartbeat_interval_seconds,
       app_biometric_enabled, app_settings_locked,
       expires_at, created_at,
       CASE WHEN status = 'used'        THEN updated_at ELSE NULL END,
       CASE WHEN status = 'invalidated' THEN updated_at ELSE NULL END
FROM device_pairings;

DROP TABLE device_pairings;

PRAGMA foreign_key_check;

COMMIT;
