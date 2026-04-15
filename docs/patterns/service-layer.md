# Service Layer

Services hold all business logic. They receive and return domain types, never HTTP or SQL types.

## Pattern

```
Handler → Service (domain types in/out) → Repository
```

- Services live in `internal/<domain>/service.go`
- Each service struct holds its repository interface + logger + optional collaborators
- **Consumer declares the interface**: when service A needs service B, A declares an interface describing what it needs, and B implements it. This prevents import cycles and keeps dependencies explicit.

## Scaffold

```go
type repository interface {
    CreateDevice(ctx context.Context, d Device) (Device, error)
    GetDevice(ctx context.Context, id string) (*Device, error)
    // ... only the methods this service actually uses
}

type Service struct {
    repo   repository
    logger *slog.Logger
}

func NewService(repo repository, logger *slog.Logger) *Service {
    return &Service{repo: repo, logger: logger}
}

func (s *Service) CreateDevice(ctx context.Context, input CreateDeviceInput) (Device, error) {
    if input.Name == "" {
        return Device{}, ErrInvalidName
    }
    // Business logic here
    device := Device{
        ID:   generateID(),
        Name: input.Name,
    }
    return s.repo.CreateDevice(ctx, device)
}
```

## Cross-domain interfaces

When a service needs functionality from another domain, the **consuming** package declares the interface:

```go
// In policy/service.go
type EnabledIPsProvider interface {
    GetEnabledUniqueIPs(ctx context.Context) ([]string, error)
}

// In app.go wiring:
// policyService := policy.NewService(deviceService, ...) // deviceService implements EnabledIPsProvider
```

Real examples:
- `policy.EnabledIPsProvider` ← implemented by `*device.Service`
- `lease.TTLConfigRetriever` ← implemented by `*rule.Service`
- `scheduler.ExpiredAddressFinder` ← implemented by `*lease.Service`
- `scheduler.AddressDisabler` ← implemented by `*device.Service`

## Key rules

- **Domain errors**: define sentinel errors (`var ErrNotFound = errors.New("not found")`) and return them. Handlers map these to HTTP status codes.
- **No HTTP awareness**: services never import `net/http` or `httpapi`.
- **No direct DB access**: always go through the repository interface.
- **Input types**: use dedicated input structs (`CreateDeviceInput`) rather than accepting loose parameters. This makes validation explicit.

---
**Verified against:** `internal/device/service.go`, `internal/auth/service.go`, `internal/policy/service.go`
**Applies to:** `internal/*/service.go`
**Known gaps:** none
**Last verified:** 2026-04-15
