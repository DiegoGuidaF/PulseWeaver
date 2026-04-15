# Logging Context

Structured logging flows through context. Handlers set the `operation` attribute; services read the logger from context.

## Pattern

```go
// In a handler:
func (h *HTTPHandler) CreateDevice(ctx context.Context, req ...) (..., error) {
    logger := logging.FromCtx(ctx)
    logger.Info("creating device")
    // ...
}

// In a service (no direct logger setup — reads from context):
func (s *Service) AssignAddress(ctx context.Context, ...) error {
    logger := logging.FromCtx(ctx)
    logger.Info("assigning address", slog.String("device_id", deviceID))
    // ...
}
```

## Enriching context

Add attributes to the logger without creating a new logger:

```go
ctx = logging.Enrich(ctx, slog.String("device_id", id))
// All subsequent logging.FromCtx(ctx) calls include device_id
```

## Available helpers

| Function | Purpose |
|----------|---------|
| `logging.FromCtx(ctx)` | Get logger from context (falls back to `slog.Default()`) |
| `logging.Enrich(ctx, attrs...)` | Return context with enriched logger |
| `logging.WithRequestID(ctx, id)` | Inject request ID into logger |

## Standard attribute keys

| Key | Constant | Set by |
|-----|----------|--------|
| `component` | `AttrKeyComponent` | Service constructors |
| `error` | `AttrKeyError` | Anywhere an error is logged |
| `operation` | `AttrKeyOperation` | Handlers (at entry) |

## Key rules

- **Handlers set `operation`** — this tags all downstream logs for the request.
- **Services only read `logging.FromCtx`** — they never create their own `slog.Logger` per-request.
- **Service constructors may store a logger** for startup/background logging, but per-request work uses context.
- **Never use `fmt.Printf` or `log.Println`** — all logging goes through `slog`.

---
**Verified against:** `internal/logging/ctx.go`, `internal/device/handler.go`, `internal/device/service.go`
**Applies to:** all packages
**Known gaps:** none
**Last verified:** 2026-04-15
