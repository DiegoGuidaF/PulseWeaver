-- Down migration for squashed initial schema.
-- Drops all application tables and indexes, preserving golang-migrate metadata.

-- Drop indexes (if they still exist)
DROP INDEX IF EXISTS idx_address_leases_device_id;
DROP INDEX IF EXISTS idx_address_leases_address_id;
DROP INDEX IF EXISTS idx_device_rules_device_rule;
DROP INDEX IF EXISTS idx_device_api_keys_key_hash;
DROP INDEX IF EXISTS idx_device_api_keys_device_id;
DROP INDEX IF EXISTS idx_sessions_user_id;
DROP INDEX IF EXISTS idx_sessions_token_hash;
DROP INDEX IF EXISTS idx_address_events_address_id_created_at;
DROP INDEX IF EXISTS idx_addresses_device_id_ip;
DROP INDEX IF EXISTS idx_address_device_id;
DROP INDEX IF EXISTS idx_devices_name_active;

-- Drop tables in dependency-safe order
DROP TABLE IF EXISTS address_leases;
DROP TABLE IF EXISTS device_rules;
DROP TABLE IF EXISTS address_current_state;
DROP TABLE IF EXISTS device_api_keys;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS address_events;
DROP TABLE IF EXISTS addresses;
DROP TABLE IF EXISTS devices;

