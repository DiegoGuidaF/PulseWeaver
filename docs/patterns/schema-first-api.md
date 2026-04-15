# Schema-First API

`api/openapi.yaml` is the single source of truth for all HTTP API contracts. Both backend and frontend types are generated from it.

## Workflow

```
1. Edit api/openapi.yaml
2. Run `make api`  →  oapi-codegen (backend)  +  @hey-api/openapi-ts (frontend)
3. Implement the new handler method on StrictServerInterface
```

## What gets generated

### Backend (`oapi-codegen`)
- `internal/httpapi/server.gen.go` — all DTOs, `StrictServerInterface`, route dispatch
- `internal/httpapi/const.go` — `SessionCookieName`, `APIKeyHeaderName`, auth scope constants

### Frontend (`@hey-api/openapi-ts`)
- `src/lib/api/types.gen.ts` — TypeScript types
- `src/lib/api/sdk.gen.ts` — typed fetch functions
- `src/lib/api/zod.gen.ts` — Zod schemas for validation
- `src/lib/api/@tanstack/react-query.gen.ts` — query/mutation options + key factories

## Key rules

- **Never edit generated files** — they are overwritten on every `make api`.
- **Never add manual `fetch` or `axios` calls** — all HTTP is through the generated SDK.
- **Handlers implement `StrictServerInterface`** — the compiler enforces that every endpoint has an implementation.
- **Breaking changes**: use `!` suffix in commit type (e.g., `feat(api)!: rename field`).

---
**Verified against:** `api/openapi.yaml`, `Makefile`, `internal/httpapi/server.gen.go`
**Applies to:** any API change
**Known gaps:** none
**Last verified:** 2026-04-15
