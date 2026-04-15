# Deriving Types from Generated OpenAPI Code

## Pattern

When a hook or component needs a type that corresponds to an OpenAPI parameter (query param, path param, enum), derive it from the generated `types.gen.ts` types rather than hardcoding.

```ts
import type { ListRegistrationsData } from "@/lib/api";

// Extract a query parameter type
type RegistrationQueryStatus = NonNullable<ListRegistrationsData["query"]>["status"];

// Use in hook signature
export function useListRegistrations(status: RegistrationQueryStatus = "pending") { ... }
```

## Why

The generated types are the single source of truth from the OpenAPI spec. Hardcoding `"pending" | "all"` works today but silently drifts if the spec adds a new enum value.

## Codebase examples

- `GetAccessLogData["query"]` — used in access-log hooks for typing the full query object
- `ListRegistrationsData["query"]` — has `status?: 'pending' | 'all'`
- `PendingRegistration["status"]` — has `'pending' | 'used' | 'expired'` (note: different from the query filter)

## What is NOT generated as standalone types

The generator produces inline unions and enums within request/response types. There are **no standalone exported enum constants** like `RegistrationStatus`. To get the type, use indexed access (`Data["query"]["field"]`).

## When manual types are acceptable

Client-only concepts that don't exist in the API (e.g., a `FilterTab` type that adds `"all"` to the server-side status enum) must be defined manually. Keep them in a feature-level `constants.ts`.

---
**Verified against:** `src/lib/api/types.gen.ts`, feature hooks using indexed access types
**Applies to:** any code needing an API-derived type
**Known gaps:** none
**Last verified:** 2026-04-15
