BEGIN TRANSACTION;

-- Irreversible by design: normalizing a 4-in-6 address to its canonical IPv4 form
-- is lossy (the original representation is not recorded) and re-mapping would serve
-- no purpose, since the engine treats both forms identically. No-op.

COMMIT;
