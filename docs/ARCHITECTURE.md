# PulseWeaver Architecture

A high-level map of how PulseWeaver is built — enough to orient a new contributor (human or AI)
before diving into code. It describes the **shape** of the system and the boundaries between its
parts; it deliberately does not duplicate details that live in the code.

Once you know which side you're working on, the detailed "what exists and where" maps are:

- [`CODEBASE-Backend.md`](../CODEBASE-Backend.md) — backend packages, domain boundaries, critical files.
- [`CODEBASE-Frontend.md`](../CODEBASE-Frontend.md) — frontend routes, features, UX surfaces.

---

## What it is

PulseWeaver is a self-hosted device IP address management service. It maintains an up-to-date list
of device IPs and acts as a **Forward Auth sidecar** for reverse proxies: on every incoming request
the proxy asks `GET /api/policy-engine/verify-ip` and PulseWeaver answers `200` (allow) or `403`
(deny). The hot path answers from an in-memory cache, with no database round-trip.

The production build compiles to a **single binary**: the React SPA is baked into the Go binary at
compile time via `embed.FS`, so there is no separate static file server to deploy.

---

## The big picture

Three parts, bound by one contract:

```
┌─────────────────┐     OpenAPI contract      ┌──────────────────┐
│  React SPA       │  api/openapi.yaml          │  Go backend       │
│  (frontend/)     │  ── make api ──▶           │  (internal/)      │
│                  │   generated SDK + types    │                   │
│  pages           │                            │  handler          │
│   → features     │  ◀── HTTP /api/v1 ──▶      │   → service        │
│    → hooks       │                            │    → repository    │
│     → generated  │                            │     → SQLite       │
│        SDK       │                            │                   │
└─────────────────┘                            └──────────────────┘
        embedded into the binary at build time ──────────┘
```

The **API contract is the seam** between frontend and backend, and it is the part worth
understanding first because everything else hangs off it (see *The API contract* below).

---

## Backend: layered + domain-oriented

The backend follows a strict layering, and dependencies only ever flow downward:

```
handler  → transport only: extract primitives from generated DTOs, map results back to DTOs
service  → business logic: orchestrates domain + repositories, owns invariants
repository → persistence: owns all SQL and maps DB errors to domain errors
SQLite
```

Two rules keep the layers honest:

- **Generated OpenAPI types stay at the transport edge.** Handlers unwrap them into primitives/domain
  types; nothing below the handler ever imports the generated package.
- **Invariants live in domain constructors** (e.g. `auth.NewUser`), not scattered through services.

Code is organised by **bounded context**, not by layer — each domain package
(`device`, `auth`, `policy`, `hosts`, `lease`, …) owns its handler, service, repository, and
domain types together. Cross-domain reads go through the `queries` package (a lite CQRS read side)
rather than reaching across domain repositories. Domains communicate changes through an in-process
**observer** mechanism (e.g. a device address change notifies the policy cache).

For the package-by-package breakdown, wiring/construction order, and the observer registrations, see
[`CODEBASE-Backend.md`](../CODEBASE-Backend.md).

---

## Frontend: pages → features → hooks → generated SDK

The SPA mirrors the same downward flow:

```
page component   → route-level composition (lib/routes.ts, App.tsx)
 → feature        → src/features/<area>/ owns its components + hooks
  → query/mutation hook  → wraps generated TanStack Query options
   → generated SDK        → typed fetch client (src/lib/api/, generated)
```

Hooks own cache invalidation; user-facing notifications live in component callbacks, not in hooks.
For the route table, feature map, and UX surfaces, see [`CODEBASE-Frontend.md`](../CODEBASE-Frontend.md).

---

## The API contract

`api/openapi.yaml` is the **single source of truth** for every endpoint and type. Running `make api`
regenerates both sides from it:

- **Backend** — `oapi-codegen` produces DTOs and a strict handler interface in `internal/httpapi/`.
- **Frontend** — a typed SDK, TanStack Query options, and zod schemas under `frontend/src/lib/api/`.

Generated files are never edited by hand, and the generators are never invoked directly — always go
through `make api`. Changing an endpoint means editing the schema, regenerating, then implementing
against the new types on both sides.

### A request, end to end

Tracing `POST /api/v1/devices` (create device) shows how the pieces connect:

```
CreateDeviceForm  → react-hook-form + zod (generated schema)
  → useCreateDevice() → useMutation(createDeviceMutation())
  → generated SDK createDevice() → fetch (credentials: include, baseUrl /api/v1)

  ──────────────── network boundary ────────────────

  → Chi router + global middleware
      (RequestID → logging → Recoverer → ClientIP → security headers → body-size limit)
  → /api/v1 middleware
      (rate limit → OpenAPI request validation → principal-from-cookie → principal-from-API-key)
  → generated StrictHandler dispatch
  → device.HTTPHandler.CreateDevice()   (extract primitives)
  → device.Service.CreateDevice()       (domain constructor + invariants)
  → device.Repository.CreateDevice()    (SQL, error mapping)
  → SQLite (sqlx, WAL, MaxOpenConns=1)

  ← domain.Device flows back up; handler maps it to the 201 DTO; JSON response

  → onSuccess: queryClient.invalidateQueries(devices) → list refetches → UI updates
```

A read (`GET /api/v1/devices`) is the same path without the domain-constructor step, ending in
TanStack Query caching the result under its query key.

---

## Single-binary build & static serving

`make build` embeds the frontend into the Go binary:

```
make build
  1. npm run build → frontend/dist/        (Vite bundles the SPA)
  2. cp -r frontend/dist internal/ui/dist/
  3. go build -tags=prod → binary          (//go:embed dist in internal/ui/ui_prod.go)
```

The Chi catch-all `r.Handle("/*", ui.Handler())` is registered after all `/api/v1` routes and serves
the embedded SPA:

- Requests under `/api` that miss a route return `404` (no SPA fallthrough for the API).
- Existing files are served directly; `/assets/*` (content-hashed by Vite) get a long immutable
  `Cache-Control`.
- Anything else falls back to `index.html` with `no-cache`, so client-side routing works.

In development the prod embed is replaced by a stub (`!prod` build tag) that returns `404` pointing
at the Vite dev server — backend and frontend run as separate processes on different ports.

---

## Where to go next

- Backend work → [`CODEBASE-Backend.md`](../CODEBASE-Backend.md)
- Frontend work → [`CODEBASE-Frontend.md`](../CODEBASE-Frontend.md)
- Contributing & conventions → [`CONTRIBUTING.md`](../CONTRIBUTING.md)
- API schema → [`api/openapi.yaml`](../api/openapi.yaml)
