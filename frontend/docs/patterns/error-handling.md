# Error Handling

All error narrowing flows through two utilities in `src/lib/api-client/errors.ts`. No other file should contain error shape knowledge.

## Utilities

| Function | Purpose |
|----------|---------|
| `toErrorMessage(err)` | Extracts a display string from any error shape |
| `toApiError(err)` | Wraps in `ApiError` preserving HTTP status code |

Both accept `unknown` and narrow internally.

## Global 401 handler

In `main.tsx`, `QueryCache.onError` redirects to `/login?returnTo=…` on 401 responses. It skips `auth/me` (which returns 401 when logged out — `ProtectedRoute` handles that case instead).

## Component-level error handling

```ts
mutation.mutate({ body: values }, {
  onError: (err) => notifications.show({
    color: 'red',
    title: 'Error',
    message: toErrorMessage(err),
  }),
});
```

## Key rules

- **Never catch and silence errors** in hooks — let TanStack Query manage error state.
- **Error display belongs in components**, not hooks.
- **`ApiError` preserves status code** — use it when you need to branch on HTTP status (e.g., 409 conflict).

---
**Verified against:** `src/lib/api-client/errors.ts`, `src/main.tsx`
**Applies to:** all error handling
**Known gaps:** none
**Last verified:** 2026-04-15
