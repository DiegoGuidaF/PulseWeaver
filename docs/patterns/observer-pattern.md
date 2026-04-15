# Observer Pattern

Domain events are propagated using the observer pattern. The `device.Service` notifies registered observers synchronously when address state changes.

## Pattern

```
device.Service.AssignAddress()
  → notifyObservers(EventTypeAddressAssigned)
    → lease.Service.OnAddressEvent()   // non-blocking channel signal
    → policy.Service.OnAddressEvent()  // non-blocking channel signal
```

## Event types

```go
type EventType int
const (
    EventTypeAddressAssigned EventType = iota
    EventTypeAddressDisabled
)

type AddressEvent struct {
    Type     EventType
    DeviceID string
    Address  Address
}
```

## Observer interface

```go
// Declared in the device package (the producer)
type AddressObserver interface {
    OnAddressEvent(event AddressEvent)
}
```

## Implementing an observer

Observers use a **buffered channel (capacity 1)** to decouple the notification from processing. This means:
- `OnAddressEvent` never blocks the caller (non-blocking send)
- A dedicated `RunListener` goroutine processes signals
- Burst events are coalesced (channel acts as a "dirty flag")

```go
type Service struct {
    signal chan struct{}
    // ...
}

func NewService(...) *Service {
    return &Service{
        signal: make(chan struct{}, 1),
    }
}

func (s *Service) OnAddressEvent(event AddressEvent) {
    select {
    case s.signal <- struct{}{}:
    default: // already signaled, skip
    }
}

func (s *Service) RunListener(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return nil
        case <-s.signal:
            s.handleEvent(ctx)
        }
    }
}
```

## Registration (in `app.go`)

```go
deviceService.AddAddressObserver(leaseService)
deviceService.AddAddressObserver(policyService)
```

## Key rules

- **Non-blocking sends**: `OnAddressEvent` must never block. Always use select-with-default.
- **Consumer declares interface**: observer interfaces are declared in the producer package but designed around what consumers need.
- **Goroutine ownership**: each observer's `RunListener` is started by `app.go` and cancelled via context.

---
**Verified against:** `internal/device/events.go`, `internal/policy/service.go`, `internal/lease/service.go`
**Applies to:** any new domain that reacts to address changes
**Known gaps:** if a new event type is added beyond address events, the pattern is the same but the interface may need extending
**Last verified:** 2026-04-15
