-- Revert to devices table with name TEXT NOT NULL UNIQUE (no deleted_at).
-- Deleted rows are dropped on rollback.

CREATE TABLE devices_new
(
    id         INTEGER PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO devices_new (id, name, created_at)
SELECT id, name, created_at
FROM devices
WHERE deleted_at IS NULL;

DROP TABLE devices;

ALTER TABLE devices_new RENAME TO devices;

CREATE INDEX idx_devices_name ON devices (name);
