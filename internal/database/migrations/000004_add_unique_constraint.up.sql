CREATE UNIQUE INDEX IF NOT EXISTS idx_addresses_device_id_ip ON addresses (device_id, ip);
