BEGIN TRANSACTION;

-- The access-log histogram aggregates raw access_log at every window width, and
-- reads exactly two columns per row: created_at to bucket by, outcome to split
-- allow from deny. idx_access_log_created_at serves the range seek but carries
-- neither payload, so SQLite fetches the full row out of the table for every row
-- in the window — a month-wide chart pays a table lookup per request ever logged.
-- Carrying outcome in the index makes it covering (EXPLAIN QUERY PLAN reports
-- "USING COVERING INDEX"), so the scan never leaves the index.
--
-- idx_access_log_outcome (outcome, created_at DESC) cannot do this job: it leads
-- on outcome, so an unfiltered window cannot seek on it.
CREATE INDEX IF NOT EXISTS idx_access_log_created_at_outcome
    ON access_log (created_at, outcome);

COMMIT;
