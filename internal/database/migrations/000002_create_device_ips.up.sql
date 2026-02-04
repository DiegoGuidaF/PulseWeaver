CREATE TABLE device_ips
(
    id          INTEGER PRIMARY KEY,
    device_id   INTEGER  NOT NULL,
    ip_address  TEXT     NOT NULL,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    disabled_at DATETIME,
    FOREIGN KEY (device_id) REFERENCES devices (id) ON DELETE CASCADE
);

CREATE INDEX idx_device_ips_device_id ON device_ips (device_id);
CREATE INDEX idx_device_ips_active ON device_ips (device_id, disabled_at);