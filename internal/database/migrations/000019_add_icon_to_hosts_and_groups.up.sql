BEGIN TRANSACTION;

ALTER TABLE known_hosts ADD COLUMN icon TEXT;
ALTER TABLE host_groups ADD COLUMN icon TEXT;

CREATE INDEX idx_access_log_target_host ON access_log (target_host);

COMMIT;
