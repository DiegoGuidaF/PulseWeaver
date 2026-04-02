-- SQLite does not support DROP COLUMN; recreate the table without owner_id.
CREATE TABLE devices_new (
    id          INTEGER PRIMARY KEY,
    name        TEXT     NOT NULL,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at  DATETIME,
    device_type TEXT     NOT NULL DEFAULT 'static',
    description TEXT,
    icon        TEXT,
    updated_at  DATETIME NOT NULL DEFAULT '1970-01-01'
);

INSERT INTO devices_new SELECT id, name, created_at, deleted_at, device_type, description, icon, updated_at FROM devices;
DROP TABLE devices;
ALTER TABLE devices_new RENAME TO devices;

CREATE UNIQUE INDEX IF NOT EXISTS idx_devices_name_active ON devices (name) WHERE deleted_at IS NULL;
