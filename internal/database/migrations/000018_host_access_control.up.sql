PRAGMA foreign_keys = OFF;
BEGIN TRANSACTION;

-- ── 1. Host-access user settings (bypass_host_allowlist lives here, not on users) ──

CREATE TABLE user_host_settings
(
    user_id               INTEGER PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    bypass_host_allowlist INTEGER NOT NULL DEFAULT 0
        CHECK (bypass_host_allowlist IN (0, 1))
);

-- Preserve unrestricted behaviour for all existing users.
INSERT INTO user_host_settings (user_id, bypass_host_allowlist)
SELECT id, 1
FROM users;

-- ── 2. Host-access domain tables ─────────────────────────────────────────────

CREATE TABLE known_hosts
(
    id         INTEGER PRIMARY KEY,
    fqdn       TEXT     NOT NULL,
    icon       TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_known_hosts_fqdn ON known_hosts (fqdn);

CREATE TABLE host_groups
(
    id          INTEGER PRIMARY KEY,
    name        TEXT     NOT NULL,
    description TEXT,
    icon        TEXT,
    color       TEXT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_host_groups_name ON host_groups (name);

CREATE TABLE host_group_members
(
    id            INTEGER PRIMARY KEY,
    host_group_id INTEGER  NOT NULL REFERENCES host_groups (id) ON DELETE CASCADE,
    known_host_id INTEGER  NOT NULL REFERENCES known_hosts (id) ON DELETE CASCADE,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_host_group_members_group_host
    ON host_group_members (host_group_id, known_host_id);

CREATE INDEX idx_host_group_members_known_host
    ON host_group_members (known_host_id);

CREATE TABLE user_allowed_hosts
(
    id            INTEGER PRIMARY KEY,
    user_id       INTEGER  NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    known_host_id INTEGER  NOT NULL REFERENCES known_hosts (id) ON DELETE CASCADE,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_user_allowed_hosts_user_host
    ON user_allowed_hosts (user_id, known_host_id);

CREATE INDEX idx_user_allowed_hosts_user
    ON user_allowed_hosts (user_id);

CREATE TABLE user_allowed_host_groups
(
    id            INTEGER PRIMARY KEY,
    user_id       INTEGER  NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    host_group_id INTEGER  NOT NULL REFERENCES host_groups (id) ON DELETE CASCADE,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_user_allowed_host_groups_user_group
    ON user_allowed_host_groups (user_id, host_group_id);

CREATE INDEX idx_user_allowed_host_groups_user
    ON user_allowed_host_groups (user_id);

CREATE TABLE ignored_host_suggestions
(
    id         INTEGER PRIMARY KEY,
    fqdn       TEXT     NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_ignored_host_suggestions_fqdn
    ON ignored_host_suggestions (fqdn);

-- ── 3. access_log_contributors (created before access_log rebuild) ────────────

CREATE TABLE access_log_contributors
(
    id            INTEGER PRIMARY KEY,
    access_log_id INTEGER NOT NULL REFERENCES access_log (id) ON DELETE CASCADE,
    device_id     INTEGER NOT NULL REFERENCES devices (id) ON DELETE CASCADE,
    address_id    INTEGER NOT NULL REFERENCES addresses (id) ON DELETE CASCADE,
    user_id       INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_access_log_contributors_log_id
    ON access_log_contributors (access_log_id);

-- ── 4. Migrate existing contributor data ──────────────────────────────────────
--   Rows with device_id IS NOT NULL get one contributor row.
--   user_id backfilled from devices.owner_id.

INSERT INTO access_log_contributors (access_log_id, device_id, address_id, user_id, created_at)
SELECT al.id, al.device_id, al.address_id, d.owner_id, al.created_at
FROM access_log al
         JOIN devices d ON d.id = al.device_id
WHERE al.device_id IS NOT NULL;

-- ── 5. Rebuild access_log: drop device_id/address_id, add contributor_count ───

CREATE TABLE access_log_new
(
    id                INTEGER PRIMARY KEY,
    client_ip         TEXT     NOT NULL,
    outcome           INTEGER  NOT NULL CHECK (outcome IN (0, 1)),
    deny_reason       TEXT,
    contributor_count INTEGER  NOT NULL DEFAULT 0,
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    xff_chain         TEXT,
    target_host       TEXT,
    target_uri        TEXT,
    http_method       TEXT,
    headers_json      TEXT     NOT NULL DEFAULT '{}',
    duration_us       INTEGER  NOT NULL DEFAULT 0
);

INSERT INTO access_log_new
(id, client_ip, outcome, deny_reason, contributor_count,
 created_at, xff_chain, target_host, target_uri, http_method, headers_json, duration_us)
SELECT id,
       client_ip,
       outcome,
       deny_reason,
       CASE WHEN device_id IS NOT NULL THEN 1 ELSE 0 END,
       created_at,
       xff_chain,
       target_host,
       target_uri,
       http_method,
       headers_json,
       duration_us
FROM access_log;

DROP TABLE access_log;
ALTER TABLE access_log_new
    RENAME TO access_log;

-- Recreate indexes (old names were idx_request_audit_log_* pre-rename in 000005).
CREATE INDEX idx_access_log_created_at ON access_log (created_at DESC);
CREATE INDEX idx_access_log_client_ip ON access_log (client_ip);
CREATE INDEX idx_access_log_outcome ON access_log (outcome, created_at DESC);

PRAGMA foreign_key_check;
COMMIT;
PRAGMA foreign_keys = ON;
