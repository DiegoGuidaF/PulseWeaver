BEGIN TRANSACTION;

DROP INDEX IF EXISTS idx_access_log_target_host;

-- SQLite does not support DROP COLUMN before 3.35; recreate tables without the icon column.

CREATE TABLE known_hosts_new
(
    id         INTEGER PRIMARY KEY,
    fqdn       TEXT     NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO known_hosts_new (id, fqdn, created_at)
    SELECT id, fqdn, created_at FROM known_hosts;

DROP TABLE known_hosts;
ALTER TABLE known_hosts_new RENAME TO known_hosts;

CREATE UNIQUE INDEX idx_known_hosts_fqdn ON known_hosts (fqdn);

CREATE TABLE host_groups_new
(
    id          INTEGER PRIMARY KEY,
    name        TEXT     NOT NULL,
    description TEXT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO host_groups_new (id, name, description, created_at)
    SELECT id, name, description, created_at FROM host_groups;

DROP TABLE host_groups;
ALTER TABLE host_groups_new RENAME TO host_groups;

CREATE UNIQUE INDEX idx_host_groups_name ON host_groups (name);

COMMIT;
