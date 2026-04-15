# Background Service Lifecycle

All long-running services (listeners, schedulers, cache refreshers) follow the same `Run(ctx) error` pattern.

## Pattern

```go
func (s *Service) Run(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return nil // clean shutdown
        case <-s.signal:
            s.process(ctx)
        }
    }
}
```

Alternatively, for periodic tasks:

```go
func RunSchedule(ctx context.Context, interval time.Duration, finder ExpiredAddressFinder, disabler AddressDisabler) error {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return nil
        case <-ticker.C:
            executeTask(ctx, finder, disabler)
        }
    }
}
```

## Lifecycle rules

1. **Run until `ctx` is cancelled** — return `nil` on cancellation, real error only on unexpected failure.
2. **Started by `app.go`** — each service is wrapped in a `wg.Add(1)` goroutine.
3. **Stopped by `App.Close()`** — cancels the context, then `wg.Wait()` for all goroutines.
4. **No self-managed goroutines** — services don't spawn their own goroutines. `app.go` owns all goroutine lifecycles.

## Wiring in `app.go`

```go
wg.Add(1)
go func() {
    defer wg.Done()
    if err := policyService.RunListener(ctx); err != nil {
        logger.Error("policy listener error", "error", err)
    }
}()
```

## Real examples

| Service | Method | Trigger |
|---------|--------|---------|
| `policy.Service` | `RunListener` | Channel signal from address events → full cache refresh |
| `lease.Service` | `RunListener` | Channel signal from address events → create/delete leases |
| `scheduler` | `RunSchedule` | Ticker at `RULE_CHECK_INTERVAL` → expire stale leases |

---
**Verified against:** `internal/policy/service.go`, `internal/lease/service.go`, `internal/scheduler/service.go`, `internal/app/app.go`
**Applies to:** any new long-running background service
**Known gaps:** none
**Last verified:** 2026-04-15
