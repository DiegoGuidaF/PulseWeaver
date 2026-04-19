# Migrations

## Transaction wrapping

Every migration file must explicitly manage its transaction:

```sql
BEGIN TRANSACTION;
-- your DDL/DML
COMMIT;
```

`make lint-back` (`check-migrations`) enforces this and will fail if either is missing. `make migrate-create` generates the file pair — write the SQL in between.

## SQLite: NOT NULL + REFERENCES via ALTER TABLE

SQLite rejects `ALTER TABLE ADD COLUMN NOT NULL DEFAULT x REFERENCES y(id)`. Use the copy-drop-rename table rebuild pattern instead (see migrations `000012`, `000013`).

## Migration test seed targets the latest schema (N)

`internal/database/migration_test_seed.sql` is inserted at **schema N** (the latest). The test then rolls back via the down migration and re-applies the up migration, verifying the round-trip doesn't crash or corrupt data.

This means:

- **Update the seed in the same commit as your migration.** If your migration adds tables, add seed rows. If it removes/renames columns, update the affected INSERTs. There is no deferred update — the seed must always match the latest schema.
- Tables introduced by migration N are dropped during rollback (they don't exist at N-1) and recreated empty on re-apply. This is expected — seed those tables for INSERT constraint coverage, not for round-trip survivability.
- After adding a migration, verify that `TestMigrations_FinalMigration_WithData` still passes and that the assertions cover the data your migration transforms.

---
**Verified against:** `scripts/check-migrations.sh`, `internal/database/migrations_test.go`, migrations `000012`–`000018`
**Applies to:** every new migration file
**Last verified:** 2026-04-19
