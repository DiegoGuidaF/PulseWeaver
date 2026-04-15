# Service Tests (Unit)

Service tests cover all business logic using fake repository implementations. No HTTP, no real DB.

## Setup

```go
package device // white-box: same package (fakes must access unexported repository interface)

// Fake repository — implements the unexported repository interface
type fakeDeviceRepo struct {
    devices []Device
    err     error // configurable error for failure simulation
}

func (f *fakeDeviceRepo) CreateDevice(_ context.Context, d Device) (Device, error) {
    if f.err != nil { return Device{}, f.err }
    f.devices = append(f.devices, d)
    return d, nil
}
// Implement remaining interface methods (return zero values if unused in test)

var _ repository = (*fakeDeviceRepo)(nil) // compile-time interface check
```

## Test scaffold

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

## Key rules

- **One top-level function per scenario** — `TestService_MethodName_Condition_ExpectedOutcome`.
- **No `t.Run` for grouping** — use `t.Run` only for true table-driven variations of the same assertion logic.
- **Fake implements full interface** — unused methods return zero value + nil. Add `var _ repository = (*fakeRepo)(nil)` compile check.
- **`err` field on fakes** — use per-method error fields for complex fakes.
- **Assert on return values** — not on internal fake state.
- **`slog.DiscardHandler`** — use for the logger in tests.

---
**Verified against:** `internal/device/service_test.go`
**Applies to:** `internal/*/service_test.go`
**Known gaps:** none
**Last verified:** 2026-04-15
