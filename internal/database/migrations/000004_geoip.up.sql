CREATE TABLE IF NOT EXISTS request_audit_log_geoip (
    audit_log_id   INTEGER PRIMARY KEY
                   REFERENCES request_audit_log(id) ON DELETE CASCADE,
    country_code   TEXT,
    country_name   TEXT,
    continent_code TEXT,
    asn            INTEGER,
    asn_org        TEXT
);
