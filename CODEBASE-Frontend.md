# Frontend Codebase Reference

> Last updated: 2026-03-12

## Directory Structure

```
src/
├── App.tsx                     # Router + providers setup (MantineProvider > BrowserRouter > AuthProvider > AppErrorBoundary)
├── main.tsx                    # QueryClient + global 401 handler, React root; imports Mantine CSS
├── pages/                      # Thin route-level components (routing guard + layout only)
│   ├── DashboardPage.tsx       # /devices — renders CreateDeviceForm + DeviceList
│   ├── DeviceDetailPage.tsx    # /devices/:deviceId — header + Tabs shell
│   ├── LoginPage.tsx           # /login — page shell + redirect if authed
│   ├── SettingsPage.tsx        # /settings — user/account settings
│   └── NotFoundPage.tsx        # * — 404
├── features/
│   ├── auth/
│   │   ├── AuthContext.tsx         # AuthProvider wraps useCurrentUser; exports useAuth() → { user, isLoading, isAuthenticated }
│   │   ├── ProtectedRoute.tsx      # Redirects to /login if not authenticated
│   │   ├── components/
│   │   │   └── LoginForm.tsx       # Login form state + submission mutation
│   │   └── hooks/
│   │       ├── useCurrentUser.ts   # Session query (5min stale, null on 401, refetches on window focus)
│   │       ├── useLogin.ts         # Login mutation → invalidate user → navigate
│   │       └── useLogout.ts        # Logout mutation → removeQueries → /login
│   └── devices/
│       ├── CreateDeviceForm.tsx    # Name form + success dialog with API key
│       ├── DeviceList.tsx          # Table of all devices + delete confirmation
│       ├── DeviceAddressesTab.tsx  # IP address list + add form + disable dialog
│       ├── DeviceSettingsTab.tsx   # Auto-expiry (lease rule) form + status
│       └── hooks/
│           ├── useDevice.ts                        # Single device query (cache-first, 404→undefined)
│           ├── useDevices.ts                       # All devices query
│           ├── useCreateDevice.ts                  # Create mutation (409 handled)
│           ├── useDeleteDevice.ts                  # Delete mutation
│           ├── useDeviceAddresses.ts               # Addresses query (enabled defaults to true)
│           ├── useAddDeviceAddress.ts              # Add address mutation
│           ├── useDisableDeviceAddress.ts          # Disable address mutation
│           ├── useDeviceAddressLeaseRule.ts        # Lease rule query (null on 404, enabled defaults to true)
│           ├── usePutDeviceAddressLeaseRule.ts     # Save/update lease rule mutation
│           └── useDisableDeviceAddressLeaseRule.ts # Disable lease rule mutation
├── components/
│   ├── layout/
│   │   └── AppShell.tsx        # Sidebar + mobile header layout; includes ColorSchemeToggle
│   └── ErrorBoundary.tsx       # AppErrorBoundary — React error boundary with "Try again"
└── lib/
    ├── api/                    # Generated — do not edit (regenerate via make api)
    ├── api-client/
    │   ├── config.ts           # Configures generated client (baseUrl=/api/v1, credentials:include)
    │   ├── errors.ts           # ApiError class, toApiError(), toErrorMessage()
    │   └── index.ts            # Re-exports api + api-client
    └── theme.ts                # Mantine theme — createTheme({ primaryColor, fontFamily, defaultRadius })
```

## Routing

| Path | Component | Guard |
|------|-----------|-------|
| `/login` | LoginPage | — |
| `/` | → redirect `/devices` | — |
| `/devices` | AppShell > DashboardPage | ProtectedRoute |
| `/devices/:deviceId` | AppShell > DeviceDetailPage | ProtectedRoute |
| `/settings` | AppShell > SettingsPage | ProtectedRoute |
| `*` | NotFoundPage | — |

## Key Patterns

### Hook conventions
- **Query hooks**: Return TanStack Query result directly. Exceptions: `useDeviceAddressLeaseRule` normalizes `null` for 404 and returns `{ data, isLoading, isError, error }`; `useCurrentUser` returns `{ data, isLoading, isAuthenticated, error }`.
- **Mutation hooks**: Own server state only (mutation + cache invalidation). No `notifications.show()` calls inside hooks.
- **Notifications**: `notifications.show()` calls belong in component `onSuccess`/`onError` callbacks, not in hooks.
- **Coordination callbacks**: Hooks may still accept an `onSuccess` option for coordination logic (form reset, modal close).
- **Cache invalidation**: Mutations invalidate the minimal relevant query key (device-specific where possible).
- **`enabled` default**: Query hooks that accept `enabled` default it to `true`.

### Error handling
- `toErrorMessage(err)` — extracts a string from any error shape
- `toApiError(err)` — wraps in `ApiError` preserving HTTP status code
- Global 401 handler in `main.tsx` via `QueryCache.onError` redirects to `/login?returnTo=…` (skips `auth/me` which returns 401 when logged out; `ProtectedRoute` handles that case instead)

### Forms
- All forms: `@mantine/form` + `zod4Resolver` from `mantine-form-zod-resolver` (Zod v4 — use `zod4Resolver`, not `zodResolver`)
- Form structure: `<form onSubmit={form.onSubmit(handleSubmit)}>` with Mantine input components taking `{...form.getInputProps('fieldName')}`
- Address form uses generated `zAddAddressRequest` Zod schema directly
- Lease rule form uses a local Zod schema (TTL value + unit)

### Auth flow
- `useCurrentUser` → `AuthContext (AuthProvider)` → `useAuth()` hook consumed by `ProtectedRoute` and `AppShell`
- `LoginForm` owns login form validation/submission and is rendered by `LoginPage`
- Login: POST /auth/login → invalidate `getCurrentUserQueryKey` → navigate (awaits invalidation)
- Logout: POST /auth/logout → `removeQueries()` (clear all) → navigate to /login

## UX Surfaces

### DashboardPage (`/devices`)
- Create new device (form in card; success dialog shows API key — only shown once)
- List all devices (table with name, ID, key prefix, created date; manage link; delete with confirmation)

### DeviceDetailPage (`/devices/:deviceId`)
**Addresses tab** (`DeviceAddressesTab`):
- Add new IP address (form; submit re-enables if IP already exists and was disabled)
- View all assigned addresses (table: IP, status dot, last updated, actions)
- Disable an active address (confirmation dialog)
- Re-enable an inactive address (click Enable in table row)

**Settings & Rules tab** (`DeviceSettingsTab`):
- Auto-expiry rule: set a TTL (seconds/minutes/days) after which addresses auto-expire
- States: disabled (show description + enable form), enabled (show TTL + Change TTL/Turn off buttons), editing (show save form + Cancel)
- Form is rendered once, shared across disabled and editing states; button label adapts (`"Enable auto-expiry"` vs `"Save"`)


## Design Principle: OpenAPI as the Single Source of Truth

The core maintainability strategy for this frontend is that **`api/openapi.yaml` is the only place HTTP API contracts are defined**. The backend generates its types from it via `oapi-codegen`; the frontend generates its entire HTTP layer from it via `@hey-api/openapi-ts`. No HTTP knowledge should exist in application code.

### What the generator produces (`frontend/openapi-ts.config.ts`)

| Plugin | Output | What it owns |
|--------|--------|-------------|
| `@hey-api/typescript` | `src/lib/api/types.gen.ts` | All request/response TypeScript types |
| `@hey-api/sdk` | `src/lib/api/sdk.gen.ts` | Typed fetch functions with transformer + validator |
| `@hey-api/schemas` | `src/lib/api/schemas.gen.ts` | JSON schemas for all models |
| `zod` | `src/lib/api/zod.gen.ts` | Zod schemas for request and response validation |
| `@hey-api/transformers` | (applied to SDK) | Deserializes `date-time` strings into `Date` objects |
| `@tanstack/react-query` | `src/lib/api/@tanstack/react-query.gen.ts` | Query/mutation options factories + query key factories |
| `@hey-api/client-fetch` | `src/lib/api/client.ts` | Configured fetch client (base URL, cookie auth) |

Everything under `src/lib/api/` is **regenerated on every `npm run generate:api`** — never hand-edited.

### The enforced layering rule

```
openapi.yaml  →  generate:api  →  src/lib/api/           (generated, do not touch)
                                       ↓
                               src/features/*/hooks/      (thin wrappers: useQuery / useMutation)
                                       ↓
                               src/features/*/components/ (consume hooks only)
                                       ↓
                               src/pages/                 (compose feature components)
```

Each layer must only import from the layer directly below it. Pages never import from `src/lib/api/` directly. Components never call generated options directly.

### Correct usage patterns

**Query hook** — wraps a generated options factory, adds `enabled` / param logic:
```typescript
// src/features/devices/hooks/useDevices.ts
import { useQuery } from '@tanstack/react-query';
import { getDevicesOptions } from '@/lib/api/@tanstack/react-query.gen';

export function useDevices() {
  return useQuery(getDevicesOptions());
}
```

**Mutation hook** — owns cache invalidation only; no notification calls:
```typescript
// src/features/devices/hooks/useCreateDevice.ts
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { createDeviceMutation, getDevicesQueryKey } from '@/lib/api/@tanstack/react-query.gen';

export function useCreateDevice() {
  const queryClient = useQueryClient();
  return useMutation({
    ...createDeviceMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: getDevicesQueryKey() });
    },
  });
}

// Component owns what to show and when:
// mutation.mutate({ body: values }, {
//   onSuccess: (data) => { setCreatedResult(data); },
//   onError: (err) => notifications.show({ color: 'red', title: 'Error', message: toErrorMessage(err) }),
// });
```

**Form validation** — use generated Zod schemas where they cover the request shape; define a local schema only for fields that deviate from the API contract (e.g. confirm-password):
```typescript
import { createDeviceRequestSchema } from '@/lib/api/zod.gen';

// Reuse directly if the form shape matches the request body exactly
const formSchema = createDeviceRequestSchema;
type FormValues = z.infer<typeof formSchema>;
```

### What this prevents

- No manual `fetch` or `axios` calls anywhere in application code
- No hand-written TypeScript types for API models — they drift; generated types don't
- No hardcoded endpoint strings — operationId changes in the spec break the build immediately
- No mismatched cache keys — query key factories are generated alongside the query options
- Date deserialization handled uniformly by the transformer plugin — no `new Date(response.created_at)` scattered through components

### `src/lib/api-client/` vs `src/lib/api/`

| Directory | Purpose | Editable? |
|-----------|---------|-----------|
| `src/lib/api/` | Everything generated from the spec | **No** — regenerated on every `generate:api` |
| `src/lib/api-client/` | `toApiError`, `toErrorMessage`, fetch client config, query keys for any non-spec endpoints | **Yes** — hand-maintained |

`toApiError` and `toErrorMessage` in `src/lib/api-client/errors.ts` must accept `unknown` and narrow — they are the only place error shape knowledge lives outside the generated layer.

## Testing

Frontend tests use `@testing-library/react`, MSW v2, and vitest (happy-dom environment).

### Two-layer handler model

**Layer 1 — global happy-path defaults** registered in `src/test/setup.ts`. Every test starts in an authenticated, data-loaded state without any per-test `server.use()` call.

**Layer 2 — per-test overrides** via `server.use(...)`. MSW prepends these, shadowing the defaults. `afterEach` → `server.resetHandlers()` (in `setup.ts`) restores layer 1.

### Handler structure (`src/test/mocks/handlers.ts`)

Handlers are domain-grouped named variants. No currying, no pre-invocation:

```typescript
export const authHandlers = {
  me: {
    success: (override?: Partial<User>) => http.get('/api/v1/auth/me', () => HttpResponse.json({ ...createMockUser(), ...override })),
    unauthenticated: () => http.get('/api/v1/auth/me', () => responses.unauthorized()),
  },
};
export const defaultHandlers = [authHandlers.me.success(), deviceHandlers.list(), ...];
```

- Endpoint path strings live only in `handlers.ts` — never hardcoded in test files.
- Mock data shapes are composed via `createMock*` factories in `src/test/mocks/data.ts`.

### `renderWithProviders`

Wraps with `MantineProvider` + `Notifications` + `QueryClientProvider` + `MemoryRouter`. Use for all component/page tests.

### Test call-site pattern

```typescript
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
