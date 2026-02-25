-- Ensure address_status exists so this migration can run safely even if
-- previous migrations were partially rolled back.
DROP TABLE IF EXISTS address_current_state;

