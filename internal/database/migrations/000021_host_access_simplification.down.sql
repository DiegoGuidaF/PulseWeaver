PRAGMA foreign_keys = OFF;
BEGIN TRANSACTION;

-- Reverse of 000021 up migration. Data dropped in the up migration is NOT restored.

-- ── Restore column and table names ────────────────────────────────────────────────────────────

ALTER TABLE user_host_settings RENAME COLUMN bypass_host_check TO bypass_host_allowlist;
ALTER TABLE network_policies RENAME COLUMN bypass_host_check TO allow_all_hosts;
ALTER TABLE network_policy_allowed_hosts RENAME COLUMN host_id TO known_host_id;
ALTER TABLE host_group_members RENAME COLUMN host_id TO known_host_id;
ALTER TABLE hosts RENAME TO known_hosts;

-- ── Restore icon column (nullable; no data to restore) ────────────────────────────────────────

ALTER TABLE known_hosts ADD COLUMN icon TEXT;

-- ── Recreate user_allowed_hosts (empty; data was dropped) ────────────────────────────────────

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

PRAGMA foreign_key_check;
COMMIT;
PRAGMA foreign_keys = ON;
