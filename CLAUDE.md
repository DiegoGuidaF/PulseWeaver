# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

PulseWeaver is a self-hosted device IP address management service. It maintains an updated list of device IPs and acts as a Forward Auth sidecar for reverse proxies (`GET /api/policy-engine/verify-ip`). Compiles to a **single binary** with the frontend SPA embedded.

## Database Migrations

The app is deployed. **Always create a new numbered migration file** for every schema change — never modify existing migration files. Use `make migrate-create` to generate the next pair, then write the `up` and `down` SQL.

## Commands

### Backend
- `make dev-back` — hot-reload backend (uses Air)
- `make test` — run all Go tests (**always use this**, not bare `go test ./...`; uses `-tags=test`)
- `go test -tags=test -v ./internal/<pkg>/...` — run tests for a single package (finish with `make test`)
- `make lint` — format + golangci-lint
- `make fix` — format + golangci-lint with auto-fix
- `make api` — regenerate backend + frontend types from OpenAPI spec (`go generate ./...` then frontend `generate:api`)

### Frontend
- `make dev-front` — Vite dev server
- `cd frontend && nvm exec $(cat .nvmrc) npm test` — run frontend tests (vitest)
- `cd frontend && nvm exec $(cat .nvmrc) npm run generate:api` — regenerate frontend API types/SDK from OpenAPI spec

### Database
- `make migrate-up` / `make migrate-down` — apply/rollback migrations
- `make migrate-create` — create new migration pair

### Build
- `make build` — full production build (frontend → embed → Go binary at `bin/pulseweaver`)

## Architecture

### Schema-First API
`api/openapi.yaml` is the **single source of truth**. Backend types generated via `oapi-codegen` → `internal/api/server.gen.go`. Frontend types/SDK generated via `@hey-api/openapi-ts` → `frontend/src/lib/api/` (do not modify this directory).

### Backend (Go — Chi, SQLite, sqlx)
Layered architecture: **Handler → Service → Repository → Database**

- **Entry point:** `cmd/api/main.go`
- **Domain packages** in `internal/`: `auth`, `policy`, `device`, `audit`, `queries`, `lease`, `rule`, `health`
- **Infrastructure:** `config`, `database`, `httpserver`, `httpapi`, `logging`, `scheduler`, `testdb`, `testutils`, `ui`, `app`
- **Domain constructors** (e.g. `NewUser`, `NewDevice`) enforce all business validation
- **Handlers** extract primitives from OpenAPI DTOs, pass to services; never pass OpenAPI types deeper
- **Repositories** implement `RunInTx` for transactional operations; map DB errors to domain errors
- **Inter-domain communication:** channels for async events, interfaces for cross-domain data access (consumer declares interface, owner implements)

### Frontend (React 19, TypeScript, Vite)
- **Entry:** `frontend/src/main.tsx`
- **Feature slices** in `frontend/src/features/` (devices, settings)
- **Server state:** TanStack Query v5 with generated query/mutation options from `@/lib/api/@tanstack/react-query.gen`
- **UI:** Mantine v8 — components, layout (`AppShell`), theming, notifications; see ADR-004
- **Icons:** `@tabler/icons-react` (all icons named `IconXxx`)
- **Forms:** `@mantine/form` + `zod4Resolver` from `mantine-form-zod-resolver` (Zod v4)
- **API helpers:** `@/lib/api-client/` for `toApiError`, `toErrorMessage`, client config

#### Documentation
- `CODEBASE-Backend.md` — Read this file before any backend task to understand the backend package structure, domain boundaries, service lifecycle, observer pattern, and key conventions.
- `CODEBASE-Frontend.md` — Read this file to understand the frontend directory structure, routing, hook conventions, and UX surfaces before making structural changes to the frontend.
- **Mantine component/hook reference (LLM-optimised):** https://mantine.dev/llms.txt — fetch this when working on any Mantine UI code; it is a full index of all components and hooks, updated with every release.

## Key Conventions

### Go Style
- "Boring Go" — explicit error handling with early returns, small focused functions
- Structured logging via `slog` with logger-in-context pattern (`logging.FromCtx`, `logging.Enrich`)
- Log attribute keys: use domain-defined constants (e.g. `AttrKeyDeviceID`), always `snake_case`
- Handlers own log enrichment (set `operation` to handler method name); services just read via `FromCtx`
- Service lifecycle: `Run(ctx context.Context) error` pattern, context cancellation for shutdown
- Config via env vars loaded in `internal/config`, passed into constructors

### Testing
- **Domain:** table-driven unit tests, no dependencies
- **Service:** table-driven unit tests with mock repositories
- **Repository:** integration tests with real in-memory SQLite (`file::memory:`)
- **Handler (E2E):** `httptest` with real services + in-memory DB; setup test data via service calls (not HTTP), HTTP calls only for assertions
- **Frontend:** `@testing-library/react`, MSW for API mocking, `renderWithProviders` for test setup (wraps with `MantineProvider` + `Notifications` + `QueryClientProvider` + `MemoryRouter`)
  - `src/test/mocks/handlers.ts` exports domain-grouped handler factories and a `defaultHandlers` array (the happy path, registered globally in `setup.ts`)
  - Tests only call `server.use(...)` for deviations from the happy path; endpoint strings never appear in test files
- Use `matryer/is` or `testify/assert` for Go assertions

### Frontend Patterns
- `function ComponentName() {}` (not arrow), named exports
- No `any` — use `unknown` and narrow
- No Tailwind classes; no `cn()` utility — use Mantine's `style` prop or CSS modules for one-off layout
- Generated query keys for cache invalidation
- Mutation hooks own server state only (mutation + cache invalidation) — no notification calls inside hooks
- Notification calls (`notifications.show()`) live in component `onSuccess`/`onError` callbacks, not in hooks
- Hooks may still accept an `onSuccess` callback for coordination logic (form reset, modal close)
- Fetch https://mantine.dev/llms.txt before writing any Mantine UI code — full component/hook reference
