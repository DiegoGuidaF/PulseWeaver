ALTER TABLE address_status
    ADD COLUMN source TEXT NOT NULL DEFAULT 'manual';

CREATE TABLE address_current_state
(
    address_id INTEGER PRIMARY KEY REFERENCES addresses (id) ON DELETE CASCADE,
    is_enabled BOOLEAN  NOT NULL DEFAULT 1 CHECK (is_enabled IN (0, 1)),
    source     TEXT     NOT NULL DEFAULT 'manual',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

DROP VIEW IF EXISTS address_with_status;

