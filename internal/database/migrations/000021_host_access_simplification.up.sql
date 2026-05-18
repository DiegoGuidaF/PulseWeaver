PRAGMA foreign_keys = OFF;
BEGIN TRANSACTION;

-- ── 1. Drop user_allowed_hosts (direct per-user host assignments replaced by group-only model) ──

DROP TABLE user_allowed_hosts;

-- ── 2. Remove icon column from known_hosts ────────────────────────────────────────────────────

ALTER TABLE known_hosts DROP COLUMN icon;

-- ── 3. Rename known_hosts → hosts ────────────────────────────────────────────────────────────
-- SQLite (3.26+) auto-updates FK target references in child tables on RENAME TO when
-- legacy_alter_table = OFF (the default). Both host_group_members and
-- network_policy_allowed_hosts FK targets are updated automatically.

ALTER TABLE known_hosts RENAME TO hosts;

-- ── 4. Rename known_host_id → host_id in M2M tables ──────────────────────────────────────────

ALTER TABLE host_group_members RENAME COLUMN known_host_id TO host_id;
ALTER TABLE network_policy_allowed_hosts RENAME COLUMN known_host_id TO host_id;

-- ── 5. Rename allow_all_hosts → bypass_host_check in network_policies ────────────────────────

ALTER TABLE network_policies RENAME COLUMN allow_all_hosts TO bypass_host_check;

-- ── 6. Rename bypass_host_allowlist → bypass_host_check in user_host_settings ───────────────

ALTER TABLE user_host_settings RENAME COLUMN bypass_host_allowlist TO bypass_host_check;

-- ── 7. Preserve unrestricted behaviour for all existing records ───────────────────────────────
-- Prevent any service interruption: set bypass_host_check = 1 for all users and policies so
-- that access continues working until host group assignments are reconfigured.

UPDATE user_host_settings SET bypass_host_check = 1;
UPDATE network_policies SET bypass_host_check = 1;

PRAGMA foreign_key_check;
COMMIT;
PRAGMA foreign_keys = ON;
