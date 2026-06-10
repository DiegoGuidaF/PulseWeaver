PRAGMA foreign_keys = OFF;

BEGIN TRANSACTION;

-- Backfill any NULL icons before adding the NOT NULL constraint.
UPDATE host_groups SET icon = 'server' WHERE icon IS NULL;

-- SQLite cannot add NOT NULL to an existing column via ALTER TABLE.
-- Use the create-copy-drop-rename pattern (foreign_keys must be OFF to
-- prevent child-table FK references from following the rename).
CREATE TABLE host_groups_new
(
    id          INTEGER PRIMARY KEY,
    name        TEXT     NOT NULL,
    description TEXT,
    icon        TEXT     NOT NULL DEFAULT 'server',
    color       TEXT     NOT NULL DEFAULT '#4C6EF5',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO host_groups_new (id, name, description, icon, color, created_at, updated_at)
    SELECT id, name, description, icon, color, created_at, updated_at
    FROM host_groups;

DROP TABLE host_groups;

ALTER TABLE host_groups_new RENAME TO host_groups;

CREATE UNIQUE INDEX idx_host_groups_name ON host_groups (name);

PRAGMA foreign_key_check;

COMMIT;

PRAGMA foreign_keys = ON;
