# Frontend Codebase Reference

> Last updated: 2026-07-11

This document is the **map** of the frontend codebase — what exists and where. For the system-level
overview (pages → features → hooks → generated SDK, the API contract), see
[`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md). For implementation conventions, hook patterns,
testing scaffolds, and OpenAPI layering, see the workspace pattern library
(`docs/patterns/frontend/`).

The route table comes first, then the auth flow, the per-page UX surfaces, the `src/features/` map,
shared components, and cross-cutting contexts/hooks. The generated API layer is summarised at the
end — its contract lives in `ARCHITECTURE.md`.

---

## Routing & layout

Routes are defined in `lib/routes.ts` (`ROUTES` map + `buildRoute` path helpers) and wired in
`App.tsx`. A `RootLayout` route renders `AuthProvider` + `AppErrorBoundary` around an `<Outlet/>`
(a layout route, so pages can call `useBlocker`/`useUnsavedChangesGuard`). Every route except
`/login` is wrapped in `ProtectedRoute` + `AppShell` via the `protectedPage()` helper.

| Route | Page component | Notes |
|-------|----------------|-------|
| `/login` | `LoginPage` | unauthenticated |
| `/` | → `Navigate` to `/dashboard` | |
| `/dashboard` | `TrafficDashboardPage` | |
| `/devices` | `DevicesPage` | all devices grouped by owner |
| `/devices/owners/:ownerId` | `UserDevicesPage` | owner-scoped device workspace (the device-detail surface) |
| `/devices/owners/:ownerId/new` | `UserDevicesPage createMode` | same page, in-pane create form (`DeviceCreatePane`) |
| `/access/hosts` | `HostsPage` | |
| `/access/host-groups` | `HostGroupsPage` | |
| `/access/users` | `UsersPage` | |
| `/access/users/:id` | `UserDetailPage` | |
| `/access/network-policies` | `NetworkPoliciesPage` | |
| `/access/network-policies/:id` | `NetworkPolicyDetailPage` | |
| `/access-log` | `AccessLogPage` | |
| `/address-history` | `AddressHistoryPage` | |
| `/policy-audit` | `PolicyAuditPage` | nav label is "Access Verification" |
| `/anomalies` | `AnomaliesPage` | |
| `/settings` | `SettingsPage` | |
| `*` | `NotFoundPage` | |

**Nav (`components/layout/AppShell.tsx`, `navGroups`):** Dashboard · **Devices** (Devices) ·
**Access** (Hosts, Host Groups, Users, Network Policies) · **Auditing** (Access Logs, IP Address
Logs, Access Verification, Anomalies) · Settings. The shell is a collapsible/resizable sidebar
(width + collapsed state persisted in `localStorage`), with a color-scheme toggle and help link in
the header and user info + logout in the footer. Nav items can carry a live-data badge (currently
only Anomalies): a `Partial<Record<href, number>>` computed in the component body from a query
hook, looked up per item at render time so the static `navGroups` declaration stays data-free;
renders as a `Badge` in the expanded item and an `Indicator` dot (plus the count appended to the
tooltip label) when the sidebar is collapsed.

## Auth flow

Drifts easily — verify against `features/auth/` before trusting.

| Concern | Where | Behaviour |
|---------|-------|-----------|
| Current user | `hooks/useCurrentUser` → `AuthProvider` (`AuthContext.tsx`) → `useAuth()` | `useAuth` exposes `{ user, isLoading, isAuthenticated }`, consumed by `ProtectedRoute` and `AppShell`. Context object lives in `auth-context.ts`. |
| Route guard | `ProtectedRoute.tsx` | Loading → spinner; unauthenticated → `Navigate` to `/login?returnTo=<path>`; authenticated but `must_change_password` → forced to `/settings`. |
| Login | `components/LoginForm` (in `LoginPage`) + `hooks/useLogin` | POST `/auth/login` → `invalidateQueries(getCurrentUserQueryKey)` (awaited) → navigate to `?returnTo=` param or `/dashboard`. |
| Logout | `hooks/useLogout` (triggered from `AppShell`) | POST `/auth/logout` → `setQueryData(getCurrentUserQueryKey, null)` (flips auth → `ProtectedRoute` redirects) → `removeQueries` for every other key. Does **not** `queryClient.clear()` first — that detaches the `useCurrentUser` observer and the redirect never fires. |
| Global 401 | `main.tsx` `QueryCache.onError` + `defaultOptions.queries.retry` | Non-auth query 401 → `setQueryData(getCurrentUser, null)` so `ProtectedRoute` redirects to `/login?returnTo=<path>` via the router (no full-page reload); 401s are not retried. The `getCurrentUser` query is exempt (401 means "not logged in"). See `error-handling.md`. |

## UX Surfaces

| Route | Page | Surface |
|-------|------|---------|
| `/dashboard` | `TrafficDashboardPage` | Security posture + traffic analytics. `features/dashboard/DashboardView` composes `PostureStrip` ("now" posture), `features/anomalies`' `AnomalySection` ("Unusual activity", also "now" state — sits outside the time-range-scoped Traffic stack), a time-range preset, `DashboardStatCards`, `AttributionSection`, `TrafficLineChart` + `ServiceBarChart`, `CountryStatsSection` (incl. `AccessMap`), `TopDeniedIPsTable`. |
| `/devices` | `DevicesPage` | Admin list of all devices grouped by owner — renders `OwnerGroupList`. |
| `/devices/owners/:ownerId` | `UserDevicesPage` | Owner-scoped device workspace: a resizable device sidebar (`OwnerDevicesPanel`) + a per-device tabbed panel (tab tracked in `?tab=`, device in `?device=`). Banners surface pending-pairing, disabled, and "API key but no limit rule" states. Device-create lives at the `/new` sub-route (in-pane `DeviceCreatePane`; `DeviceCreateEmptyState` when the owner has none). |
| `/access/hosts` | `HostsPage` | Tabbed (`HostsTab`, `SuggestionsTab`) with a staged-changes bar (`StagedChangesBar`) for bulk reconcile; drafts in `features/host-access/drafts/`. |
| `/access/host-groups` | `HostGroupsPage` | Host-group master/detail + membership editing via `HostGroupsTab` (wraps `GroupMasterList`, `GroupDetailPanel`, `GroupMembershipTables`, `GroupMetadataModal`); staged-changes bar. |
| `/access/users` | `UsersPage` | User administration list (`DataTable` + `GroupBadgeList`); create via `CreateUserModal`. |
| `/access/users/:id` | `UserDetailPage` | Per-user detail with **Access** tab (subject groups + effective hosts via shared `features/subjects/` panels, staged-changes save with bypass acknowledgement) and **Devices** tab (`UserDevicesTab`); promote/demote/delete in the header. |
| `/access/network-policies` | `NetworkPoliciesPage` | CIDR network-policy list (`NetworkPoliciesTable`, filterable by group); `CreateNetworkPolicyModal`. |
| `/access/network-policies/:id` | `NetworkPolicyDetailPage` | Per-policy detail + access editing; `EditNetworkPolicyModal`, `DeleteNetworkPolicyModal`. |
| `/access-log` | `AccessLogPage` | `PageToolbar` (time range + auto-refresh) over `AccessLogTable`; a row opens `AccessLogDetailDrawer` (request/device/location/headers). Answers "which IPs requested verification and the outcome". |
| `/address-history` | `AddressHistoryPage` | `AddressHistoryView` — address-over-time plot + filterable list of address lease events. |
| `/policy-audit` | `PolicyAuditPage` (nav "Access Verification") | Policy decision-cache snapshot + simulation: cache stats header, a "Test request" `SimulateBar`, and tabs **Device entries** (`PolicyUserTable` + `PolicyUserDrawer`) / **Network policies** (`NetworkPolicyCacheTab`). |
| `/anomalies` | `AnomaliesPage` | Detected-anomaly list: status/severity/kind filters (`AnomaliesFilterBar`), narrative rows (`AnomalyRow`) that expand to raw `evidence` (`EvidenceList`), inline acknowledge. Shares its default-filter query with the dashboard section and the nav badge — see `features/anomalies`. |
| `/settings` | `SettingsPage` | Tabbed: **Account** (`AccountTab` — profile + change password, with unsaved-changes guard) and **Preferences** (`PreferencesTab` — date-time locale). User administration lives under Access › Users, not here. |

### Device workspace tabs (`UserDevicesPage`, `?tab=`)

| Tab | Component | What |
|-----|-----------|------|
| Addresses | `DeviceAddressesTab` | add / disable / re-enable IPs; table of assigned addresses |
| Rules | `DeviceRulesTab` | `AddressLeaseRuleCard` (auto-expiry TTL) + `MaxActiveIpsRuleCard` (cap on active IPs); each toggle-gated |
| Pairing | `device-pairing/DevicePairingTab` | generate / revoke a one-time pairing code for the heartbeat client; tab shows an indicator in `PENDING_CLAIM`/`EXPIRED_CLAIM` |
| History | `DeviceHistoryTab` | device-scoped address event log |
| Settings | `DeviceSettingsTab` | profile (`DeviceProfileCard`), API key prefix + regenerate/remove, transfer ownership, delete (danger zone) |

## Features map (`src/features/`)

Each feature owns its own `components/` + `hooks/` (and where relevant `constants.ts`, `drafts/`,
`utils/`, config files). "Key files" orients you inside the folder.

| Feature | Owns | Key files |
|---------|------|-----------|
| `auth` | Authentication, the auth context/route guard, and user administration (CRUD + role + password). | `AuthContext.tsx`, `auth-context.ts`, `ProtectedRoute.tsx`, `components/LoginForm.tsx`, `components/{CreateUser,DeleteUser,RoleChange}Modal.tsx`, `hooks/use{Login,Logout,CurrentUser,CreateUser,DeleteUser,Promote,Demote,UpdateMe,ChangePassword}.ts` |
| `dashboard` | Security-posture + traffic analytics surface. | `components/DashboardView.tsx`, `PostureStrip`, `DashboardStatCards`, `AttributionSection`/`AttributionTable`, `ServiceBarChart`/`ServiceDonutChart`, `CountryStatsSection`, `AccessMap`, `TopCountriesTable`, `TopDeniedIPsTable`; `hooks/useDashboard*.ts` |
| `devices` | Device list + the per-device workspace (addresses, rules, history, settings, create). | `OwnerGroupList`, `OwnerDevicesPanel`, `OwnerCard`, `DeviceRow`, `Device{Addresses,Rules,History,Settings}Tab`, `DeviceCreatePane`, `DeviceProfileCard`, `{AddressLease,MaxActiveIps}RuleCard`, `deviceTypeConfig.ts`; `hooks/useDevice*.ts`, `useOwnerGroup.ts` |
| `device-pairing` | Pairing-code lifecycle so the heartbeat client can claim a device API key. | `DevicePairingTab.tsx`, `PairingCreationForm`, `PairingCodeDisplay`, `PairingStatusHero`, `PairingConfigSummary`, `DevicePairingBanner`; `hooks/use{Create,Delete,List}DevicePairing.ts` |
| `host-access` | Known hosts, host groups, and suggestions; staged-changes bulk reconcile. | `HostsTab`, `SuggestionsTab`, `HostGroupsTab`, `GroupMasterList`, `GroupDetailPanel`, `GroupMembershipTables`, `GroupMetadataModal`, `StagedChangesBar`, `AddHostModal`; `drafts/`, `hooks/use{Hosts,HostGroups,HostSuggestions}.ts`, reconcile/ignore hooks |
| `network-policies` | CIDR network-policy CRUD. | `NetworkPoliciesTable`, `{Create,Edit,Delete}NetworkPolicyModal`, `NetworkPolicyHeader`; `hooks/use{Create,Update,Delete}NetworkPolicy.ts`, `useNetworkPolic{y,ies}.ts` |
| `subjects` | Shared access-subject panels reused by user detail (effective hosts, subject groups, group filter, devices). | `EffectiveHostsPanel`, `SubjectGroupsPanel`, `GroupFilterBar`, `UserDevicesTab`, `AllHostsBypassPill`; `drafts/subjectAccessDraft.ts`; `hooks/use{UserAccessDetail,SetUserAccess,ListUsersWithAccess}.ts` |
| `policy-audit` | Policy decision-cache snapshot + request simulation. | `components/{SimulateBar,PolicyUserTable,PolicyUserDrawer,NetworkPolicyCacheTab}.tsx`; `hooks/use{PolicyMap,PolicySimulate}.ts` |
| `anomalies` | Detected-anomaly presentation (dashboard section, full page, nav badge) — all three share one open-anomalies query. | `components/{AnomalySection,AnomaliesFilterBar,AnomalyRow,AnomalyAttributionChips,EvidenceList,SeverityIndicator}.tsx`; `hooks/use{Anomalies,OpenAnomalies,AcknowledgeAnomaly}.ts`; `constants.ts` (kind→label/icon/family, severity→color, shared open-query params) |
| `access-log` | Access-decision log list, filtering, and detail drawer. | `components/{AccessLogTable,AccessLogDetailDrawer,ColumnFilter,FilterableCell}.tsx`, `accessLogColumns.tsx`, `filterConfig.ts`; `hooks/useAccessLog*.ts` |
| `address-history` | Address-lease event list + chart. | `components/{AddressHistoryView,AddressHistoryTable}.tsx`, `addressHistoryColumns.tsx`; `hooks/useAddressHistory*.ts` |
| `settings` | Account + preferences tab bodies. | `AccountTab.tsx`, `PreferencesTab.tsx` |

## Shared components (`src/components/`)

Cross-cutting building blocks reused across surfaces:

| Component | Role |
|-----------|------|
| `layout/AppShell` | App frame: collapsible/resizable nav, header, footer (user + logout) |
| `EmptyState` | centered icon + title + optional description, with an optional `action` slot (e.g. first-run CTA) for zero-result views |
| `ErrorState` | inline `isError` branch for failed loads (Alert + `toErrorMessage`, optional `message`/`title`/`onRetry`); use where `ErrorBoundary` (crashes) and notifications (mutations) don't apply |
| `ErrorBoundary` | `AppErrorBoundary` — catches render crashes |
| `PageToolbar` | page title/subtitle + slot for filters/refresh controls |
| `ActiveFilterChips` | renders applied filters as removable chips |
| `CursorPagination` | keyset-pagination prev/next controls |
| `TimeRangePresetSelect`, `AutoRefreshSelect` | time-range preset + auto-refresh interval pickers |
| `TrafficLineChart` | shared traffic-over-time area chart (dashboard + access log) |
| `GeoCell` | country/ASN cell rendering (flag + label) |
| `InfoTooltip` | small "?" info tooltip |
| `BrandName` | product wordmark |

See `loading-empty-error-states.md` for the loading → error → empty → data convention.

## Contexts & hooks

| Location | Exports | Role |
|----------|---------|------|
| `contexts/DateTimePrefsContext.tsx` + `useDateTimePrefs.ts` | `DateTimePrefsProvider`, `useDateTimePrefs`, `useDateFormatter`, `usePickerValueFormat` | date-time locale preference, synced via `localStorage` + a custom event; provider mounted in `App.tsx` |
| `hooks/useUnsavedChangesGuard.tsx` | `useUnsavedChangesGuard(isDirty)` | native beforeunload prompt for unsaved changes (paired with router `useBlocker` in pages) |
| `hooks/useClipboard.ts` | `useClipboard` | copy-to-clipboard with transient "copied" state |
| `hooks/useFilterButtonLabels.ts` | `useFilterButtonLabels` | builds human labels for active filter buttons |

## Generated API layer (`src/lib/api`, `src/lib/api-client`)

`src/lib/api/` is generated by `@hey-api/openapi-ts` (`cd frontend && npm run generate:api`,
regenerated by `make api`): `sdk.gen.ts` (typed request functions), `@tanstack/react-query.gen.ts`
(TanStack Query `*Options`/`*Mutation` + query-key helpers), `zod.gen.ts` (zod schemas),
`types.gen.ts` / `schemas.gen.ts`, and the `client/`+`core/` runtime. `src/lib/api-client/` wraps
it: `config.ts` configures the shared `client`, `errors.ts` provides `ApiError` + `toApiError` /
`toErrorMessage`, and `index.ts` re-exports both. **Never edit generated files.** The schema-first
contract (the seam shared with the backend) is owned by [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).
