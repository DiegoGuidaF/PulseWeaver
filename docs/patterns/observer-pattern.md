# Observer Pattern

Domain events are propagated using the observer pattern. The `device.Service` notifies registered observers synchronously when address state changes.

## Pattern

```
device.Service.AssignAddress()
  → notifyObservers(EventTypeAddressAssigned)
    → lease.Service.OnAddressEvent()   // non-blocking channel signal
    → policy.Service.OnAddressEvent()  // non-blocking channel signal
```

## Observer interface

```go
// Declared in the producer package (device)
type AddressObserver interface {
    OnAddressEvent(ctx context.Context, event AddressEvent)
}
```

## Implementing an observer

Observers use a **buffered channel (capacity 1)** to decouple the notification from processing:
- `OnAddressEvent` never blocks the caller (non-blocking send)
- A dedicated `RunListener` goroutine processes signals
- Burst events are coalesced (channel acts as a "dirty flag" — only one pending refresh at a time)

```go
type Service struct {
    signal chan struct{}
}

func NewService(...) *Service {
    return &Service{signal: make(chan struct{}, 1)}
}

func (s *Service) OnAddressEvent(_ context.Context, e AddressEvent) {
    select {
    case s.signal <- struct{}{}:
    default:
    }
}

func (s *Service) RunListener(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return nil
        case <-s.signal:
            s.refresh(ctx)
        }
    }
}
```

## Multiple signal sources in one RunListener

When a service reacts to more than one independent observer domain, add a second signal channel and select on both in the same `RunListener`. Do **not** add a second goroutine — app.go owns all goroutine lifecycles.

```go
type Service struct {
    addressSignal   chan struct{}
    hostAccessSignal chan struct{}
}

// Implements device.AddressObserver
func (s *Service) OnAddressEvent(_ context.Context, _ device.AddressEvent) {
    select { case s.addressSignal <- struct{}{}: default: }
}

// Implements hostaccess.Observer
func (s *Service) OnHostAccessChanged(_ context.Context) {
    select { case s.hostAccessSignal <- struct{}{}: default: }
}

func (s *Service) RunListener(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return nil
        case <-s.addressSignal:
            s.refresh(ctx)
        case <-s.hostAccessSignal:
            s.refresh(ctx)
        }
    }
}
```

Both signals trigger the same refresh — the coalescing property is preserved per channel.

## Registration (in `app.go`)

```go
deviceService.AddAddressObserver(leaseService)
deviceService.AddAddressObserver(policyService)
hostaccessService.AddObserver(policyService)
```

## Key rules

- **Non-blocking sends**: observer methods must never block. Always use select-with-default.
- **Producer declares interface**: observer interfaces live in the producer package.
- **Goroutine ownership**: each observer's `RunListener` is started by `app.go` and cancelled via context.
- **One goroutine per service**: multiple signal sources → multiple channels in one `RunListener`, not multiple goroutines.

---
**Verified against:** `internal/device/events.go`, `internal/policy/service.go`, `internal/lease/service.go`
**Applies to:** any new domain that reacts to address or domain-entity changes
**Last verified:** 2026-04-19
