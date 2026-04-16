-- Seed data for TestMigrations_FinalMigration_WithData.
-- Inserted after Steps(-1) (schema at N-1), before the final migration is re-applied.
-- Rows are ordered by FK dependency so foreign_keys=ON is satisfied.
--
-- When a new migration is added: review whether the seed still covers all tables
-- and constraints affected by the migration (NOT NULL, CHECK, UNIQUE, FK, nullable
-- columns). Add or update rows below as needed.
--
-- Coverage goals:
--   - Every table has at least one row
--   - Both sides of CHECK constraints are represented (e.g. outcome 0 and 1)
--   - Nullable columns have both NULL and non-NULL examples where meaningful
--   - Soft-delete patterns (deleted_at) are represented

-- ── No FK dependencies ────────────────────────────────────────────────────────

-- A user
INSERT INTO users (username, display_name, email, password_hash, role)
VALUES ('seed-user', 'Seed User', 'seed@example.com', X'DEADBEEF', 'user');

-- Active device
INSERT INTO devices (name, owner_id) VALUES ('seed-router', 1);
-- Soft-deleted device — migrations must handle both live and deleted rows
INSERT INTO devices (name, deleted_at, owner_id) VALUES ('seed-old-device', '2024-01-01 00:00:00', 1);

-- Both outcome values to exercise CHECK (outcome IN (0, 1))
INSERT INTO hourly_traffic_aggregates
    (bucket_at, client_ip, target_host, outcome, deny_reason, request_count)
VALUES ('2024-01-01 00:00:00', '10.0.0.1', 'example.com', 1, '', 5);

INSERT INTO hourly_traffic_aggregates
    (bucket_at, client_ip, target_host, outcome, deny_reason, request_count)
VALUES ('2024-01-01 00:00:00', '10.0.0.2', 'example.com', 0, 'no_device', 2);

-- ── Depend on devices ─────────────────────────────────────────────────────────

-- Enabled address
INSERT INTO addresses (device_id, ip, source, is_enabled)
    SELECT id, '192.168.1.1', 'manual', 1 FROM devices WHERE name = 'seed-router';

-- Disabled address — exercises CHECK (is_enabled IN (0, 1)) with value 0
INSERT INTO addresses (device_id, ip, source, is_enabled)
    SELECT id, '192.168.1.2', 'heartbeat', 0 FROM devices WHERE name = 'seed-router';

INSERT INTO device_api_keys (device_id, key_prefix, key_hash)
    SELECT id, 'pw_test', 'hash-seed' FROM devices WHERE name = 'seed-router';

INSERT INTO device_rules (device_id, rule_type, config)
    SELECT id, 'max_ips', '{"max": 3}' FROM devices WHERE name = 'seed-router';

-- ── Depend on addresses ───────────────────────────────────────────────────────

-- Both is_enabled values to exercise CHECK (is_enabled IN (0, 1))
INSERT INTO address_events (address_id, is_enabled, source)
    SELECT id, 1, 'manual' FROM addresses WHERE ip = '192.168.1.1';

INSERT INTO address_events (address_id, is_enabled, source)
    SELECT id, 0, 'manual' FROM addresses WHERE ip = '192.168.1.2';

INSERT INTO address_leases (device_id, address_id)
    SELECT d.id, a.id FROM devices d
    JOIN addresses a ON a.device_id = d.id AND a.ip = '192.168.1.1'
    WHERE d.name = 'seed-router';

-- ── Depend on users ───────────────────────────────────────────────────────────

INSERT INTO sessions (user_id, token_hash, expires_at)
    SELECT id, X'CAFEBABE', '2099-01-01 00:00:00' FROM users WHERE username = 'seed-user';

-- ── Pending registration ──────────────────────────────────────────────────────

-- Unclaimed invite (registration_code and device_api_key present).
-- Uses column names from 000014 schema (app_biometric_enabled / app_settings_locked);
-- this seed runs at N-1 (= 000014); migration 000015 drops device_api_key and device_api_key_prefix.
INSERT INTO pending_registrations
    (id, device_name, owner_id, registration_code, device_api_key, device_api_key_prefix,
     heartbeat_server_url, heartbeat_interval_seconds, app_biometric_enabled, app_settings_locked,
     expires_at, created_at)
    SELECT 'seed-reg-01', 'seed-device', u.id, 'code-abc', 'raw-key-abc', 'pw_seed',
           'https://pulse.example.com', 900, 0, 0,
           '2099-01-01 00:00:00', '2024-01-01 00:00:00'
    FROM users u WHERE u.username = 'seed-user';

-- ── access_log: device_id/address_id nullable ─────────────────────────────────

-- Allow entry — with non-empty headers_json and xff_chain
INSERT INTO access_log
    (client_ip, outcome, device_id, address_id, xff_chain, target_host, target_uri, http_method, headers_json)
    SELECT '10.0.0.2', 1, d.id, a.id, '10.0.0.1', 'example.com', '/api', 'GET',
           '{"X-Real-IP": "10.0.0.2"}'
    FROM devices d JOIN addresses a ON a.device_id = d.id AND a.ip = '192.168.1.1'
    WHERE d.name = 'seed-router';

-- Deny entry — no device/address match, deny_reason set, nullable FKs are NULL
INSERT INTO access_log
    (client_ip, outcome, device_id, address_id, xff_chain, target_host, target_uri, http_method, headers_json)
VALUES ('10.9.9.9', 0, NULL, NULL, NULL, 'example.com', '/api', 'GET', '{}');

-- ── Depend on access_log ──────────────────────────────────────────────────────

INSERT INTO access_log_geoip (access_log_id, country_code, country_name, asn)
    SELECT id, 'US', 'United States', 1234 FROM access_log
    WHERE outcome = 1 LIMIT 1;
