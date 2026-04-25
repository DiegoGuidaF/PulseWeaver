# Frontend Pattern Index

> Before implementing a feature, scan this index and read every pattern that applies.
> After implementing, check the [self-improvement protocol](../../../project/workflow/WORKFLOW.md#pattern-maintenance).

| Pattern | File | Use when | Avoid when | Refs |
|---------|------|----------|------------|------|
| Hook conventions | `hook-conventions.md` | Writing any query or mutation hook | — | `features/devices/hooks/`, `features/auth/hooks/` |
| OpenAPI layering | `openapi-layering.md` | Adding API interactions, understanding the generated layer | Modifying generated files (never do) | `openapi-ts.config.ts`, `src/lib/api/` |
| Form validation | `form-validation.md` | Building any form with validation | Form shape differs from API request (define local schema) | `UsersTab.tsx`, `zod.gen.ts` |
| Error handling | `error-handling.md` | Handling API errors, displaying error messages | — | `src/lib/api-client/errors.ts`, `src/main.tsx` |
| Date handling | `date-handling.md` | Displaying or inputting dates | — | `src/lib/dates.ts` |
| Create flow UX | `create-flow-ux.md` | Adding a create/add form to a page | — | `UsersTab.tsx` |
| Feature constants | `feature-constants.md` | Adding label maps, badge colors, UI option arrays | Deriving types from OpenAPI (use `type-derivation-from-openapi`) | `access-log/constants.ts`, `provisioning/constants.ts` |
| Type derivation from OpenAPI | `type-derivation-from-openapi.md` | Typing a query param, enum, or any API-derived value | Client-only concepts not in the API spec | `types.gen.ts` |
| Shared hooks | `shared-hooks.md` | Using clipboard, or adding a cross-feature hook | Hook is used by only one feature (keep it feature-local) | `src/hooks/useClipboard.ts` |
| Frontend testing | `frontend-testing.md` | Writing component or integration tests | — | `src/test/setup.ts`, `src/test/mocks/handlers.ts` |
| Entity display config | `entity-display-config.md` | Entities need consistent color+icon across multiple components (groups, tags) | Single-use display, one component only | `features/host-access/hostIconConfig.ts`, `features/host-access/utils/groupColor.ts` |
| Badge overflow list | `badge-overflow-list.md` | Table cell shows unbounded list of entity badges | List is always small/bounded | `features/host-access/components/GroupBadgeList.tsx` |
