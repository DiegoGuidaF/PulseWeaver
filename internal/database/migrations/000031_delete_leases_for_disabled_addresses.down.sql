BEGIN TRANSACTION;

-- Irreversible data cleanup: the removed lease rows carried no meaningful expiry
-- (disabled addresses are out of lease scope) and cannot be reconstructed.

COMMIT;
