PRAGMA foreign_keys = OFF;
BEGIN TRANSACTION;

-- Recreates network_policy_allowed_hosts with the post-000021 schema (host_id, not known_host_id).
CREATE TABLE network_policy_allowed_hosts (
    policy_id INTEGER NOT NULL REFERENCES network_policies(id) ON DELETE CASCADE,
    host_id   INTEGER NOT NULL REFERENCES hosts(id)            ON DELETE CASCADE,
    PRIMARY KEY (policy_id, host_id)
);

COMMIT;
PRAGMA foreign_keys = ON;
