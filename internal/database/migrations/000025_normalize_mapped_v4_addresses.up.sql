BEGIN TRANSACTION;

-- Before PW-67 an address was stored in whatever representation the heartbeat
-- arrived in, so some rows may hold an IPv4-mapped IPv6 address (::ffff:a.b.c.d)
-- instead of the canonical dotted-decimal form the engine now keys on. Normalize
-- them. "::ffff:" is 7 characters, so the dotted-decimal tail starts at position 8.
--
-- This migration is pure data hygiene: the engine unmaps at cache-build time, so
-- correctness does not depend on it. On a fresh database both statements are no-ops.

-- 1. Drop a mapped row when the same device already has its canonical twin, so the
--    UPDATE below cannot violate the UNIQUE(device_id, ip) index.
DELETE FROM addresses
WHERE ip LIKE '::ffff:%.%.%.%'
  AND EXISTS (
      SELECT 1 FROM addresses b
      WHERE b.device_id = addresses.device_id
        AND b.ip = substr(addresses.ip, 8)
  );

-- 2. Normalize the remaining 4-in-6 rows to their plain IPv4 form in place.
UPDATE addresses
SET ip = substr(ip, 8)
WHERE ip LIKE '::ffff:%.%.%.%';

COMMIT;
