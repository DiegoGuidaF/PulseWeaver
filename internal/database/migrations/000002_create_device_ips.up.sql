CREATE TABLE addresses
(
    id          INTEGER PRIMARY KEY,
    device_id   INTEGER  NOT NULL,
    ip          TEXT     NOT NULL,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    disabled_at DATETIME,
    FOREIGN KEY (device_id) REFERENCES devices (id) ON DELETE CASCADE
);

CREATE INDEX idx_address_device_id ON addresses (device_id);