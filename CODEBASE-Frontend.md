# Frontend Codebase Reference

> Last updated: 2026-05-29

This document is the **map** of the frontend codebase — what exists and where. For implementation
conventions, hook patterns, testing scaffolds, and OpenAPI layering, see the pattern library at
[`../docs/patterns/frontend/_index.md`](../docs/patterns/frontend/_index.md).

## Routing & layout

Routes are defined in `lib/routes.ts` (`ROUTES` map + `buildRoute` helpers) and wired in `App.tsx`.
Every route except `/login` is wrapped in `ProtectedRoute` + `AppShell`.

| Route | Page component | Notes |
|-------|----------------|-------|
| `/login` | `LoginPage` | unauthenticated |
| `/` | → redirect to `/dashboard` | |
| `/dashboard` | `TrafficDashboardPage` | |
| `/devices` | `DevicesPage` | all devices grouped by owner |
| `/devices/owners/:ownerId` | `UserDevicesPage` | owner-scoped device workspace (the device-detail surface) |
| `/access/hosts` | `HostsPage` | |
| `/access/host-groups` | `HostGroupsPage` | |
| `/access/users` + `/access/users/:id` | `UsersPage`, `UserDetailPage` | |
| `/access/network-policies` + `/:id` | `NetworkPoliciesPage`, `NetworkPolicyDetailPage` | |
| `/access-log` | `AccessLogPage` | |
| `/address-history` | `AddressHistoryPage` | |
| `/policy-audit` | `PolicyAuditPage` | nav label is "Access Policy Cache" |
| `/settings` | `SettingsPage` | |
| `*` | `NotFoundPage` | |

**Nav (`components/layout/AppShell.tsx`, `navGroups`):** Dashboard · Devices (Devices) ·
Access (Hosts, Host Groups, Users, Network Policies) · Auditing (Access Logs, IP Address Logs,
Access Policy Cache) · Settings.

## Auth Flow

- `useCurrentUser` → `AuthContext` (`AuthProvider`) → `useAuth()` hook consumed by `ProtectedRoute`
  and `AppShell`. Lives in `features/auth/` (`AuthContext.tsx`, `auth-context.ts`, `ProtectedRoute.tsx`).
- `LoginForm` owns login validation/submission, rendered by `LoginPage`.
- Login: POST `/auth/login` → invalidate `getCurrentUserQueryKey` → navigate (awaits invalidation).
- Logout: POST `/auth/logout` → `removeQueries()` (clear all) → navigate to `/login`.
- Global 401 handling lives in `main.tsx` `QueryCache.onError` (see `error-handling.md`).

## UX Surfaces

### TrafficDashboardPage (`/dashboard`)
- Condensed read-only overview: requests received, accepted/denied, response time, traffic
  over time, per-service breakdown, top denied IPs, geographic stats.
- Feature: `features/dashboard/` — `DashboardView` composes `DashboardStatCards`, `TrafficLineChart`,
  `ServiceBarChart`/`ServiceDonutChart`, `TopCountriesTable`, `TopDeniedIPsTable`, `CountryStatsSection`,
  `AccessMap`.

### DevicesPage (`/devices`)
- Admin-facing list of all devices grouped by owner — renders `OwnerGroupList`.
- Create new device via `CreateDeviceModal`.

### UserDevicesPage (`/devices/owners/:ownerId`)
- Owner-scoped device workspace; the device-detail surface. A sidebar device list + a per-device
  tabbed panel. Tab is tracked in the `?tab=` search param.
- **Addresses** (`DeviceAddressesTab`) — add/disable/re-enable IPs; table of assigned addresses.
- **Rules** (`DeviceRulesTab`) — `AddressLeaseRuleCard` (auto-expiry TTL) + `MaxActiveIpsRuleCard`
  (cap on active IPs); each toggle-gated.
- **Pairing** (`DevicePairingTab`) — generate/revoke a one-time pairing code for the heartbeat
  client (`PairingCreationForm`, `PairingCodeDisplay`); recent-codes history. Tab shows an
  indicator when the device is in `PENDING_CLAIM`/`EXPIRED_CLAIM`.
- **History** (`DeviceHistoryTab`) — device-scoped address event log.
- **Settings** (`DeviceSettingsTab`) — profile (name/icon/type/description via `DeviceProfileCard`),
  API key prefix + regenerate/remove, transfer ownership, delete device (danger zone).

### AccessLogPage (`/access-log`)
- Traffic-over-time plot + filterable per-request list; row opens `AccessLogDetailDrawer`
  (request/device/location/headers). Answers "which IPs requested verification and the outcome".

### AddressHistoryPage (`/address-history`)
- Active-address-over-time plot + filterable list of address update events.

### Access management (`/access/*`)
- **HostsPage** (`/access/hosts`) — tabbed (`HostsTab`, `SuggestionsTab`) with a staged-changes bar
  (`StagedChangesBar`) for bulk reconcile; drafts live in `features/host-access/drafts/`.
- **HostGroupsPage** (`/access/host-groups`) — group master/detail + membership
  (`GroupMasterList`, `GroupDetailPanel`, `GroupMembershipTables`, `GroupMetadataModal`).
- **UsersPage** + **UserDetailPage** — user administration list + per-user detail (effective host
  access via the shared `features/subjects/` panels).
- **NetworkPoliciesPage** + **NetworkPolicyDetailPage** — list (filterable by group) + per-policy
  detail; `CreateNetworkPolicyModal`, `DeleteNetworkPolicyModal`, `NetworkPoliciesTable`.

### PolicyAuditPage (`/policy-audit`, nav "Access Policy Cache")
- Policy decision cache / simulation surface — `SimulateBar`, `NetworkPolicyCacheTab`,
  `PolicyUserTable`, `PolicyUserDrawer`.

### Settings (`/settings`)
- Tabbed (`SettingsPage`); user/server settings.
- **Account** (`AccountTab`) — view/change profile (name, username, email) + change password.
- **Preferences** (`PreferencesTab`) — date-time locale preference.
- _Note: user administration moved to Access › Users (`/access/users`); Settings no longer has a Users tab._

## Features map (`src/features/`)

`auth` · `dashboard` · `devices` · `device-pairing` · `host-access` · `network-policies` ·
`subjects` (shared access-subject panels: effective hosts, subject groups, group filter) ·
`policy-audit` · `access-log` · `address-history` · `settings`. Each owns its components, hooks,
and (where relevant) `constants.ts`, `drafts/`, and config files.

## Shared components (`src/components/`)

Cross-cutting building blocks reused across surfaces:

- `EmptyState` — centered icon + title + optional description, with an optional `action` slot
  (e.g. a first-run CTA button) for zero-result views.
- `ErrorState` — inline `isError` branch for failed data loads (Alert + `toErrorMessage`, optional
  `message`/`title` overrides and `onRetry`). Use where `ErrorBoundary` (crashes) and notifications
  (mutations) don't apply. See `loading-empty-error-states.md` for the full loading→error→empty→data
  convention.
- `ErrorBoundary` — catches render crashes.
- `PageToolbar`, `ActiveFilterChips`, `CursorPagination`, `TimeRangePresetSelect`,
  `AutoRefreshSelect`, `TrafficLineChart`, `BrandName`, and `layout/` (`AppShell` etc.).
