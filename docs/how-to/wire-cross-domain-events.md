# How to: Wire cross-domain events (observer pattern)

Use this when Service B needs to react whenever Service A mutates its domain — e.g. policy cache refresh after host-access changes, lease expiry after rule changes.

The channel-based buffered signal lets observers coalesce bursts and never block the producer. See `observer-pattern.md` for the channel mechanics.

## Files you will touch

| Step | File |
|------|------|
| 1 | `internal/<producer>/events.go` (or `service.go`) — declare the Observer interface |
| 2 | `internal/<consumer>/service.go` — implement the interface + signal channel |
| 3 | `internal/<producer>/service.go` — add `AddObserver` + `notifyObservers` |
| 4 | `internal/app/app.go` — register observer; add `RunListener` goroutine |

## Step 1 — Declare the Observer interface in the producer package

The producer package owns the interface so the consumer imports the producer, not the other way around.

```go
// internal/myproducer/events.go
package myproducer

import "context"

type Observer interface {
    OnMyThingChanged(ctx context.Context)
}
```

## Step 2 — Implement the interface in the consumer

```go
// internal/myconsumer/service.go
package myconsumer

type Service struct {
    signal chan struct{}
    // ...
}

func NewService(...) *Service {
    return &Service{signal: make(chan struct{}, 1), ...}
}

// Implements myproducer.Observer
func (s *Service) OnMyThingChanged(_ context.Context) {
    select {
    case s.signal <- struct{}{}:
    default: // already a pending refresh — coalesce
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

If the consumer **already has a `RunListener`** for another signal, add a second channel and a second `case` in the existing `select` — do NOT add a second goroutine. `app.go` owns all goroutine lifecycles.

```go
// Adding a second signal source to an existing service:
type Service struct {
    addressSignal    chan struct{}
    myThingSignal    chan struct{}
}

func (s *Service) RunListener(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():      return nil
        case <-s.addressSignal: s.refresh(ctx)
        case <-s.myThingSignal: s.refresh(ctx)
        }
    }
}
```

## Step 3 — Add observer registration to the producer service

```go
// internal/myproducer/service.go
type Service struct {
    observers []Observer
    // ...
}

func (s *Service) AddObserver(o Observer) {
    if o != nil {
        s.observers = append(s.observers, o)
    }
}

func (s *Service) notifyObservers(ctx context.Context) {
    for _, o := range s.observers {
        o.OnMyThingChanged(ctx)
    }
}

// Call notifyObservers after every successful mutation:
func (s *Service) CreateMyThing(ctx context.Context, ...) (MyThing, error) {
    thing, err := s.repo.CreateMyThing(ctx, ...)
    if err != nil {
        return MyThing{}, err
    }
    s.notifyObservers(ctx)
    return thing, nil
}
```

## Step 4 — Wire in `app.go`

```go
// internal/app/app.go  — in NewWithConfigAndLogger:
myProducerService.AddObserver(myConsumerService)

// in App.Run:
g.Go(func() error {
    return ignoreContextCanceled(myConsumerService.RunListener(gCtx))
})
```

Both lines are needed: `AddObserver` links them; the goroutine in `Run` processes signals.

## Checklist

- [ ] Observer interface declared in the **producer** package
- [ ] Consumer's `OnXxx` method does a non-blocking send (`select { case s.signal <- …: default: }`)
- [ ] Consumer has a `RunListener(ctx) error` that reads the channel in a `select`
- [ ] `AddObserver` called before `Run` in `app.go`
- [ ] `RunListener` goroutine added to the `errgroup` in `App.Run`

---
**Related patterns:** `observer-pattern.md`, `background-service-lifecycle.md`
**Real examples:** `hostaccess/service.go` (producer), `policy/service.go` (consumer — two signals), `device/events.go`, `app/app.go`
**Last verified:** 2026-04-20
