# Repository Tests (Integration)

Repository tests use real in-memory SQLite. They test CRUD, constraints, filters, pagination, and transaction rollback.

## Setup

```go
package device_test // black-box

func TestDeviceRepository_CreateAndGet(t *testing.T) {
    is := is.New(t)
    db := testdb.New(t) // fresh in-memory SQLite per test
    repo := device.NewRepository(db)

    // Given
    inserted := insertDevice(t, repo, "dev-1")

    // When
    got, err := repo.GetDevice(context.Background(), inserted.ID)

    // Then
    is.NoErr(err)
    is.Equal(got.Name, "dev-1")
}

func TestDeviceRepository_CreateDuplicate_ReturnsError(t *testing.T) {
    is := is.New(t)
    db := testdb.New(t)
    repo := device.NewRepository(db)
    insertDevice(t, repo, "dev-1")

    _, err := repo.CreateDevice(context.Background(), device.Device{Name: "dev-1"})

    is.True(errors.Is(err, device.ErrDeviceNameConflict))
}
```

## Seed helpers

```go
func insertDevice(t *testing.T, repo *device.Repository, name string) device.Device {
    t.Helper()
    d, err := repo.CreateDevice(context.Background(), device.Device{Name: name})
    if err != nil { t.Fatalf("insertDevice: %v", err) }
    return d
}
```

- Take only minimal fields — no optional parameters, no conditionals.
- Call sequentially in the Given block — don't chain helpers inside each other.

## Key rules

- **One top-level function per case** — `TestDeviceRepository_CreateAndGet`, not nested `t.Run`.
- **`testdb.New(t)`** — each test gets a fresh DB (no shared state).
- **Seed via public repo methods** — not direct SQL inserts.
- **`RunInTx` rollback test**: intentionally error inside the transaction, confirm the row doesn't exist.
- **Test constraint violations**: unique, FK, NOT NULL — using domain sentinel errors.

---
**Verified against:** `internal/device/repository_test.go`, `internal/testdb/`
**Applies to:** `internal/*/repository_test.go`
**Known gaps:** none
**Last verified:** 2026-04-15
