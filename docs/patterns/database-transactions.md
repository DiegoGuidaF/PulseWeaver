# Database Transactions

Transactions are coordinated through two structs in `internal/database`: `Transactor` (service-facing) and `DB` (repository-facing). The active transaction, if any, is carried in the `context.Context` — callers never hold a `*sqlx.Tx` directly.

## Two-handle design

| Handle | Holds | Used by | Why |
|--------|-------|---------|-----|
| `*database.Transactor` | `WithinTx` only | Services | Services orchestrate work but must not execute SQL |
| `*database.DB` | `WithinTx` + full query surface | Repositories | Repos need both to scope their own atomic flows |

This boundary is enforced by the type system: a service that holds only a `Transactor` cannot call `ExecContext`, `GetContext`, etc.

## How tx propagation works

`WithinTx` stores the active `*sqlx.Tx` in `ctx`. Every `DB` query method calls `d.exec(ctx)`, which returns the tx if one is present, or falls back to the pool. This means:

- **Nested `WithinTx` calls reuse the existing tx** — no savepoints, no new transaction.
- Repositories can call `WithinTx` safely even when the service has already opened a tx; they just join it.
- On `fn` error: the tx is rolled back. On success: it is committed. Panics trigger rollback before re-panicking.

## Service usage

Services declare a local `transactor` interface and hold it as a field:

```go
type transactor interface {
    WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type Service struct {
    repo repository
    tx   transactor
    // ...
}

func NewService(repo repository, tx transactor, logger *slog.Logger) *Service {
    return &Service{repo: repo, tx: tx}
}

func (s *Service) DisableAddress(ctx context.Context, deviceID DeviceID, addressID AddressID) (*Address, error) {
    var result *Address

    err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
        // All repo calls here share the same tx.
        if _, err := s.repo.GetDevice(ctx, deviceID); err != nil {
            return err
        }
        var err error
        result, err = s.repo.DisableAddress(ctx, addressID)
        return err
    })
    if err != nil {
        return nil, err
    }

    // Side effects (observers, logging) happen AFTER the tx commits.
    s.notifyObservers(ctx, NewAddressEvent(result, EventTypeAddressDisabled))
    return result, nil
}
```

**Key rule:** trigger observers and log success *after* `WithinTx` returns — the tx is not committed until then.

## Repository usage

Repositories use `r.db.WithinTx` when they need to make multiple writes atomic internally:

```go
type Repository struct {
    db *database.DB
}

func NewRepository(db *database.DB) *Repository {
    return &Repository{db: db}
}

func (r *Repository) DisableAddresses(ctx context.Context, ids []AddressID) ([]Address, error) {
    results := make([]Address, len(ids))

    err := r.db.WithinTx(ctx, func(ctx context.Context) error {
        for i, id := range ids {
            addr, err := r.disableOne(ctx, id)
            if err != nil {
                return fmt.Errorf("disable address %d: %w", id, err)
            }
            results[i] = *addr
        }
        return nil
    })
    if err != nil {
        return nil, err
    }
    return results, nil
}
```

If the service already opened a tx, `r.db.WithinTx` joins it — no nested transaction is started.

## Key rules

- **Services hold `Transactor`, repos hold `*database.DB`** — never pass `*database.DB` to a service.
- **Pass `ctx` through**: always forward the `ctx` returned by `WithinTx`'s callback so that nested calls see the tx.
- **No side effects inside `WithinTx`**: avoid observers, external HTTP calls, or logging success inside the tx closure. Do that after the call returns.
- **Error wrapping**: wrap errors from inside `WithinTx` with `fmt.Errorf("...: %w", err)` as usual; `WithinTx` propagates whatever `fn` returns.

---
**Verified against:** `internal/database/db.go`, `internal/database/transactor.go`, `internal/device/service.go`, `internal/device/repository.go`
**Applies to:** any service or repository that needs atomic multi-step operations
**Known gaps:** none
**Last verified:** 2026-04-17
