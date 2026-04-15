# Repository Layer

Repositories are the DB boundary. They translate between domain types and SQL, handle constraints, and provide transactions.

## Pattern

```
Service → Repository (domain types in/out) → sqlx → SQLite
```

- Repositories live in `internal/<domain>/repository.go`
- Each repository struct holds `*sqlx.DB`
- Return domain types, never `sql.Row` or raw maps
- Map DB errors to domain sentinel errors

## Scaffold

```go
type Repository struct {
    db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
    return &Repository{db: db}
}

func (r *Repository) GetDevice(ctx context.Context, id string) (*Device, error) {
    device := new(Device) // Go 1.26: use new(T) for zero-value pointer allocation
    query := `SELECT id, name, created_at FROM devices WHERE id = ? AND deleted_at IS NULL`
    if err := r.db.GetContext(ctx, device, query, id); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrNotFound
        }
        return nil, fmt.Errorf("get device: %w", err)
    }
    return device, nil
}

func (r *Repository) CreateDevice(ctx context.Context, d Device) (Device, error) {
    query := `INSERT INTO devices (id, name, created_at) VALUES (?, ?, ?)`
    _, err := r.db.ExecContext(ctx, query, d.ID, d.Name, d.CreatedAt)
    if err != nil {
        if isUniqueViolation(err) {
            return Device{}, ErrDeviceNameConflict
        }
        return Device{}, fmt.Errorf("create device: %w", err)
    }
    return d, nil
}
```

## Transactions (`RunInTx`)

For multi-step operations that must be atomic:

```go
func (r *Repository) RunInTx(ctx context.Context, fn func(tx *sqlx.Tx) error) error {
    tx, err := r.db.BeginTxx(ctx, nil)
    if err != nil {
        return fmt.Errorf("begin tx: %w", err)
    }
    if err := fn(tx); err != nil {
        _ = tx.Rollback()
        return err
    }
    return tx.Commit()
}
```

## Key rules

- **Error mapping**: check `sql.ErrNoRows` → `ErrNotFound`, unique constraint violations → `ErrConflict` / domain-specific error.
- **`new(T)` for scan targets**: use `new(Device)` not `&Device{}` for Go 1.26 compliance (see `pointer-conventions.md`).
- **Wrap errors**: always `fmt.Errorf("method name: %w", err)` for traceability.
- **No business logic**: repositories are pure data access. Validation belongs in services.
- **SQLite specifics**: WAL mode, `MaxOpenConns=1`, `_loc=auto` for timezone-aware timestamps.

---
**Verified against:** `internal/device/repository.go`, `internal/lease/repository.go`
**Applies to:** `internal/*/repository.go`
**Known gaps:** none
**Last verified:** 2026-04-15
