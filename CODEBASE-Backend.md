# Backend Codebase Reference

> Last updated: 2026-03-18

## Directory Structure

```
internal/
├── app/            # Dependency injection and application wiring
├── auth/           # User authentication (sessions, bcrypt, principals)
├── policy/          # Forward-auth sidecar (IP allow/deny, in-memory cache)
├── config/         # Env var parsing (caarlos0/env)
├── database/       # SQLite connection, WAL mode, migrations
├── accesslog/      # Request access log (sink, service, repository)
├── device/         # Device and address management (core domain)
├── health/         # GET /health handler
├── httpapi/        # oapi-codegen generated types and strict handler interface
├── httpserver/     # Chi router, middleware chain, route registration
├── lease/          # Address lease TTL management
├── logging/        # slog helpers, context logger, short IDs
├── queries/        # Read-only query endpoints (devices+addresses, audit log)
├── rule/           # Device rule management (currently: address lease TTL rules)
├── scheduler/      # Periodic background tasks (auto-expiry of leases)
├── testdb/         # In-memory SQLite setup for integration tests
├── testutils/      # Integration server factory (SetupIntegrationServer)
└── ui/             # embed.FS SPA serving (prod) / dev stub (dev)
```

---

## Package Responsibilities

### Domain Packages

**`device`** — Core domain. Manages devices and their IP addresses.
- `Device` — name, created/deleted timestamps, API key prefix
- `Address` — IP string, `is_enabled` bool, status source (`heartbeat` | `manual` | `expiry`), optional lease expiry
- `Service.AssignAddress` — creates or re-enables an address; fires `EventTypeAddressAssigned`
- `Service.DisableAddress` / `DisableAddresses` — fires `EventTypeAddressDisabled`
- `Service.GetEnabledUniqueIPs` — `SELECT DISTINCT ip WHERE is_enabled = 1`; consumed by `policy`
- `Service.AddAddressObserver` — registers listeners for address events (`AddressObserver` interface)
- API key auth: `wdk_<base64>` prefix; SHA256 hash stored in DB
- Device principal context: `PrincipalDeviceContextMiddleware` / `PrincipalFromContext`

**`auth`** — User authentication (session cookies).
- `User` — bcrypt password, role (`admin` | `user`)
- `Session` — SHA256 token hash, 7-day TTL, revocation support
- `Service.BootstrapAdmin` — auto-creates admin user from `ADMIN_PASSWORD` on first startup
- Cookie: `__Host-wdc_session` (HTTP-only)
- User principal context: `PrincipalUserContextMiddleware` / `PrincipalFromContext`

**`policy`** — Forward-auth sidecar. Answers "is this IP enabled?" without a DB round-trip.
- Maintains `map[string]struct{}` (enabled IPs) under `sync.RWMutex`
- `Initialize(ctx)` — full `GetEnabledUniqueIPs` query at startup
- `OnAddressEvent` — non-blocking signal to buffered channel (cap 1); coalesces bursts
- `RunListener(ctx)` — goroutine; calls `refreshCache` (full DB query) on each signal
- `HandleForwardAuthIP` — `GET /api/policy-engine/verify-ip`; Bearer token + `X-Real-IP`; returns 200 or 403; fail-closed on any missing/invalid input
- Secret: `POLICY_ENGINE_API_SECRET` env var (minimum 32 chars)

**`lease`** — Address lease TTL. Automatically disables addresses when their lease expires.
- `AddressLease` — links an `AddressID` to an `ExpiresAt` timestamp
- `Service.AddAddressLease` — creates/updates lease using TTL from `rule.Service`
- `Service.RunListener` — goroutine; on `EventTypeAddressAssigned` → create lease; on `EventTypeAddressDisabled` → delete lease
- No lease is created if the device has no active lease rule

**`rule`** — Device-level rules (currently one type: address lease TTL).
- `DeviceAddressLeaseRule` — per-device TTL in seconds, enabled/disabled flag
- Config stored as JSON blob in `rules` table; parsed into typed structs
- `Service.GetDeviceAddressLeaseTTLSeconds` — consumed by `lease.Service`

**`accessLog`** — Request access logging.
- `RequestLog` — structured log of a single API request (method, path, status, duration, principal)
- `Sink` — `DecisionObserver`; receives `policy.DecisionEvent` and persists via `Repository`
- `Service` — business logic for creating and querying audit logs
- `HTTPHandler` — no direct endpoints; access log data exposed via `queries` package
- Repository writes to `access_logs` table

**`queries`** — Read-only query endpoints. Aggregates data across domains for list/filter views.
- `DeviceView`, `AddressView`, `AccessLogView` — read-model types joining multiple tables
- `HTTPHandler` — list endpoints: devices with addresses, access log entries (pagination, filters)
- `Repository` — SQL SELECT only; no writes; no transactions needed

**`health`** — Simple `GET /health` handler returning `{"status":"ok","timestamp":"..."}`.

### Infrastructure Packages

**`app`** — Wires everything together in `NewWithConfigAndLogger`.
- Construction order: DB → auth → device → policy → rule → lease → scheduler → HTTP server
- Observer registration: `deviceService.AddAddressObserver(addressLeaseService)` then `policy`
- Goroutines started: `policy.RunListener`, `lease.RunListener`, `scheduler.RunSchedule`
- `App` struct exposes `DeviceService`, `AuthService`, `PolicyService` for test access

**`config`** — Env var parsing via `caarlos0/env/v11`. Optional `.env` file (godotenv).
- `ConfServer`: `ADMIN_PASSWORD` (required), `SERVER_PORT`, `TRUSTED_PROXY`, `TZ`
- `ConfDB`: `DB_DIR` (default ./data, write access validated)
- `ConfRules`: `RULE_CHECK_INTERVAL` (default 1m)
- `ConfPolicy`: `POLICY_ENGINE_API_SECRET` (minimum 32 chars, validated in `Load()`)

**`database`** — Single SQLite connection (sqlx). WAL mode, `MaxOpenConns=1`.
- `NewSQLite(conf)` — applies pragmas and runs `db.Migrate()`
- Migrations embedded from `internal/database/migrations/` via `embed.FS`

**`httpserver`** — Chi router assembly.
- `NewServer`: global middleware chain: RequestID → slog-chi → Recoverer → ClientIP → security headers → MaxBodySize (256 KB)
- `addRoutes`: registers `/health`, `/api/policy-engine/verify-ip` (no OpenAPI validation), then `/api/v1` sub-router
- `/api/v1` sub-router adds: `LoginRateLimitMiddleware` → OpenAPI validator → `PrincipalUserContextMiddleware` → `PrincipalDeviceContextMiddleware` → generated strict handler
- `ClientIPFromXFFHeaderMiddleware` used when `TRUSTED_PROXY` is set; otherwise `ClientIPFromRequestMiddleware`

**`httpapi`** — Generated by `oapi-codegen` from `api/openapi.yaml`. Do not modify.
- `server.gen.go` — all DTOs, `StrictServerInterface`, route dispatch
- `const.go` — `SessionCookieName`, `APIKeyHeaderName`, `CookieAuthScope`, `APIKeyAuthScope`

**`scheduler`** — Periodic background task runner.
- `RunSchedule(ctx, interval)` — ticks at `interval`; runs `executeAutoExpiry`
- `executeAutoExpiry` — calls `lease.GetExpiredAddressIDs()` → `device.DisableAddresses()`

**`logging`** — slog helpers.
- `FromCtx(ctx)` — retrieves logger from context (falls back to `slog.Default()`)
- `Enrich(ctx, attrs...)` — returns context with enriched logger
- `WithRequestID(ctx, id)` — injects request ID into logger
- `AttrKeyComponent`, `AttrKeyError`, `AttrKeyOperation` — shared log attribute key constants

**`testdb`** — In-memory SQLite for integration tests (`file::memory:?_loc=auto`).

**`testutils`** — `SetupIntegrationServer(t)` — builds a full `*app.App` against in-memory DB, registers `t.Cleanup` for graceful shutdown. Used by all handler integration tests.

**`ui`** — `embed.FS` SPA serving.
- `ui_prod.go` (build tag `prod`): embeds `dist/`, serves assets with long cache headers, falls back to `index.html` for client-side routing
- `ui_dev.go` (build tag `!prod`): returns 404 pointing to Vite dev server
- `/api` prefix → 404 immediately (prevents API fallthrough)

---

## Key Patterns

### Layered Architecture
```
HTTP Handler → Service → Repository → Database
```
- Handlers extract primitives from OpenAPI DTOs; never pass generated types deeper
- Services hold business logic; receive domain types; return domain types
- Repositories interface over sqlx; map DB errors to domain errors; implement `RunInTx`

### Observer Pattern (Address Events)
`device.Service` notifies registered `AddressObserver`s synchronously on every address state change:
```
device.Service.AssignAddress()
  → notifyObservers(EventTypeAddressAssigned)
    → lease.Service.OnAddressEvent()   // non-blocking channel signal
    → policy.Service.OnAddressEvent()   // non-blocking channel signal
```
Both observers use a buffered channel (cap 1) + dedicated goroutine (`RunListener`) to process signals asynchronously. **Consumer declares the interface; owner implements.**

### Service Lifecycle
All long-running services follow `Run(ctx) error`:
- Run until `ctx` is cancelled
- Return `nil` on cancellation, or a real error if something unexpected happened
- `app.go` wraps each in a `wg.Add(1)` goroutine; `Close()` calls `wg.Wait()`

### Schema-First API
`api/openapi.yaml` is the single source of truth:
- `make api` runs `go generate ./...` (oapi-codegen) + frontend `generate:api`
- Never modify `internal/httpapi/server.gen.go` or `frontend/src/lib/api/` directly

### Authentication
Two independent schemes:
1. **Session cookie** (`__Host-wdc_session`): UI users; validated by `PrincipalUserContextMiddleware`
2. **API key header** (`X-API-Key`): device heartbeats; validated by `PrincipalDeviceContextMiddleware`

Both are OpenAPI `securitySchemes`; the `AuthenticationFunc` in `httpserver/authentication.go` wires them to the OpenAPI validator.

### Config Pattern
All config via env vars loaded at startup into `config.Conf`. The struct is passed into constructors — never accessed globally. Test helpers bypass `Load()` by constructing `*config.Conf` directly.

### Pointer Allocation (Go 1.26)
Use `new(T)` instead of a pointer to a local variable when allocating a zero-value struct that will be returned or passed as a pointer:

```go
// correct
user := new(User)
if err := r.db.GetContext(ctx, user, query, id); err != nil { ... }
return user, nil

// incorrect — triggers Go 1.26 lint warning
user := &User{}
// also incorrect
var user User
return &user, nil
```

The rule applies whenever the struct is initialised to its zero value — i.e. no fields are set at the declaration site. Composite literals with field values (`&User{Name: "alice"}`) stay as-is.

### When to use pointers (general guide)

**Receivers** — always use `*T` when the type has a channel, mutex, connection, or any field that must not be copied (e.g. `*Service`, `*Repository`, `*HTTPHandler`). Value receivers are fine only for small, purely data structs with no mutable state.

**Return values** — return `*T` when:
- The struct is filled by a DB scanner (`new(T)` pattern above).
- The value is optional and `nil` is a meaningful "absent" state (distinct from the zero value).

Return `T` by value when the function either validates input and returns a domain object (e.g. constructors like `NewDeviceAddressLeaseConfig`) or converts between representations. If the call site would immediately dereference the pointer (`*result`), that is a signal the function should return by value instead.

**Parameters** — pass `T` by value for small structs (≤ ~64 bytes) that the function only reads. Pass `*T` only when the function must mutate the argument, or when the struct is large enough that copying on every call is measurable. Passing a pointer to a tiny config struct (e.g. `*DeviceAddressLeaseConfig` with one `int` field) adds a heap allocation with no benefit.

**Optional fields** — use `*T` for struct fields that are genuinely nullable (e.g. `DenyReason *DenyReason`, `TTLSeconds *int`). Do not use `*T` merely to avoid copying.

**`new(expr)` for `*T` fields (Go 1.26)** — when you need a pointer to a copy of an existing value (e.g. to populate a `*int` field), use `new(expr)`:
```go
TTLSeconds: new(config.TTLSeconds)  // *int pointing to a copy — correct
TTLSeconds: &config.TTLSeconds      // *int aliasing the struct field — wrong
```

### Cross-Domain Dependencies
Interfaces are declared in the **consuming** package, implemented by the owning package:
- `policy.EnabledIPsProvider` ← implemented by `*device.Service`
- `lease.TTLConfigRetriever` ← implemented by `*rule.Service`
- `scheduler.ExpiredAddressFinder` ← implemented by `*lease.Service`
- `scheduler.AddressDisabler` ← implemented by `*device.Service`

---

## Testing

### What requires tests

Every package with non-trivial logic must have tests. The minimum bar per package type:

| Package type | Test file required? | Minimum coverage |
|---|---|---|
| Domain service (`service.go`) | **Yes** | All public methods: happy path + each distinct error path |
| Repository (`repository.go`) | **Yes** | Each method: happy path + constraint errors (`ErrNotFound`, unique violations, FK failures) |
| HTTP handler (`handler.go`) | **Yes** | One E2E test per endpoint (happy path) + handler-specific validation paths |
| Background service (`RunSchedule`, `RunListener`, sink) | **Yes** | Core execution logic + context cancellation exits cleanly |
| Domain constructors / value objects | **Yes (same file as service)** | Valid inputs + each invalid input variant |
| Pure infrastructure (logging helpers, middleware, config parsing) | Recommended | Key behaviours that are not obvious from the code |
| Generated code, embed stubs, trivial wrappers | No | — |

When reviewing a package: check for the *absence* of `_test.go` files, not just the quality of existing ones. A package with no tests is a gap even if its code looks clean.

---

### Handler Tests (E2E)

**Package:** `package foo_test` (black-box) — handlers are tested purely through the HTTP interface; no access to unexported internals is needed or wanted.

**Philosophy:** Treat handlers as integration smoke tests. One test per endpoint for the happy path. Add extra tests only for logic that lives in the handler itself (auth enforcement, input validation returning 400, response shaping). Do **not** repeat business-rule combinations here — those belong in service tests.

**Scaffold:**
```go
func TestHandler_CreateDevice(t *testing.T) {
    // Given
    srv := testutils.SetupIntegrationServer(t)
    // use srv.DeviceService / srv.AuthService etc. to seed prerequisite state

    // When
    req := httptest.NewRequest(http.MethodPost, "/api/v1/devices", body)
    req.Header.Set(...)
    w := httptest.NewRecorder()
    srv.ServeHTTP(w, req)

    // Then
    is := is.New(t)
    is.Equal(w.Code, http.StatusCreated)
    var resp httpapi.DeviceResponse
    json.NewDecoder(w.Body).Decode(&resp)
    is.Equal(resp.Name, "my-device")
    // only reach for the repo if the response doesn't expose the state you need:
    // device, _ := srv.DeviceService.GetDevice(ctx, resp.ID)
    // is.True(device.CreatedAt.After(before))
}
```

**Rules:**
- `SetupIntegrationServer(t)` is the only setup entry point — never construct services or repos manually in handler tests.
- Given: call service methods to build state. Never seed the DB directly or call repos from tests.
- When: always a real HTTP call through `ServeHTTP` (exercises the full middleware chain).
- Then: assert on the HTTP response (status code + decoded body). Reach into repo/service only for side effects not visible in the response.
- Auth paths: test unauthenticated and forbidden cases with a dedicated short test (`is.Equal(w.Code, 401)`), not a table.

---

### Service Tests (Unit)

**Package:** `package foo` (white-box) — fake repository implementations must satisfy unexported repository interfaces, which are only visible inside the package.

**Philosophy:** All business logic lives here. No HTTP, no real DB. Use fake repository implementations. One top-level function per scenario — same flat rule as repository tests (not grouped inside a single function with `t.Run`). Use `t.Run` only for true table-driven variations of the same scenario where the same assertion logic applies.

**Fake repository pattern:**
```go
// fakeDeviceRepo implements the unexported repository interface
type fakeDeviceRepo struct {
    devices []device.Device
    err     error // configurable error to simulate failures
}

func (f *fakeDeviceRepo) CreateDevice(_ context.Context, d device.Device) (device.Device, error) {
    if f.err != nil { return device.Device{}, f.err }
    f.devices = append(f.devices, d)
    return d, nil
}
// ... implement remaining interface methods (return zero values if unused in a test)

var _ repository = (*fakeDeviceRepo)(nil) // compile-time interface check
```

**Scaffold:**
```go
func TestService_CreateDevice_ValidInput_CreatesDevice(t *testing.T) {
    is := is.New(t)
    repo := &fakeDeviceRepo{}
    svc := NewService(repo, slog.New(slog.DiscardHandler))

    got, err := svc.CreateDevice(context.Background(), CreateDeviceInput{Name: "dev-1"})

    is.NoErr(err)
    is.Equal(got.Name, "dev-1")
}

func TestService_CreateDevice_EmptyName_ReturnsErr(t *testing.T) {
    is := is.New(t)
    repo := &fakeDeviceRepo{}
    svc := NewService(repo, slog.New(slog.DiscardHandler))

    _, err := svc.CreateDevice(context.Background(), CreateDeviceInput{Name: ""})

    is.True(err != nil)
}

func TestService_CreateDevice_RepoError_Propagated(t *testing.T) {
    is := is.New(t)
    repo := &fakeDeviceRepo{err: errors.New("db")}
    svc := NewService(repo, slog.New(slog.DiscardHandler))

    _, err := svc.CreateDevice(context.Background(), CreateDeviceInput{Name: "dev-1"})

    is.True(err != nil)
}
```

**Rules:**
- One top-level function per scenario (`TestService_MethodName_Condition_ExpectedOutcome`).
- Fake implements the full repository interface; unused methods return zero value + nil error. Add a compile-time check with `var _ repository = (*fakeRepo)(nil)`.
- Add an `err` field (or per-method error fields) to the fake to simulate failure paths without complex setup.
- Test the returned domain value, not internal fake/repo state.

---

### Repository Tests (Integration)

**Package:** `package foo_test` (black-box) — tests only what callers can observe through the public repository API. If something can only be verified by reaching into unexported fields, that is a signal to either add a read method or drop the assertion.

**Philosophy:** Real in-memory SQLite. Test CRUD, constraint violations, filter and pagination behaviour, and `RunInTx` rollback. One top-level function per case — do **not** wrap independent cases inside a single function with `t.Run`. Use private helpers to seed prerequisite rows but keep those helpers unconditional — no branching, just inserts.

**Scaffold:**
```go
func TestDeviceRepository_CreateAndGet(t *testing.T) {
    is := is.New(t)
    db := testdb.New(t)
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

// seed helper — unconditional, returns the created entity for later reference:
func insertDevice(t *testing.T, repo *device.Repository, name string) device.Device {
    t.Helper()
    d, err := repo.CreateDevice(context.Background(), device.Device{Name: name})
    if err != nil { t.Fatalf("insertDevice: %v", err) }
    return d
}
```

**Rules:**
- One top-level function per case (`TestDeviceRepository_CreateAndGet`, `TestDeviceRepository_CreateDuplicate_ReturnsError`, …). `t.Run` is for table-driven variations of the same scenario, not for grouping independent cases under a method name.
- `testdb.New(t)` provides the in-memory SQLite; each test gets a fresh DB (no shared state between tests).
- Seed helpers (`insertDevice`, `insertAddress`, …) take only the minimal fields needed — no optional-parameter tricks, no conditionals.
- When multiple helpers are needed in a single test, call them sequentially in the Given block — do not chain them inside each other.
- `RunInTx` rollback: test by intentionally erroring inside the transaction and confirming the row does not exist afterward.

---

## Critical Files

| File | Purpose |
|------|---------|
| `cmd/api/main.go` | Entry point; signal handling |
| `internal/app/app.go` | Dependency injection and startup |
| `internal/config/config.go` | All env vars; validation in `Load()` |
| `internal/httpserver/server.go` | Global middleware chain |
| `internal/httpserver/routes.go` | Route registration; sub-router middleware |
| `internal/httpserver/authentication.go` | OpenAPI security scheme validation |
| `internal/httpapi/server.gen.go` | Generated DTOs and strict handler interface |
| `internal/device/service.go` | Core business logic; observer notifications |
| `internal/device/repository.go` | DB access; `GetEnabledUniqueIPs`; `RunInTx` |
| `internal/device/events.go` | `AddressEvent`, `EventType`, `AddressObserver` interface |
| `internal/policy/service.go` | In-memory IP cache; `RunListener` |
| `internal/policy/handler.go` | `HandleForwardAuthIP` (Bearer + X-Real-IP) |
| `internal/lease/service.go` | Lease creation/deletion; `RunListener` |
| `internal/scheduler/service.go` | Periodic auto-expiry task |
| `internal/logging/ctx.go` | `FromCtx`, `Enrich` — logger-in-context |
| `internal/testutils/server.go` | `SetupIntegrationServer` for handler tests |
| `api/openapi.yaml` | API schema source of truth |
