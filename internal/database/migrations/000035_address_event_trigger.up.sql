BEGIN TRANSACTION;

ALTER TABLE address_events ADD COLUMN trigger_type TEXT NOT NULL DEFAULT 'schedule';

-- Every source but 'heartbeat' determines its trigger exactly, so derive those
-- and leave heartbeat rows on the column default: a scheduled beat is what the
-- overwhelming majority of them were.
-- Runs before the source rename below, which would otherwise erase 'manual'.
UPDATE address_events SET trigger_type = 'user' WHERE source = 'manual';
UPDATE address_events SET trigger_type = 'system' WHERE source IN ('expiry', 'limit_exceeded');

-- addresses.source keeps its DEFAULT 'manual' from 000001. Changing a default
-- needs a full table rebuild in SQLite, and every insert path names the column
-- explicitly, so the stale default is accepted rather than rebuilt for.
UPDATE address_events SET source = 'web_ui' WHERE source = 'manual';
UPDATE addresses SET source = 'web_ui' WHERE source = 'manual';

COMMIT;
