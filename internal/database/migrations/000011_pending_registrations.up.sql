BEGIN TRANSACTION;

CREATE TABLE pending_registrations (
    id                          TEXT     PRIMARY KEY,

    -- Admin-provided metadata
    device_name                 TEXT     NOT NULL,

    -- Registration code: plaintext until claimed/expired, then nulled.
    -- Stored in full so the admin can retrieve and reshare it before use.
    registration_code           TEXT     UNIQUE,        -- nulled after claim
    device_api_key              TEXT,                   -- plaintext until claim, then nulled
    device_api_key_prefix       TEXT     NOT NULL,      -- kept after claim for admin reference

    -- Config payload delivered to the app on claim
    heartbeat_server_url        TEXT     NOT NULL,
    heartbeat_interval_seconds  INTEGER  NOT NULL,
    biometric_enabled           INTEGER  NOT NULL DEFAULT 0,
    biometric_user_can_toggle   INTEGER  NOT NULL DEFAULT 1,

    -- Lifecycle
    expires_at                  DATETIME NOT NULL,
    created_at                  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    used_at                     DATETIME,
    created_device_id           INTEGER  REFERENCES devices(id)
);

COMMIT;
