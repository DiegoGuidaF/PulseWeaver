-- Remove null-valued entries from headers_json in request_audit_log.
-- Rows stored before the store-all-headers change may contain entries like
-- {"User-Agent": null, "Referer": null, ...} from the old allowlist approach.
-- This rebuilds each JSON object keeping only non-null entries.
UPDATE request_audit_log
SET headers_json = (
    SELECT COALESCE(
        json_group_object(key, json(value)),
        '{}'
    )
    FROM json_each(headers_json)
    WHERE value != 'null'
)
WHERE headers_json != '{}';
