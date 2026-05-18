PRAGMA foreign_keys = OFF;
BEGIN TRANSACTION;

DROP TABLE network_policy_allowed_hosts;
DROP TABLE network_policy_allowed_host_groups;
DROP TABLE network_policies;

COMMIT;
PRAGMA foreign_keys = ON;
