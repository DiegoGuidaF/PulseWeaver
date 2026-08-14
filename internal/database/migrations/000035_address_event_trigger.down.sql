BEGIN TRANSACTION;

UPDATE address_events SET source = 'manual' WHERE source = 'web_ui';
UPDATE addresses SET source = 'manual' WHERE source = 'web_ui';

ALTER TABLE address_events DROP COLUMN trigger_type;

COMMIT;
