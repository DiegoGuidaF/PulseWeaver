# OpenAPI Layering (Frontend)

The frontend generates its entire HTTP layer from `api/openapi.yaml` via `@hey-api/openapi-ts`. No HTTP knowledge should exist in application code.

## Generated output (`src/lib/api/`)

| Plugin | Output | What it owns |
|--------|--------|-------------|
| `@hey-api/typescript` | `types.gen.ts` | All request/response TypeScript types |
| `@hey-api/sdk` | `sdk.gen.ts` | Typed fetch functions with transformer + validator |
| `@hey-api/schemas` | `schemas.gen.ts` | JSON schemas for all models |
| `zod` | `zod.gen.ts` | Zod schemas for request and response validation |
| `@hey-api/transformers` | (applied to SDK) | Deserializes `date-time` strings into `Date` objects |
| `@tanstack/react-query` | `@tanstack/react-query.gen.ts` | Query/mutation options factories + query key factories |
| `@hey-api/client-fetch` | `client.gen.ts` | Configured fetch client (base URL, cookie auth) |

Everything under `src/lib/api/` is **regenerated on every `npm run generate:api`** — never hand-edit.

## Enforced layering

```
openapi.yaml  →  generate:api  →  src/lib/api/           (generated, do not touch)
                                       ↓
                               src/features/*/hooks/      (thin wrappers: useQuery / useMutation)
                                       ↓
                               src/features/*/components/ (consume hooks only)
                                       ↓
                               src/pages/                 (compose feature components)
```

Each layer imports only from the layer directly below it. Pages never import from `src/lib/api/` directly.

## `src/lib/api-client/` vs `src/lib/api/`

| Directory | Purpose | Editable? |
|-----------|---------|-----------|
| `src/lib/api/` | Everything generated from the spec | **No** |
| `src/lib/api-client/` | `toApiError`, `toErrorMessage`, fetch client config | **Yes** |

## What this prevents

- No manual `fetch` or `axios` calls
- No hand-written TypeScript types for API models
- No hardcoded endpoint strings
- No mismatched cache keys
- Uniform date deserialization via transformer plugin

---
**Verified against:** `frontend/openapi-ts.config.ts`, `src/lib/api/`, `src/lib/api-client/`
**Applies to:** any API interaction
**Known gaps:** none
**Last verified:** 2026-04-15
