BEGIN TRANSACTION;

ALTER TABLE devices ADD COLUMN owner_id INTEGER REFERENCES users(id);

UPDATE devices
SET owner_id = (SELECT id FROM users WHERE role = 'admin' ORDER BY id LIMIT 1);

COMMIT;
