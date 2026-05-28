# Frontend Codebase Reference

> Last updated: 2026-05-29

This document is the **map** of the frontend codebase — what exists and where. For implementation conventions, hook patterns, testing scaffolds, and OpenAPI layering, see [`docs/patterns/_index.md`](frontend/docs/patterns/_index.md).

## Auth Flow

- `useCurrentUser` → `AuthContext (AuthProvider)` → `useAuth()` hook consumed by `ProtectedRoute` and `AppShell`
- `LoginForm` owns login form validation/submission and is rendered by `LoginPage`
- Login: POST /auth/login → invalidate `getCurrentUserQueryKey` → navigate (awaits invalidation)
- Logout: POST /auth/logout → `removeQueries()` (clear all) → navigate to /login

## UX Surfaces

### TrafficDashboardPage(`/dashboard`)
- Shows aggregated metrics for requests received, accepted/denied, response time, traffic over-time requests per service...
- Mostly a condensed centralized read-only overview of what has happened.

### DevicesPage (`/devices`)
- Create new device via modal
- List all devices (table with name, ID, key prefix, created date; manage link; delete with confirmation)

### DeviceDetailPage (`/devices/:deviceId`)
- Device dedicated page to show the addreses along with settings and rules of the device.
- Linked from the DevicesPage along with some links on tabs that mention the device name
**Addresses tab** (`DeviceAddressesTab`):
- Add new IP address (form; submit re-enables if IP already exists and was disabled)
- View all assigned addresses (table: IP, status dot, last updated, actions)
- Disable an active address (confirmation dialog)
- Re-enable an inactive address (click Enable in table row)
- 
**Rules tab** (`DeviceRulesTab`):
- Manage device rules config and status (enabled/disabled)
- Auto-expiry rule: set a TTL (seconds/minutes/days) after which addresses auto-expire
- Max active IPs: Number of allowed active IPs (ie 2)
 
**History tab** (`DeviceHistoryTab`):
- Shows the device address log. Allowing to track the enable/disable status of the addresses for this device
- Same information as in AddressLogPage but device scoped

**Settings** (`DeviceSettingsTab`):
- Allows changing device name, icon, type (mobile/static), description
- Allows changing ownership (user)
- Allows seeing API key prefix and regenerating it

### DeviceProvisioningPage (`/device-provisioning`)
- Allows generation invitation codes for devices to be used in the heartbeat-client application (android/desktop app)
- Lists pending/used invitations
- Allows viewing an invitation code or invalidating the invitation

### AccessLogPage (`/access-log`)
- Shows traffic over time (plot)
- Lists each request and allows filtering on them
- Allows checking a request details (headers)
- The intention is to be able to answer which IPs have requested verification and what was the result of it

### AddressHistoryPage (`/address-history`)
- Shows active address IP over time (plot)
- Lists each address update and allows filtering on it

### Settings (`/settings`)
- Allows viewing and changing user/server settings
- Has tabs to organize content

**Account tab** (`AccountTab`)
- View and change user profile (name, username and email)
- Changing current password

**Preferences tab** (`PreferencesTab`)
- Allows setting date-time locale preference

**Users tab** (`UsersTab`)
- Allows viewing existing users as well as creating a new one
- Allows promoting/demoting users (to admin or normal user)

### Access management (`/access/*`)

> Catalogued during the PW-64 audit (structural — owning components confirmed, detailed
> behaviours not yet documented here). Lives under `pages/access/` + the `host-access`,
> `network-policies`, `subjects` features.

- **HostsPage** (`/access/hosts`) — manages hosts/host-groups; tabbed UI (`HostsTab`,
  `HostGroupsTab`, `SuggestionsTab`) with a staged-changes bar for bulk reconcile.
- **HostGroupsPage** (`/access/host-groups`) — host group listing/membership
  (`GroupMembershipTables`, `GroupMetadataModal`).
- **NetworkPoliciesPage** (`/access/network-policies`) + **NetworkPolicyDetailPage** —
  list network policies (filterable by group) and a per-policy detail page.
- **UsersPage** (`/access/users`) + **UserDetailPage** — user administration list + detail.

### PolicyAuditPage (`/policy-audit`)
- Policy decision audit / simulation surface (`SimulateBar`).

### UserDevicesPage (`/devices` for non-admin scope)
- User-scoped device listing (composes `OwnerGroupList`, as does the admin `DevicesPage`).

## Shared components (`src/components/`)

Cross-cutting building blocks reused across surfaces:

- `EmptyState` — centered icon + title + description for zero-result views.
- `ErrorState` — inline `isError` branch for failed data loads (Alert + `toErrorMessage`,
  optional retry). Use where `ErrorBoundary` (crashes) and notifications (mutations) don't apply.
- `ErrorBoundary` — catches render crashes.
- `PageToolbar`, `ActiveFilterChips`, `CursorPagination`, `TimeRangePresetSelect`,
  `AutoRefreshSelect`, `TrafficLineChart`, `BrandName`, `layout/` (AppShell etc.).
