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

## Migration test seed runs at schema N-1

`internal/database/migration_test_seed.sql` is inserted *after* all migrations except the last one, then the final migration is re-applied on top. This means:

- **Do not include a newly added column in the seed row.** If your migration adds the column with a `DEFAULT`, let the migration apply that default to the seed row — don't provide it in the INSERT.
- After adding a migration, review the seed and check whether new NOT NULL constraints, CHECK constraints, or FK constraints require a new seed row or an update to an existing one (as stated in the seed file header).

---
**Verified against:** `scripts/check-migrations.sh`, `internal/database/migrations_test.go`, migrations `000012`–`000013`
**Applies to:** every new migration file
**Last verified:** 2026-04-16
