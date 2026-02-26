-- Squashed initial schema for WallyDic.
-- Generated from a fully-migrated SQLite database.

-- Core device and address tables

CREATE TABLE IF NOT EXISTS "devices"
(
    id         INTEGER PRIMARY KEY,
    name       TEXT NOT NULL,
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
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_address_device_id
    ON addresses (device_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_addresses_device_id_ip
    ON addresses (device_id, ip);

CREATE TABLE IF NOT EXISTS address_status
(
    id         INTEGER PRIMARY KEY,
    address_id INTEGER  NOT NULL REFERENCES addresses(id) ON DELETE CASCADE,
    status     BOOLEAN  NOT NULL CHECK (status IN (0, 1)),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    source     TEXT     NOT NULL DEFAULT 'manual'
);

CREATE INDEX IF NOT EXISTS idx_address_status_address_id_created_at
    ON address_status (address_id, created_at DESC);

-- Authentication tables

CREATE TABLE IF NOT EXISTS users
(
    id            INTEGER PRIMARY KEY,
    username      TEXT      NOT NULL UNIQUE COLLATE NOCASE,
    display_name  TEXT      NOT NULL,
    email         TEXT UNIQUE,
    password_hash BLOB      NOT NULL,
    role          TEXT      NOT NULL DEFAULT 'user',
    created_by    INTEGER   REFERENCES users (id) ON DELETE SET NULL,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

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
    device_id  INTEGER  NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    key_prefix TEXT     NOT NULL,
    key_hash   TEXT     NOT NULL UNIQUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_device_api_keys_device_id
    ON device_api_keys(device_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_device_api_keys_key_hash
    ON device_api_keys(key_hash);

-- Address current state

CREATE TABLE IF NOT EXISTS address_current_state
(
    address_id INTEGER PRIMARY KEY REFERENCES addresses (id) ON DELETE CASCADE,
    is_enabled BOOLEAN  NOT NULL DEFAULT 1 CHECK (is_enabled IN (0, 1)),
    source     TEXT     NOT NULL DEFAULT 'manual',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

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
    address_id INTEGER  NOT NULL REFERENCES addresses (id) ON DELETE CASCADE,
    expires_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_address_leases_address_id_expires_at
    ON address_leases (address_id, expires_at);

