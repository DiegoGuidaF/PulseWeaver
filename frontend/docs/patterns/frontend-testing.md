# Frontend Testing

Tests use `@testing-library/react`, MSW v2, and vitest (happy-dom environment).

## Two-layer handler model

**Layer 1 — global happy-path defaults** registered in `src/test/setup.ts`. Every test starts in an authenticated, data-loaded state without any per-test `server.use()` call.

**Layer 2 — per-test overrides** via `server.use(...)`. MSW prepends these, shadowing the defaults. `afterEach` → `server.resetHandlers()` (in `setup.ts`) restores layer 1.

## Handler structure (`src/test/mocks/handlers.ts`)

Handlers are domain-grouped named variants. No currying, no pre-invocation:

```ts
export const authHandlers = {
  me: {
    success: (override?: Partial<User>) =>
      http.get('/api/v1/auth/me', () =>
        HttpResponse.json({ ...createMockUser(), ...override })),
    unauthenticated: () =>
      http.get('/api/v1/auth/me', () => responses.unauthorized()),
  },
};
export const defaultHandlers = [authHandlers.me.success(), deviceHandlers.list(), ...];
```

- Endpoint path strings live only in `handlers.ts` — never hardcoded in test files.
- Mock data shapes are composed via `createMock*` factories in `src/test/mocks/data.ts`.

## Test utilities

| Utility | Location | Purpose |
|---------|----------|---------|
| `renderWithProviders` | `src/test/utils.tsx` | Wraps with `MantineProvider` + `Notifications` + `QueryClientProvider` + `MemoryRouter` |
| `TEST_TIMEOUTS` | `src/test/constants.ts` | Shared timeout values for async assertions |

## Test call-site pattern

```ts
// happy path — no server.use() needed
it('renders device list', async () => {
  renderWithProviders(<DeviceList />);
  expect(await screen.findByText('Test Device')).toBeInTheDocument();
});

// error / specific-data case — only declare the deviation
it('shows empty state', async () => {
  server.use(deviceHandlers.list([]));
  renderWithProviders(<DeviceList />);
  expect(await screen.findByText(/no devices/i)).toBeInTheDocument();
});
```

## Key rules

- **No per-test boilerplate** — layer 1 handles the common case.
- **Override only the deviation** — `server.use()` for the specific endpoint being tested.
- **`renderWithProviders`** — always use this, never raw `render()`.
- **Endpoint strings only in `handlers.ts`** — tests reference handler variants by name.

---
**Verified against:** `src/test/setup.ts`, `src/test/mocks/handlers.ts`, `src/test/utils.tsx`
**Applies to:** all frontend tests
**Known gaps:** none
**Last verified:** 2026-04-15
