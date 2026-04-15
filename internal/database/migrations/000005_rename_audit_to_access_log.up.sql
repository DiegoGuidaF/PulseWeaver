BEGIN TRANSACTION;

ALTER TABLE request_audit_log RENAME TO access_log;
ALTER TABLE request_audit_log_geoip RENAME TO access_log_geoip;

COMMIT;
