# CLAUDE.md

## Project Overview

PulseWeaver is a self-hosted device IP management service. It maintains an updated list of device IPs and acts as a Forward Auth sidecar for reverse proxies (`GET /api/policy-engine/verify-ip`). Compiles to a **single binary** with the frontend SPA embedded.

## Database Migrations

The app is deployed. **Always create a new numbered migration file** for every schema change — never modify existing migration files. Use `make migrate-create` to generate the next pair, then write the `up` and `down` SQL.

After adding a migration: review `internal/database/migration_test_seed.sql` and check whether the seed still covers all tables and constraints affected by the migration. Update the seed if the migration touches a table, adds constraints (NOT NULL, CHECK, UNIQUE, FK), or introduces a new table.

## Commands

### Backend
- `make dev-back` — hot-reload backend (uses Air)
- `make test` — run all Go tests (**always use this**, not bare `go test ./...`; uses `-tags=test`)
- `go test -tags=test -v ./internal/<pkg>/...` — run tests for a single package (finish with `make test`)
- `make lint-back` — format + golangci-lint + check-migrations(ensures they start/end transaction)
- `make fix` — format + golangci-lint with auto-fix
- `make api` — regenerate backend + frontend types from OpenAPI spec

### Frontend
- `make dev-front` — Vite dev server
- `make test-front` — run frontend tests (vitest)
- `make lint-front` — run ESLint on frontend
- `make typecheck-front` — run TypeScript type-check (`tsc --noEmit -p tsconfig.app.json`; **never** use bare `tsc --noEmit`)
- `cd frontend && npm run generate:api` — regenerate frontend API types/SDK

### Combined / Database / Build
- `make lint-all` — all linters + frontend type-check
- `make check` — full pre-push: lint-all + all tests
- `make migrate-up` / `make migrate-down` / `make migrate-create`
- `make build` — production build → `bin/pulseweaver`

## Architecture & Conventions

**Pattern library:** Before implementing any feature, read `docs/patterns/_index.md` and load every pattern that applies. The index has "use when" / "avoid when" columns to help you pick the right ones. After implementing, follow the [pattern maintenance protocol](../project/workflow/WORKFLOW.md#pattern-maintenance).

Full reference docs:
- **Backend:** `CODEBASE-Backend.md` — package map, domain responsibilities, critical files
- **Frontend:** `CODEBASE-Frontend.md` — directory layout, routing, auth flow, UX surfaces
- **UI style:** `../planning/ui-style-guide.md` — two-color system, component color assignments
- **Mantine reference:** https://mantine.dev/llms.txt — fetch before writing any Mantine UI code

For worktree setup (node_modules symlinking, go.work, naming): `../planning/WORKTREES.md`

Critical rules:
- `api/openapi.yaml` is the single source of truth — run `make api` to regenerate; never edit generated files in `internal/httpapi/` or `frontend/src/lib/api/`
- Handlers extract primitives from OpenAPI DTOs; never pass generated types deeper
- Mutation hooks own cache invalidation only — `notifications.show()` calls belong in component callbacks
- No Tailwind classes; no `cn()` — use Mantine's `style` prop or CSS modules
- Handlers set `operation` in the logger; services only read via `logging.FromCtx`
