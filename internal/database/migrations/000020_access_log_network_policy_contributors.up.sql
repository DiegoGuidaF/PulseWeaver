PRAGMA foreign_keys = OFF;
BEGIN TRANSACTION;

CREATE TABLE access_log_network_policy_contributors (
    access_log_id INTEGER NOT NULL PRIMARY KEY REFERENCES access_log(id) ON DELETE CASCADE,
    policy_id     INTEGER REFERENCES network_policies(id) ON DELETE SET NULL,
    policy_name   TEXT    NOT NULL
);

COMMIT;
PRAGMA foreign_keys = ON;
