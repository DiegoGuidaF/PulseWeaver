PRAGMA foreign_keys = OFF;
BEGIN TRANSACTION;

CREATE TABLE network_policies (
    id              INTEGER  PRIMARY KEY AUTOINCREMENT,
    name            TEXT     NOT NULL,
    cidr            TEXT     NOT NULL UNIQUE,
    description     TEXT,
    enabled         INTEGER  NOT NULL DEFAULT 1,
    allow_all_hosts INTEGER  NOT NULL DEFAULT 0,
    created_at      DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at      DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE network_policy_allowed_host_groups (
    policy_id     INTEGER NOT NULL REFERENCES network_policies(id) ON DELETE CASCADE,
    host_group_id INTEGER NOT NULL REFERENCES host_groups(id)      ON DELETE CASCADE,
    PRIMARY KEY (policy_id, host_group_id)
);

CREATE TABLE network_policy_allowed_hosts (
    policy_id     INTEGER NOT NULL REFERENCES network_policies(id) ON DELETE CASCADE,
    known_host_id INTEGER NOT NULL REFERENCES known_hosts(id)      ON DELETE CASCADE,
    PRIMARY KEY (policy_id, known_host_id)
);

COMMIT;
PRAGMA foreign_keys = ON;
