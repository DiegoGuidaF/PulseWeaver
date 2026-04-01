-- Default to 0 for prexisting data
ALTER TABLE access_log ADD COLUMN duration_us INTEGER NOT NULL default 0;