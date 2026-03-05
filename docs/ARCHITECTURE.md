# WallyDic Architecture

## Overview

WallyDic is a self-hosted device IP address management service. It maintains an updated list of device IPs and acts as a Forward Auth sidecar for reverse proxies: on every incoming request the proxy asks `GET /api/authz/verify-ip` and WallyDex responds `200` or `403`. The production build compiles to a **single binary** — the React SPA is baked into the Go binary at compile time via `embed.FS`, so no separate static file server is needed.

---

## Data Flow: React Component → Go Handler

### Mutation (Create Device)

```
frontend/src/features/devices/CreateDeviceForm.tsx
  → react-hook-form + zod validation (zCreateDeviceRequest from zod.gen.ts)
  → useCreateDevice() hook
      (frontend/src/features/devices/hooks/useCreateDevice.ts)
  → useMutation({ ...createDeviceMutation() })
  → Generated mutation options
      (frontend/src/lib/api/@tanstack/react-query.gen.ts)
  → SDK function createDevice()
      (frontend/src/lib/api/sdk.gen.ts)
  → HTTP client fetch()
      (frontend/src/lib/api/client/client.gen.ts)
      credentials: "include", baseUrl: /api/v1
  → POST /api/v1/devices

  ──────────────────── network boundary ────────────────────

  → Chi router
      (internal/httpserver/routes.go)
  → Global middleware chain:
      RequestID → RequestLogger → slog-chi → Recoverer
      → ClientIP (from RemoteAddr or X-Forwarded-For)
      → Security headers (CSP, HSTS, X-Frame-Options, …)
      → MaxBodySizeMiddleware (256 KB)
      (internal/httpserver/server.go)
  → /api/v1 sub-router middleware:
      LoginRateLimitMiddleware
      → OapiRequestValidatorWithOptions (schema validation against openapi.yaml)
      → PrincipalUserContextMiddleware (session cookie → ctx)
      → PrincipalDeviceContextMiddleware (API key → ctx)
      (internal/httpserver/routes.go)
  → Generated StrictHandler dispatch
      (internal/httpapi/server.gen.go)
  → HTTPHandler.CreateDevice()
      (internal/device/handler.go)
  → Service.CreateDevice()
      (internal/device/service.go)
  → Repository.CreateDevice()
      (internal/device/repository.go)
  → SQLite via sqlx (MaxOpenConns=1, WAL mode)
      (internal/database/sqlite.go)

  ← domain.Device returned up the stack
  ← Handler maps to httpapi.CreateDevice201JSONResponse
  ← JSON response

  → onSuccess: queryClient.invalidateQueries(getDevicesQueryKey())
  → DeviceList auto-refetches, UI updates
```

### Query (List Devices)

```
frontend/src/features/devices/hooks/useDevices.ts
  → useQuery({ ...getDevicesOptions() })
      (frontend/src/lib/api/@tanstack/react-query.gen.ts)
  → SDK function getDevices()
  → GET /api/v1/devices

  ──────────────────── network boundary ────────────────────

  → same middleware chain as above
  → HTTPHandler.GetDevices()
  → Service.GetDevices()
  → Repository.GetDevices()
  → SQLite SELECT
  ← []domain.Device → JSON array response
  → TanStack Query caches under getDevicesQueryKey()
  → Component re-renders with device list
```

---

## embed.FS: Static Asset Serving

### Build-time embedding

```
make build
  1. npm run build → frontend/dist/   (Vite bundles React SPA)
  2. cp -r frontend/dist internal/ui/dist/
  3. go build -tags=prod → binary
       //go:build prod
       //go:embed dist
       var distFS embed.FS          (internal/ui/ui_prod.go)
```

The `//go:embed dist` directive bakes the entire `dist/` tree into the binary. `fs.Sub(distFS, "dist")` creates a sub-FS rooted at `dist/` so paths like `/assets/main.js` resolve without the `dist/` prefix.

### Request handling (`ui.Handler()`)

The Chi catch-all `r.Handle("/*", ui.Handler())` is registered after all `/api/v1` routes in `internal/httpserver/routes.go:66`. The handler logic:

1. **`/api` prefix** → `404` immediately (prevents API fallthrough even if a route is missing).
2. **File exists in embed.FS** → serve directly.
   - Paths under `/assets/` get `Cache-Control: public, max-age=31536000, immutable` (Vite content-hashes these filenames, so they are safe to cache forever).
   - Directory paths are skipped (no directory listings).
3. **File not found** → serve `index.html` with `Cache-Control: no-cache` (supports SPA client-side routing).

### Dev mode

`internal/ui/ui_dev.go` (build tag `!prod`) returns a plain `404` with a message pointing to the Vite dev server (`npm run dev` in `frontend/`). The backend and frontend run as separate processes on different ports in development.

---

## Top 3 Architectural Improvements

### 1. Add Pagination to List Endpoints

**Problem:** `GET /api/v1/devices` and `GET /api/v1/devices/{id}/addresses` return every record in a single query. `api/openapi.yaml`, the handlers, and the repositories have no `limit`/`offset` or cursor parameters.

**Impact:** Unbounded memory use and query time as the dataset grows. The frontend loads everything into a table at once with no virtual scrolling.

**Recommendation:**
- Add `limit` + `cursor` (or `page`/`offset`) query parameters to `api/openapi.yaml` and regenerate types with `make api`.
- Update repository queries to apply `LIMIT`/`OFFSET` and return a total count or next-cursor.
- Update frontend list components to use TanStack Query's `useInfiniteQuery` with infinite scroll, or paginated controls.

---

### 2. Leverage WAL-mode SQLite Read Concurrency

**Problem:** `internal/database/sqlite.go` sets `MaxOpenConns=1`, serialising **all** database operations — reads and writes — to a single connection, despite `PRAGMA journal_mode=WAL` being enabled. WAL mode supports concurrent readers; one writer never blocks readers.

**Impact:** Every HTTP request that touches the DB blocks all others. Under any meaningful concurrency (multiple browser tabs, API key clients, lease renewals) all requests queue behind one connection.

**Recommendation:** Split into two connection pools:
- **Write pool:** `MaxOpenConns=1`, used exclusively for mutations passed through `RunInTx`.
- **Read pool:** `MaxOpenConns=4` (or higher), opened with a `?mode=ro` DSN suffix, used for all query-only operations.

Repositories would accept the appropriate pool based on operation type, keeping the interface surface minimal.

---

### 3. Optimistic UI Updates for Mutations

**Problem:** Every mutation hook (`useCreateDevice`, `useDeleteDevice`, `useAddDeviceAddress`, etc.) calls `queryClient.invalidateQueries()` in `onSuccess`, triggering a full server round-trip before the UI reflects the change.

**Impact:** Users see a loading state or stale data between the action and confirmation, even on fast local networks, reducing perceived responsiveness.

**Recommendation:** Use TanStack Query's `useMutation` lifecycle for optimistic updates:

```ts
useMutation({
  ...createDeviceMutation(),
  onMutate: async (newDevice) => {
    await queryClient.cancelQueries({ queryKey: getDevicesQueryKey() });
    const previous = queryClient.getQueryData(getDevicesQueryKey());
    queryClient.setQueryData(getDevicesQueryKey(), (old) => [...old, optimisticDevice(newDevice)]);
    return { previous };
  },
  onError: (_err, _vars, ctx) => {
    queryClient.setQueryData(getDevicesQueryKey(), ctx?.previous);
  },
  onSettled: () => {
    queryClient.invalidateQueries({ queryKey: getDevicesQueryKey() });
  },
});
```

`onMutate` applies the change instantly; `onError` rolls it back; `onSettled` syncs with the server regardless of outcome.

---

## Critical Files Reference

| File | Purpose |
|------|---------|
| `api/openapi.yaml` | Schema source of truth for all types and routes |
| `cmd/api/main.go` | Entry point |
| `internal/database/sqlite.go` | Connection config, pragmas, migrations |
| `internal/httpserver/server.go` | Middleware chain assembly |
| `internal/httpserver/routes.go` | Route registration, sub-router middleware |
| `internal/httpserver/middleware.go` | Individual middleware implementations |
| `internal/httpapi/server.gen.go` | Generated routing and DTOs (oapi-codegen) |
| `internal/device/handler.go` | Device HTTP handlers |
| `internal/device/service.go` | Device business logic + observer notifications |
| `internal/device/repository.go` | Device DB access, `RunInTx` |
| `internal/ui/ui_prod.go` | `embed.FS` declaration + SPA handler (prod) |
| `internal/ui/ui_dev.go` | Dev stub returning 404 |
| `internal/logging/ctx.go` | Logger-in-context pattern (`FromCtx`, `Enrich`) |
| `frontend/src/features/devices/CreateDeviceForm.tsx` | Form → mutation wiring |
| `frontend/src/features/devices/hooks/useCreateDevice.ts` | Mutation hook |
| `frontend/src/lib/api/@tanstack/react-query.gen.ts` | Generated query/mutation options |
| `frontend/src/lib/api/sdk.gen.ts` | Generated SDK functions |
| `frontend/src/lib/api/client/client.gen.ts` | HTTP client (fetch wrapper) |
| `frontend/src/lib/api-client/config.ts` | Client setup (baseUrl, credentials) |
