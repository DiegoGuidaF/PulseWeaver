-- Recreate devices table with deleted_at and partial unique index on (name) WHERE deleted_at IS NULL.
-- SQLite cannot drop UNIQUE from a column; table must be recreated.

CREATE TABLE devices_new
(
    id         INTEGER PRIMARY KEY,
    name       TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);

CREATE UNIQUE INDEX idx_devices_name_active ON devices_new (name) WHERE deleted_at IS NULL;

INSERT INTO devices_new (id, name, created_at, deleted_at)
SELECT id, name, created_at, NULL
FROM devices;

DROP TABLE devices;

ALTER TABLE devices_new RENAME TO devices;
