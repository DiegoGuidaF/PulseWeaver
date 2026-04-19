# Access Log Supplemental Tables

When `access_log` rows need optional detail data (data that is not present for every row), use a sparse child table rather than nullable columns on `access_log` itself.

## Pattern

```
access_log (primary row, always written)
  └─ access_log_<detail> (child rows, written only when data exists)
```

Existing instances:
- `access_log_geoip` — geo/ASN enrichment; written when the GeoIP resolver returns a result
- `access_log_ip_devices` (PW-38) — contributor list for shared-IP intersection; written when `ip_device_count > 1`

## Schema conventions

```sql
CREATE TABLE access_log_<detail> (
    id            INTEGER PRIMARY KEY,
    access_log_id INTEGER NOT NULL REFERENCES access_log(id) ON DELETE CASCADE,
    -- detail columns
);
CREATE INDEX idx_access_log_<detail>_log_id ON access_log_<detail>(access_log_id);
```

- Always `ON DELETE CASCADE` — detail rows have no meaning without the parent.
- Always index the FK for join/lookup performance.

## Fast-filter column

If the UI needs to filter access_log rows by whether the supplemental data exists, add a **denormalized count or flag column** to `access_log` rather than requiring a subquery:

```sql
ALTER TABLE access_log ADD COLUMN ip_device_count INTEGER NOT NULL DEFAULT 1;
```

This avoids a join in list queries. Only add this if the UI actually needs the filter; skip it for detail-only data like geoip where no "show only rows with geo" filter exists.

## Write path

Supplemental rows are written inside the **same transaction** as the parent `access_log` row in `accesslog.Repository.BatchInsert`. The `RETURNING id` from the parent insert gives the FK value.

```go
var accessID int64
r.db.GetContext(ctx, &accessID, insertAccessLog, ...)

if shouldWriteDetail(e) {
    r.db.ExecContext(ctx, insertDetail, accessID, ...)
}
```

## Key rules

- **Sparse by design**: only write child rows when data actually exists; do not insert empty/zero rows.
- **Same transaction**: parent and child rows must be atomic — use `r.db.WithinTx` wrapping the whole batch.
- **No nullable columns on `access_log`**: prefer a child table over adding a nullable column to the primary table.

---
**Verified against:** `internal/accesslog/repository.go`, migration `000018` (PW-38)
**Applies to:** any future `access_log_*` supplemental table
**Last verified:** 2026-04-19
