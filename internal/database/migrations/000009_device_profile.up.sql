ALTER TABLE devices
    ADD COLUMN device_type TEXT NOT NULL DEFAULT 'static';
ALTER TABLE devices
    ADD COLUMN description TEXT;
ALTER TABLE devices
    ADD COLUMN icon TEXT;
ALTER TABLE devices
    ADD COLUMN updated_at DATETIME NOT NULL DEFAULT '1970-01-01';
UPDATE devices SET updated_at = created_at;
