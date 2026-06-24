# CLAUDE.md

## Project Overview

PulseWeaver is a self-hosted device IP management service. It maintains an updated list of device IPs and acts as a Forward Auth sidecar for reverse proxies (`GET /api/policy-engine/verify-ip`). Compiles to a **single binary** with the frontend SPA embedded.

## Database Migrations

The app is deployed. **Always create a new numbered migration file** for every schema change — never modify existing migration files. Add the next-numbered `NNNNNN_<name>.up.sql` / `NNNNNN_<name>.down.sql` pair under `internal/database/migrations/`, then write the `up` and `down` SQL. Migrations are applied automatically at app startup.

After adding a migration: review `internal/database/migration_test_seed.sql` and check whether the seed still covers all tables and constraints affected by the migration. Update the seed if the migration touches a table, adds constraints (NOT NULL, CHECK, UNIQUE, FK), or introduces a new table.

## Commands

Targets follow a `back-*` / `front-*` prefix convention (run `make help` for the full list). Backend and frontend commands tab-complete under their prefix; cross-cutting commands (`api`, `build`, `check`) are unprefixed.

### Backend
- `make back-dev` — hot-reload backend (uses Air)
- `make back-test` — run all Go tests (**always use this**, not bare `go test ./...`; uses `-tags=test`)
- `go test -tags=test -v ./internal/<pkg>/...` — run tests for a single package (finish with `make back-test`)
- `make back-lint` — format + golangci-lint + migration-syntax check (ensures migrations start/end a transaction)
- `make back-fix` — format + golangci-lint with auto-fix
- `make back-bench` — run all Go benchmarks (`-tags=test`, no tests)

### Frontend
- `make front-dev` — Vite dev server
- `make front-test` — run frontend tests (vitest)
- `make front-lint` — ESLint + TypeScript type-check (`tsc --noEmit -p tsconfig.app.json`; **never** use bare `tsc --noEmit`)
- `cd frontend && npm run generate:api` — regenerate frontend API types/SDK

### Cross-cutting / Database / Build
- `make api` — regenerate backend + frontend types from `api/openapi.yaml`. **Run this every time `api/openapi.yaml` changes. Never run the underlying generators directly — always use this target.**
- `make check` — full pre-push: back-lint + front-lint + all tests. **Run before declaring any work complete:**
  ```bash
  make check 2>&1 | grep -E "^(FAIL|---FAIL|Error|error:|lint)" | head -40
  ```
- `make back-seed` — seed the local dev DB with the sample world (`make back-seed-db` / `make back-seed-db-sample` build standalone seed-DB artifacts)
- `make build` — production build → `bin/pulseweaver`

## Commits

Conventional Commits, enforced by `scripts/commit-msg` (install once with `make install-hooks`):

```
type(scope)[PW-N]: description
```

- **type** (required): build, chore, ci, docs, feat, fix, perf, refactor, revert, style, test
- **scope** (optional): broad area like `backend`, `ui`; omit for cross-cutting changes
- **[PW-N]** (optional): story ID, after scope and before the colon
- `!` before the colon marks a breaking change

Keep the subject to one concise line. Types map to CHANGELOG sections via `cliff.toml`
(`make release-{patch,minor,major}`), so every type used must have a parser there.

## Architecture & Conventions

**Codebase map:** Before exploring or implementing any backend feature, read `CODEBASE-Backend.md`. It tells you which package owns what, where the critical files are, and what the construction order is — read it before opening files or running searches so you go straight to the right place. For frontend work read `CODEBASE-Frontend.md` first for the same reason.

**Pattern library:** Before implementing any feature, read the relevant index — `../docs/patterns/backend/_index.md` (backend) or `../docs/patterns/frontend/_index.md` (frontend) — then load each individual pattern file whose "Use when" column matches your task. For multi-file backend tasks (add endpoint, wire events, cross-domain queries), also read `../docs/how-to/backend/_index.md`. After implementing, follow the [pattern maintenance protocol](../project/workflow/WORKFLOW.md#pattern-maintenance).

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
- Comments explain what a future reader needs to understand the code — its intent, a non-obvious constraint, a "why". Do not write comments tied to the moment of authoring: no story/ticket IDs, no "newly added", no "changed from X" / "this used to". Prefer timeless rationale over change-narration; the git history records when and why something changed
- Default to no comment. Only add one when the code truly needs it — a hidden constraint, subtle invariant, or surprising behavior that isn't obvious from reading it. Never add a comment that just describes what the code does or how it differs from what it replaced (e.g. "previously this also did X" / "now handles Y instead of Z") — that belongs in the commit message, not the source
