CREATE TABLE device_api_keys
(
    id         INTEGER PRIMARY KEY,
    device_id  INTEGER  NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    key_prefix TEXT     NOT NULL,
    key_hash   TEXT     NOT NULL UNIQUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX idx_device_api_keys_device_id ON device_api_keys(device_id);
CREATE UNIQUE INDEX idx_device_api_keys_key_hash ON device_api_keys(key_hash);
