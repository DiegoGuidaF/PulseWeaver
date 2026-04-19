PRAGMA foreign_keys = OFF;
BEGIN TRANSACTION;

CREATE TABLE devices_new
(
    id          INTEGER PRIMARY KEY,
    name        TEXT                               NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at  DATETIME,
    device_type TEXT     DEFAULT 'static'          NOT NULL,
    description TEXT,
    icon        TEXT,
    updated_at  DATETIME DEFAULT '1970-01-01'      NOT NULL,
    owner_id    INTEGER                            NOT NULL
        REFERENCES users
);

INSERT INTO devices_new
(id,
 name,
 created_at,
 deleted_at,
 device_type,
 description,
 icon,
 updated_at,
 owner_id)
SELECT id,
       name,
       created_at,
       deleted_at,
       device_type,
       description,
       icon,
       updated_at,
       owner_id
FROM devices;

DROP TABLE devices;
ALTER TABLE devices_new
    RENAME TO devices;

CREATE UNIQUE INDEX idx_devices_name_owner_active
    ON devices (name, owner_id)
    WHERE deleted_at IS NULL;

PRAGMA foreign_key_check;
COMMIT;
PRAGMA foreign_keys = ON;
