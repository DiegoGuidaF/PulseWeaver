CREATE TABLE addresses
(
    id         INTEGER PRIMARY KEY,
    device_id  INTEGER  NOT NULL REFERENCES devices (id) ON DELETE CASCADE,
    ip         TEXT     NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_address_device_id ON addresses (device_id);
CREATE UNIQUE INDEX idx_addresses_device_id_ip ON addresses (device_id, ip);