-- Create address_status table to track enable/disable events
-- Addresses are immutable, all status changes are recorded here
CREATE TABLE address_status
(
    id         INTEGER PRIMARY KEY,
    address_id INTEGER  NOT NULL REFERENCES addresses(id) ON DELETE CASCADE ,
    status     BOOLEAN  NOT NULL CHECK (status IN (0, 1)),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Composite index for efficiently querying latest status per address
-- The DESC ordering optimizes queries that need the most recent status
CREATE INDEX idx_address_status_address_id_created_at ON address_status (address_id, created_at DESC);
