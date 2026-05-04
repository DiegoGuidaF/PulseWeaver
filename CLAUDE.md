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
- `make api` — regenerate backend + frontend types from `api/openapi.yaml`. **Run this every time `api/openapi.yaml` changes. Never run the underlying generators directly — always use this target.**

### Frontend
- `make dev-front` — Vite dev server
- `make test-front` — run frontend tests (vitest)
- `make lint-front` — run ESLint on frontend
- `make typecheck-front` — run TypeScript type-check (`tsc --noEmit -p tsconfig.app.json`; **never** use bare `tsc --noEmit`)
- `cd frontend && npm run generate:api` — regenerate frontend API types/SDK

### Combined / Database / Build
- `make lint-all` — all linters + frontend type-check
- `make check` — full pre-push: lint-all + all tests. **Run before declaring any work complete:**
  ```bash
  make check 2>&1 | grep -E "^(FAIL|---FAIL|Error|error:|lint)" | head -40
  ```
- `make migrate-up` / `make migrate-down` / `make migrate-create`
- `make build` — production build → `bin/pulseweaver`

## Architecture & Conventions

**Codebase map:** Before exploring or implementing any backend feature, read `CODEBASE-Backend.md`. It tells you which package owns what, where the critical files are, and what the construction order is — read it before opening files or running searches so you go straight to the right place. For frontend work read `CODEBASE-Frontend.md` first for the same reason.

**Pattern library:** Before implementing any feature, read `../docs/patterns/backend/_index.md` (backend) and `../docs/patterns/frontend/_index.md` (frontend) and load every pattern that applies. The index has "use when" / "avoid when" columns to help you pick the right ones. After implementing, follow the [pattern maintenance protocol](../project/workflow/WORKFLOW.md#pattern-maintenance).

Other reference docs:
- **UI style:** `../project/workflow/ui-style-guide.md` — two-color system, component color assignments
- **Mantine reference:** https://mantine.dev/llms.txt — fetch before writing any Mantine UI code

For worktree setup (node_modules symlinking, go.work, naming): `../project/workflow/WORKTREES.md`

Critical rules:
- `api/openapi.yaml` is the single source of truth — always run `make api` to regenerate types; never edit generated files in `internal/httpapi/` or `frontend/src/lib/api/`, and never invoke the underlying generators (oapi-codegen, etc.) directly
- Handlers extract primitives from OpenAPI DTOs; never pass generated types deeper
- Mutation hooks own cache invalidation only — `notifications.show()` calls belong in component callbacks
- No Tailwind classes; no `cn()` — use Mantine's `style` prop or CSS modules
- Handlers set `operation` in the logger; services only read via `logging.FromCtx`
