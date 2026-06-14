BEGIN TRANSACTION;

-- A lease row now exists only while its address is enabled: disabling an address
-- deletes the row (see lease.ClearAddressLease) rather than nulling its expiry.
-- Rows written before that change still linger on already-disabled addresses, and
-- the device-wide expiry re-arm (SetDeviceAddressLeasesExpiry, run on every
-- lease-rule save) would resurrect them and "expire" the addresses again in a
-- batch. Drop those leftovers so the invariant holds for existing data too.
DELETE FROM address_leases
WHERE address_id IN (SELECT id FROM addresses WHERE is_enabled = 0);

COMMIT;
