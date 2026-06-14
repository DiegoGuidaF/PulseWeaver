-- Seed data for TestMigrations_FinalMigration_WithData.
-- Targets the LATEST schema (N). The test seeds at N, rolls back to N-1 via the
-- down migration, then re-applies the up migration — verifying the round-trip.
-- Rows are ordered by FK dependency so foreign_keys=ON is satisfied.
--
-- When a new migration is added: update this seed so it inserts cleanly at the
-- new latest schema. Tables and columns added/removed by the migration must be
-- reflected here in the same commit.
--
-- Coverage goals:
--   - Every table has at least one row
--   - Both sides of CHECK constraints are represented (e.g. outcome 0 and 1)
--   - Nullable columns have both NULL and non-NULL examples where meaningful
--   - Soft-delete patterns (deleted_at) are represented

-- ── No FK dependencies ────────────────────────────────────────────────────────

-- An admin user (loggable; password_hash non-null)
INSERT INTO users (username, display_name, email, password_hash, role)
VALUES ('seed-admin', 'Seed Admin', 'seed-admin@example.com', X'DEADBEEF', 'admin');

-- A regular user (non-loggable; password_hash will be nulled by migration 000016)
INSERT INTO users (username, display_name, email, password_hash, role)
VALUES ('seed-user', 'Seed User', 'seed@example.com', X'DEADBEEF', 'user');

-- Active device
INSERT INTO devices (name, owner_id) VALUES ('seed-router', 1);
-- Soft-deleted device — migrations must handle both live and deleted rows
INSERT INTO devices (name, deleted_at, owner_id) VALUES ('seed-old-device', '2024-01-01 00:00:00', 1);

-- Both outcome values to exercise CHECK (outcome IN (0, 1));
-- one row with country attribution, one relying on the '' defaults
INSERT INTO hourly_traffic_aggregates
    (bucket_at, client_ip, target_host, outcome, deny_reason, request_count, country_code, country_name, continent_code)
VALUES ('2024-01-01 00:00:00', '10.0.0.1', 'example.com', 1, '', 5, 'DE', 'Germany', 'EU');

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

-- Native IPv6 address — exercises IPv6 storage/lookup (PW-67). Stored canonical
-- (unmapped), so migration 000025's 4-in-6 normalization must leave it untouched.
INSERT INTO addresses (device_id, ip, source, is_enabled)
    SELECT id, '2001:db8::1', 'manual', 1 FROM devices WHERE name = 'seed-router';

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

-- ── Device pairing ───────────────────────────────────────────────────────────

INSERT INTO device_pairings
    (device_id, pairing_code,
     heartbeat_server_url, heartbeat_interval_seconds, app_biometric_enabled, app_settings_locked,
     expires_at, created_at)
    SELECT d.id, 'code-abc',
           'https://pulse.example.com', 900, 0, 0,
           '2099-01-01 00:00:00', '2024-01-01 00:00:00'
    FROM devices d WHERE d.name = 'seed-router';

-- ── access_log (000018+: no device_id/address_id, uses contributor_count) ─────

-- Allow entry — contributor_count=1 (matching contributor row below)
INSERT INTO access_log
    (client_ip, outcome, contributor_count, xff_chain, target_host, target_uri, http_method, headers_json)
VALUES ('10.0.0.2', 1, 1, '10.0.0.1', 'example.com', '/api', 'GET', '{"X-Real-IP": "10.0.0.2"}');

-- Deny entry — no contributors, deny_reason set
INSERT INTO access_log
    (client_ip, outcome, contributor_count, xff_chain, target_host, target_uri, http_method, headers_json)
VALUES ('10.9.9.9', 0, 0, NULL, 'example.com', '/api', 'GET', '{}');

-- ── Depend on access_log ──────────────────────────────────────────────────────

INSERT INTO access_log_contributors (access_log_id, device_id, address_id, user_id)
    SELECT al.id, d.id, a.id, d.owner_id
    FROM access_log al, devices d
    JOIN addresses a ON a.device_id = d.id AND a.ip = '192.168.1.1'
    WHERE d.name = 'seed-router' AND al.outcome = 1;

INSERT INTO access_log_geoip (access_log_id, country_code, country_name, asn)
    SELECT id, 'US', 'United States', 1234 FROM access_log
    WHERE outcome = 1 LIMIT 1;

-- ── Host access control (000018+, simplified in 000021) ──────────────────────
-- These tables are new in migration 000018. Their data is lost on rollback
-- (down migration drops the tables) and the tables are recreated empty on
-- re-apply. This section ensures INSERT constraints are valid at schema N.

-- user_host_settings owns bypass_host_check; every active user needs a row.
INSERT INTO user_host_settings (user_id, bypass_host_check)
    SELECT id, 1 FROM users WHERE username = 'seed-admin';

INSERT INTO user_host_settings (user_id, bypass_host_check)
    SELECT id, 0 FROM users WHERE username = 'seed-user';

INSERT INTO hosts (fqdn) VALUES ('seed.example.com');
INSERT INTO hosts (fqdn) VALUES ('seed-no-fqdn.example.com');

INSERT INTO host_groups (name, description, icon, color) VALUES ('seed-group', 'Seed host group', 'folder', '#4C6EF5');
INSERT INTO host_groups (name, icon, color) VALUES ('seed-group-2', 'tag', '#7950F2');

INSERT INTO host_group_members (host_group_id, host_id)
    SELECT hg.id, h.id FROM host_groups hg, hosts h
    WHERE hg.name = 'seed-group' AND h.fqdn = 'seed.example.com';

INSERT INTO user_allowed_host_groups (user_id, host_group_id)
    SELECT u.id, hg.id FROM users u, host_groups hg
    WHERE u.username = 'seed-user' AND hg.name = 'seed-group';

INSERT INTO ignored_host_suggestions (fqdn) VALUES ('ignored.example.com');

-- ── Network Policies (000019+, column renamed in 000021) ──────────────────────

INSERT INTO network_policies (name, cidr, description, enabled, bypass_host_check)
VALUES ('Seed Home', '192.168.1.0/24', 'Seed home network', 1, 1);

INSERT INTO network_policies (name, cidr, enabled, bypass_host_check)
VALUES ('Seed VPN', '10.0.0.0/8', 0, 0);

INSERT INTO network_policy_allowed_host_groups (policy_id, host_group_id)
    SELECT np.id, hg.id FROM network_policies np, host_groups hg
    WHERE np.name = 'Seed VPN' AND hg.name = 'seed-group';

-- ── access_log_network_policy_contributors (000020+) ──────────────────────────

INSERT INTO access_log_network_policy_contributors (access_log_id, policy_id, policy_name)
    SELECT al.id, np.id, np.name
    FROM access_log al, network_policies np
    WHERE al.outcome = 0 AND np.name = 'Seed Home'
    LIMIT 1;

-- ── hourly_attribution_aggregates (000032+) ───────────────────────────────────
-- Rows for every entity_kind, covering both outcomes and both the live
-- (entity_id set) and deleted (entity_id NULL, name retained) paths — exercising
-- the CHECK, the entity_name key, and the post-deletion survival path.
INSERT INTO hourly_attribution_aggregates (bucket_at, entity_kind, entity_id, entity_name, outcome, request_count)
    SELECT '2024-01-01 00:00:00', 'policy', np.id, np.name, 1, 7
    FROM network_policies np WHERE np.name = 'Seed Home';

INSERT INTO hourly_attribution_aggregates (bucket_at, entity_kind, entity_id, entity_name, outcome, request_count)
    SELECT '2024-01-01 00:00:00', 'user', u.id, u.display_name, 0, 4
    FROM users u WHERE u.username = 'seed-user';

INSERT INTO hourly_attribution_aggregates (bucket_at, entity_kind, entity_id, entity_name, outcome, request_count)
    SELECT '2024-01-01 00:00:00', 'device', d.id, d.name, 1, 5
    FROM devices d WHERE d.name = 'seed-router';

INSERT INTO hourly_attribution_aggregates (bucket_at, entity_kind, entity_id, entity_name, outcome, request_count)
    VALUES ('2024-01-01 00:00:00', 'policy', NULL, 'Seed Deleted Policy', 0, 3);
