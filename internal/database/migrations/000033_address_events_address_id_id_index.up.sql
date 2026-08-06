BEGIN TRANSACTION;

-- The address-history read model classifies each event against the immediately
-- preceding event for the same address, via a correlated subquery that seeks on
-- address_id and then takes the highest id below the current one. The existing
-- idx_address_events_address_id_created_at serves the address_id seek but orders
-- by created_at, so SQLite sorts each address's slice per row (EXPLAIN QUERY PLAN
-- reports "USE TEMP B-TREE FOR ORDER BY" inside the correlated subquery). Ordering
-- this index by id instead lets the subquery stop at the first row it reads.
CREATE INDEX IF NOT EXISTS idx_address_events_address_id_id
    ON address_events (address_id, id DESC);

COMMIT;
