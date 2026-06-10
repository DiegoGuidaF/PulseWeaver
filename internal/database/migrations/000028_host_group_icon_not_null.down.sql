PRAGMA foreign_keys = OFF;

BEGIN TRANSACTION;

CREATE TABLE host_groups_old
(
    id          INTEGER PRIMARY KEY,
    name        TEXT     NOT NULL,
    description TEXT,
    icon        TEXT,
    color       TEXT     NOT NULL DEFAULT '#4C6EF5',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO host_groups_old (id, name, description, icon, color, created_at, updated_at)
    SELECT id, name, description, icon, color, created_at, updated_at
    FROM host_groups;

DROP TABLE host_groups;

ALTER TABLE host_groups_old RENAME TO host_groups;

CREATE UNIQUE INDEX idx_host_groups_name ON host_groups (name);

PRAGMA foreign_key_check;

COMMIT;

PRAGMA foreign_keys = ON;
