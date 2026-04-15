BEGIN TRANSACTION;

-- Core device and address tables

CREATE TABLE IF NOT EXISTS "devices"
(
    id         INTEGER PRIMARY KEY,
    name       TEXT     NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_devices_name_active
    ON "devices" (name) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS addresses
(
    id         INTEGER PRIMARY KEY,
    device_id  INTEGER  NOT NULL REFERENCES devices (id) ON DELETE CASCADE,
    ip         TEXT     NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_enabled BOOLEAN  NOT NULL DEFAULT 1 CHECK (is_enabled IN (0, 1)),
    source     TEXT     NOT NULL DEFAULT 'manual',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_address_device_id
    ON addresses (device_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_addresses_device_id_ip
    ON addresses (device_id, ip);

CREATE INDEX IF NOT EXISTS idx_addresses_is_enabled
    ON addresses (is_enabled);

CREATE TABLE IF NOT EXISTS address_events
(
    id         INTEGER PRIMARY KEY,
    address_id INTEGER  NOT NULL REFERENCES addresses (id) ON DELETE CASCADE,
    is_enabled BOOLEAN  NOT NULL CHECK (is_enabled IN (0, 1)),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    source     TEXT     NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_address_events_address_id_created_at
    ON address_events (address_id, created_at DESC);

-- Authentication tables

CREATE TABLE IF NOT EXISTS users
(
    id            INTEGER PRIMARY KEY,
    username      TEXT      NOT NULL COLLATE NOCASE,
    display_name  TEXT      NOT NULL,
    email         TEXT NOT NULL DEFAULT '',
    password_hash BLOB      NOT NULL,
    role          TEXT      NOT NULL DEFAULT 'user',
    must_change_password BOOLEAN NOT NULL DEFAULT 0 CHECK (must_change_password IN (0, 1)),
    created_by    INTEGER   REFERENCES users (id) ON DELETE SET NULL,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at    DATETIME
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username_active
    ON users (username COLLATE NOCASE) WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_active
    ON users (email) WHERE email != '' AND deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS sessions
(
    id           INTEGER PRIMARY KEY,
    user_id      INTEGER REFERENCES users (id) ON DELETE CASCADE,
    token_hash   BLOB     NOT NULL,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at   DATETIME NOT NULL,
    last_used_at DATETIME,
    revoked_at   DATETIME
);

CREATE INDEX IF NOT EXISTS idx_sessions_token_hash
    ON sessions (token_hash);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id
    ON sessions (user_id);

-- Device API keys

CREATE TABLE IF NOT EXISTS device_api_keys
(
    id         INTEGER PRIMARY KEY,
    device_id  INTEGER  NOT NULL REFERENCES devices (id) ON DELETE CASCADE,
    key_prefix TEXT     NOT NULL,
    key_hash   TEXT     NOT NULL UNIQUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_device_api_keys_device_id
    ON device_api_keys (device_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_device_api_keys_key_hash
    ON device_api_keys (key_hash);

-- Device rules

CREATE TABLE IF NOT EXISTS device_rules
(
    id         INTEGER PRIMARY KEY,
    device_id  INTEGER  NOT NULL REFERENCES devices (id) ON DELETE CASCADE,
    rule_type  TEXT     NOT NULL,
    enabled    BOOLEAN  NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
    config     TEXT     NOT NULL DEFAULT '{}',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_device_rules_device_rule
    ON device_rules (device_id, rule_type);

-- Address leases

CREATE TABLE IF NOT EXISTS address_leases
(
    id         INTEGER PRIMARY KEY,
    device_id  INTEGER  NOT NULL REFERENCES devices (id) ON DELETE CASCADE,
    address_id INTEGER  NOT NULL REFERENCES addresses (id) ON DELETE CASCADE,
    expires_at DATETIME,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_address_leases_device_id
    ON address_leases (device_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_address_leases_address_id
    ON address_leases (address_id);

-- Request audit log

CREATE TABLE IF NOT EXISTS request_audit_log (
    id           INTEGER PRIMARY KEY,
    client_ip    TEXT     NOT NULL,
    outcome      INTEGER  NOT NULL CHECK (outcome IN (0, 1)),
    deny_reason  TEXT,                                       -- NULL on allow
    device_id    INTEGER  REFERENCES devices(id) ON DELETE SET NULL,
    address_id   INTEGER  REFERENCES addresses(id) ON DELETE SET NULL,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    xff_chain    TEXT,
    target_host  TEXT,                                       -- X-Forwarded-Host
    target_uri   TEXT,                                       -- X-Forwarded-Uri
    http_method  TEXT,                                       -- X-Forwarded-Method
    headers_json TEXT     NOT NULL DEFAULT '{}'              -- JSON blob of enrichment headers
);

CREATE INDEX IF NOT EXISTS idx_request_audit_log_created_at
    ON request_audit_log (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_request_audit_log_client_ip
    ON request_audit_log (client_ip);

CREATE INDEX IF NOT EXISTS idx_request_audit_log_device_id
    ON request_audit_log (device_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_request_audit_log_outcome
    ON request_audit_log (outcome, created_at DESC);
COMMIT;
