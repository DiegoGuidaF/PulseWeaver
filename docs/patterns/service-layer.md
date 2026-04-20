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

## Input validation with Params structs

When a service method requires normalization (trim, lowercase) or deduplication of batch inputs before the repository call, encapsulate both in a `<Entity>Params` struct whose constructor validates on creation:

```go
// In params.go — same package as the service

var ErrBadRequest = errors.New("bad request")

type BulkCreateKnownHostsParams struct {
    FQDNs []string
}

func NewBulkCreateKnownHostsParams(fqdns []string) (BulkCreateKnownHostsParams, error) {
    if len(fqdns) == 0 {
        return BulkCreateKnownHostsParams{}, fmt.Errorf("%w: at least one FQDN required", ErrBadRequest)
    }
    // trim, lowercase, deduplicate...
    return BulkCreateKnownHostsParams{FQDNs: out}, nil
}
```

The service calls the constructor and propagates the error — it never validates the same input itself:

```go
func (s *Service) BulkCreateKnownHosts(ctx context.Context, fqdns []string) ([]KnownHost, error) {
    params, err := NewBulkCreateKnownHostsParams(fqdns)
    if err != nil {
        return nil, err  // ErrBadRequest propagates to handler
    }
    return s.repo.BulkCreateKnownHosts(ctx, params.FQDNs)
}
```

Rules:
- Only use a Params struct when normalization or deduplication is needed, not for simple field presence checks.
- Validation errors **must** wrap `ErrBadRequest` (defined in the same package) so the handler can detect them with `errors.Is`.
- Use this pattern for set-replacement operations (`SetHostGroupMembers`, `SetUserGrants`) where silently deduplicating IDs is the right contract.

Real example: `hostaccess/params.go` — `BulkCreateKnownHostsParams`, `SetHostGroupMembersParams`, `SetUserGrantsParams`.

## Key rules

- **Domain errors**: define sentinel errors (`var ErrNotFound = errors.New("not found")`) and return them. Handlers map these to HTTP status codes.
- **No HTTP awareness**: services never import `net/http` or `httpapi`.
- **No direct DB access**: always go through the repository interface.
- **Input types**: use dedicated input structs or Params constructors rather than accepting loose parameters. This makes validation and normalization explicit.

---
**Verified against:** `internal/device/service.go`, `internal/auth/service.go`, `internal/policy/service.go`
**Applies to:** `internal/*/service.go`
**Known gaps:** none
**Last verified:** 2026-04-15
