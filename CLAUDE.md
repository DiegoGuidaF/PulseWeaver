# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

WallyDic (WallyDex) is a self-hosted device IP address management service. It maintains an updated list of device IPs and acts as a Forward Auth sidecar for reverse proxies (`GET /api/authz/verify-ip`). Compiles to a **single binary** with the frontend SPA embedded.

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
- `cd frontend && npm test` — run frontend tests (vitest)
- `cd frontend && npm run generate:api` — regenerate frontend API types/SDK from OpenAPI spec

### Database
- `make migrate-up` / `make migrate-down` — apply/rollback migrations
- `make migrate-create` — create new migration pair

### Build
- `make build` — full production build (frontend → embed → Go binary at `bin/wallydic`)

## Architecture

### Schema-First API
`api/openapi.yaml` is the **single source of truth**. Backend types generated via `oapi-codegen` → `internal/api/server.gen.go`. Frontend types/SDK generated via `@hey-api/openapi-ts` → `frontend/src/lib/api/` (do not modify this directory).

### Backend (Go — Chi, SQLite, sqlx)
Layered architecture: **Handler → Service → Repository → Database**

- **Entry point:** `cmd/api/main.go`
- **Domain packages** in `internal/`: `auth`, `authz`, `device`, `lease`, `rule`, `health`
- **Infrastructure:** `config`, `database`, `httpserver`, `httpapi`, `logging`, `scheduler`, `testdb`, `testutils`, `ui`, `app`
- **Domain constructors** (e.g. `NewUser`, `NewDevice`) enforce all business validation
- **Handlers** extract primitives from OpenAPI DTOs, pass to services; never pass OpenAPI types deeper
- **Repositories** implement `RunInTx` for transactional operations; map DB errors to domain errors
- **Inter-domain communication:** channels for async events, interfaces for cross-domain data access (consumer declares interface, owner implements)

### Frontend (React 19, TypeScript, Vite)
- **Entry:** `frontend/src/main.tsx`
- **Feature slices** in `frontend/src/features/` (devices, settings)
- **Server state:** TanStack Query v5 with generated query/mutation options from `@/lib/api/@tanstack/react-query.gen`
- **UI:** shadcn/ui + Tailwind CSS v4; forms via react-hook-form + zod
- **API helpers:** `@/lib/api-client/` for `toApiError`, `toErrorMessage`, client config

#### Documentation
- `CODEBASE-Frontend.md` — Read this file to understand the frontend directory structure, routing, hook conventions, and UX surfaces before making structural changes to the frontend.
- `CODEBASE-Backend.md` — Read this file to understand the backend package structure, domain boundaries, service lifecycle, observer pattern, and key conventions before making structural changes to the backend.

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
- **Frontend:** `@testing-library/react`, MSW for API mocking (`createHttpHandler` pattern in `src/test/mocks/handlers.ts`), `renderWithProviders` for test setup
- Use `matryer/is` or `testify/assert` for Go assertions

### Frontend Patterns
- `function ComponentName() {}` (not arrow), named exports
- No `any` — use `unknown` and narrow
- Use `cn()` for conditional Tailwind classes
- Generated query keys for cache invalidation; Sonner toasts for all mutation errors/successes
