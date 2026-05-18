PRAGMA foreign_keys = OFF;
BEGIN TRANSACTION;

DROP TABLE network_policy_allowed_hosts;

COMMIT;
PRAGMA foreign_keys = ON;
