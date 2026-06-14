BEGIN TRANSACTION;

-- Per-entity hourly allow/deny rollup serving the dashboard's attribution-split
-- widgets (network policy, user, device) for windows wider than
-- RawWindowThreshold. One table, one entity_kind discriminator per row: a
-- network policy, a user, and a device contributor are all the same shape — an
-- access_log row linked to the entity that matched it.
--
-- Kept separate from hourly_traffic_aggregates: client_ip/target_host/country
-- are columns on access_log (no fan-out, no entity), whereas these are
-- link-table attributions at a coarser grain. entity_name is denormalized so a
-- row still attributes traffic after its entity is deleted (entity_id then
-- nulls).
CREATE TABLE hourly_attribution_aggregates (
    bucket_at     DATETIME NOT NULL,   -- truncated to the hour, UTC
    entity_kind   TEXT     NOT NULL,   -- 'policy' | 'user' | 'device'
    entity_id     INTEGER,             -- nullable: entity hard-deleted
    entity_name   TEXT     NOT NULL,   -- denormalized, survives deletion
    outcome       INTEGER  NOT NULL CHECK (outcome IN (0, 1)), -- 1=allow, 0=deny
    request_count INTEGER  NOT NULL DEFAULT 0
);

-- Idempotent INSERT OR REPLACE key. Keyed on entity_name, not entity_id: a hard
-- delete nulls entity_id and SQLite treats every NULL as distinct, so an id key
-- would stop deduping a deleted entity's rows. entity_name is the stable
-- post-delete identifier (denormalized at rollup time).
CREATE UNIQUE INDEX idx_hourly_attr_aggregates_key
    ON hourly_attribution_aggregates (bucket_at, entity_kind, entity_name, outcome);

CREATE INDEX idx_hourly_attr_aggregates_bucket_at
    ON hourly_attribution_aggregates (bucket_at DESC);

COMMIT;
