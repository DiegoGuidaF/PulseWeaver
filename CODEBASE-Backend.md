# Backend Codebase Reference

> Last updated: 2026-08-12

This document is the **map** of the backend codebase — what exists and where. For the system-level
overview (layering, the API seam, request flow, single-binary build), see
[`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md). For implementation conventions, scaffolds, and
testing patterns, see the workspace pattern library (`docs/patterns/backend/`).

Three tables of packages follow (domain, shared, infrastructure), then the central files and the app
wiring. "Key files" orients you inside a package; the **Critical Files** table further down lists the
most important files across the whole backend with their purpose.

---

## Domain Packages (`internal/`)

| Package | Owns | Key files |
|---------|------|-----------|
| `device` | Core domain: devices, their addresses (IPs), and device API keys. Emits address lifecycle events; observes user lifecycle to cascade device deletion. Provides device-API-key auth middleware. Serves `/devices/refs`, `/devices/{device_id}/addresses`, and the address-history events (`GET /address-history`) + histogram (`GET /address-history/histogram`) endpoints — an enriched, filterx-backed read model over `address_events` (`event_kind`, `ttl_risk`, `renewal_gap_seconds` derived server-side) — and exposes the batched `FleetRows` reader behind the fleet composer (`internal/queries/fleet_view.go`) — the only fleet query that knows about owner scoping. | `service.go`, `addresses.go`, `device_repository.go`, `address_repository.go`, `events.go`, `middleware.go`, `fleet_view.go`, `device_refs_view.go`, `address_view.go`, `address_history_view.go`, `address_history_enum_map.go`, `handler_address_history.go` |
| `auth` | User authentication (session cookies), users, and sessions. Emits user lifecycle events; `BootstrapAdmin` ensures one admin on startup. | `service.go`, `session.go`, `cookie.go`, `middleware.go`, `principal.go` |
| `devicepairing` | Device provisioning via short-lived pairing codes; the heartbeat client claims a code to receive a fresh device API key. Supersedes the former `registration` package. Exposes the batched, device-id-keyed `PairingsForDevices` reader for the fleet composer; it knows nothing about owners. | `service.go`, `pairing.go`, `code.go`, `repository.go`, `device_pairings_view.go` |
| `policy` | Forward-auth hot path: answers "can this IP reach this host?" from an in-memory cache (exact IP → CIDR network policy → deny). Observes address/user/host/network-policy changes; emits decisions to `accesslog`. Also serves the user-pivoted policy-map audit view (`GET /admin/policy-map`, DB-backed, per ADR-009) and the cache-backed simulate endpoint. | `service.go`, `cache.go`, `access.go`, `lifecycle.go`, `handler.go`, `decision.go`, `request.go`, `audit.go`, `observer.go`, `repository.go`, `policy_user_view.go` |
| `hosts` | Known hosts (FQDNs) and host groups; bulk reconciliation of membership. Notifies policy on change. Also serves the host list, host-groups list, and host-group detail views (`GET /admin/access/hosts`, `/host-groups`, `/host-groups/{group_id}`, DB-backed, per ADR-009). | `service.go`, `host.go`, `host_group.go`, `reconcile_hosts.go`, `reconcile_groups.go`, `host_view.go`, `host_group_view.go` |
| `useraccess` | Per-user host access: the bypass-host-check flag and host-group grants. Observes user lifecycle, notifies policy on grant changes. (Carved out of the former `hostaccess`.) Also serves `GET /owners/refs` and exposes `SubjectRows` — the subject identity/role/bypass/group read behind the fleet composer. | `service.go`, `user_access.go`, `repository.go`, `events.go`, `subject_view.go`, `owner_refs_view.go` |
| `networkpolicies` | CIDR-based network policies (named ranges + own bypass flag + grants); the second match tier in `policy`. Exposes `CacheEntry` to the policy cache; notifies policy on change. Also serves the network-policy list and detail views (`GET /admin/access/network-policies[/{policy_id}]`, DB-backed, per ADR-009). | `service.go`, `network_policy.go`, `repository.go`, `events.go`, `network_policy_view.go` |
| `lease` | Address lease TTL: disables addresses whose lease expired. Reads per-device config from `rule`; runs a `RunListener` and exposes `NewExpiryJob`. | `service.go`, `address_lease.go`, `expiry_job.go`, `repository.go` |
| `maxaddr` | Enforces the max active addresses per device. Observes address + rule changes; runs a `RunListener`. | `service.go` |
| `rule` | Per-device rules (lease TTL, max active address count). Emits rule change events. A read-only view joins the device domain's `addresses`/`devices` tables for a live active-address count (`max_active_addresses_view.go`), per ADR-009. Exposes the batched, device-id-keyed `RulesForDevices` reader for the fleet composer; it knows nothing about owners. | `service.go`, `rule.go`, `repository.go`, `events.go`, `max_active_addresses_view.go`, `device_rules_view.go` |
| `accesslog` | Async batch logging of policy decisions (`Sink` implements `policy.DecisionObserver`) **and** every read over `access_log`: the filterx-backed list (`GET /access-log`), one entry's full detail (`GET /access-log/{id}`, 404 once retention prunes it), the filtered allow/deny histogram (`GET /access-log/histogram`) and the country rollup. List, count and histogram share one WHERE builder (`accessLogConditions`), so the chart always reconciles with the table. | `sink.go`, `repository.go`, `access_log_view.go`, `access_log_histogram_view.go`, `country_stats_view.go`, `handler.go`, `handler_access_log.go` |
| `queries` | Reads belonging to no single domain: the SQL-free fleet composer (`fleet_view.go`, `GET /device-fleet`) and the analytical read models over event-scale tables. Shrinking under ADR-009 as single-domain views migrate to their owning domains (address history moved to `device`, the access log to `accesslog`); one view + handler file per surface. Folds join rows via `collate`. Still owns the host-suggestions view (shared with the dashboard's pending-suggestion count) alongside users and dashboard posture. | `repository.go`, `fleet_view.go`, `*_view.go`, `handler_*.go` |
| `rollup` | Hourly traffic + attribution aggregate tables; catch-up `RollupJob`; serves the dashboard read API (raw vs aggregate on `RawWindowThreshold`). | `job.go`, `traffic_rollup.go`, `traffic_reads.go`, `attribution_rollup.go`, `attribution_reads.go`, `handler.go`, `types.go` |
| `geoip` | IP → location/ASN enrichment from an MMDB (db-ip.com); background `RunUpdater` refresh. | `lookup.go`, `updater.go`, `result.go` |
| `health` | `GET /health` → `{"status":"ok","timestamp":…}`. | `health.go` |
| `timebucket` | Shared time-bucket granularity settings + parsing (rollup, dashboard, address history, …), including `GranularityForWindow(d)` — maps a query window to a bucket size, shared by any histogram endpoint that auto-selects granularity from the window rather than accepting it as a param. | `granularity.go` |

`policy` and `rollup` are the read-heavy hot paths; everything else flows handler → service →
repository. Cross-domain reads live in the consuming domain's own `*_view.go` files (ADR-009);
`queries` is the shrinking legacy home for views not yet migrated.

## Shared / Utility Packages (`internal/`)

| Package | Owns | Key files |
|---------|------|-----------|
| `ids` | Typed `int64` ID newtypes (`DeviceID`, `UserID`, `HostID`, `HostGroupID`, `NetworkPolicyID`, …) shared across domains for type-safe boundaries. | `types.go` |
| `filterx` | Column-allowlist registry for filter/sort/keyset pagination (ADR-007): per-column operators, the NULL-safe `not_in` rule, the relational `EXISTS` template, and the opaque server-issued cursor. Imported by any domain's read models (`accesslog`, `device`). | `filterx.go`, `cursor.go`, `values.go` |
| `collate` | Generic `Collapse`: folds flat parent×child SQL rows (LEFT JOINs) into nested DTOs in first-seen order. Replaces the hand-written "seen map" idiom in `queries`. | `collate.go` |
| `slicex` | Generic slice helpers absent from the stdlib `slices` (`Dedup`, sorted `Intersect`). | `slicex.go` |
| `buildmeta` | Build identity (version/commit/build time) injected at link time via `-ldflags -X`, served by the authenticated `GET /api/v1/version`. Named to dodge the stdlib collisions `version` and `buildinfo`. | `buildmeta.go`, `handler.go` |

## Infrastructure Packages (`internal/`)

| Package | Owns | Key files |
|---------|------|-----------|
| `app` | Dependency injection, startup, and observer wiring (see **App Wiring** below). | `app.go` |
| `config` | Env var parsing (`caarlos0/env/v11`); optional `.env` (godotenv). | `config.go` |
| `database` | Single SQLite connection (sqlx, WAL, `MaxOpenConns=1`); migrations embedded via `embed.FS`. | `sqlite.go`, `db.go`, `transactor.go`, `migrations/` |
| `httpserver` | Chi router + global middleware chain; `/api/v1` sub-router; graceful shutdown; OpenAPI security-scheme validation; build-tag-gated pprof. | `server.go`, `routes.go`, `lifecycle.go`, `authentication.go`, `middleware.go`, `contention.go` |
| `httpapi` | `oapi-codegen` output: DTOs + strict handler interface. The contract is owned by `ARCHITECTURE.md` (schema-first, `make api`); this is only the backend's generated side. **Never modify the `*.gen.go` files.** The package also holds hand-written companions to those DTOs — `utctime.go`, `nullable.go`, `const.go`, `context.go`, and `geo.go` (`GeoInfoFromResult`, the one `geoip.Result → GeoInfo` mapper, shared by device/queries/policy/rollup — it lives here because `geoip` is infrastructure and must not import the HTTP layer). | `server.gen.go`, `geo.go` |
| `scheduler` | Generic periodic `Job` runner (ticks at `RULE_CHECK_INTERVAL`, `AddJob`); retention job prunes logs/events/aggregates. | `service.go`, `retention_runner.go` |
| `logging` | slog helpers: logger-in-context (`FromCtx`/`Enrich`), canonical attribute keys, request-ID-stamping handler. | `ctx.go`, `attribute_keys.go`, `handler.go` |
| `testdb` | In-memory SQLite for integration tests. | `setup.go` |
| `testutils` | Test scaffolding: `SetupIntegrationServer`, admin principal helpers, `NoopTransactor`, typed API client, world seeders. | `server.go`, `seeder*.go`, `apiclient.go`, `auth.go`, `db_transactor.go` |
| `integrationtest` | Cross-domain lifecycle tests (`test`-tagged) exercising the wired app end-to-end. | `*_test.go` |
| `ui` | `embed.FS` SPA serving (prod build tag) / dev stub pointing at Vite. | `ui_prod.go`, `ui_dev.go` |

The `/api/v1` middleware chain (in `httpserver`): rate limit → OpenAPI request validation →
principal-from-cookie → principal-from-API-key → generated strict handler.

---

## Critical Files

| File | Purpose |
|------|---------|
| `cmd/api/main.go` | Entry point; signal handling |
| `internal/app/app.go` | Dependency injection, startup, observer wiring |
| `internal/config/config.go` | All env vars; validation in `Load()` |
| `internal/httpserver/server.go` | Chi router construction; global middleware chain |
| `internal/httpserver/routes.go` | `CompositeHandler`, route registration, `/api/v1` sub-router middleware |
| `internal/httpserver/lifecycle.go` | `StartAndWait` — graceful HTTP server shutdown |
| `internal/httpserver/authentication.go` | OpenAPI security scheme validation |
| `internal/httpapi/server.gen.go` | Generated DTOs and strict handler interface |
| `internal/ids/types.go` | Typed ID newtypes shared across domains |
| `internal/device/service.go` | `Service`, interfaces, constructor; device CRUD + API key methods |
| `internal/device/addresses.go` | Address lifecycle: `RegisterAddressActivity`, `DisableAddress(es)`, `GetAddressHistory`; observer fan-out |
| `internal/device/device_repository.go` | DB queries for `devices` and `device_api_keys` |
| `internal/device/address_repository.go` | DB queries for `addresses` and `address_events`; history SQL |
| `internal/device/events.go` | `AddressEvent`/`EventType` + `DeviceEvent`/`DeviceEventType` — domain events emitted to observers |
| `internal/queries/fleet_view.go` | `FleetComposer` — the SQL-free composer behind `GET /device-fleet`; consumer-side reader interfaces, stitches owners→devices→rules/pairing, sums owner aggregates |
| `internal/useraccess/subject_view.go` | `SubjectRows` — subject identity, role/bypass_host_check (rendered, not interpreted) and host-group membership, optionally narrowed to one user |
| `internal/useraccess/owner_refs_view.go` | `GetOwnerRefs` — flat `{id, display_name}` owner references for pickers |
| `internal/device/fleet_view.go` | `FleetRows` — devices in list shape tagged with their owner, optionally narrowed to one owner; the only owner-scoped query in the fleet composition |
| `internal/device/device_refs_view.go` | `GetDeviceRefs` — flat `{id, name, owner_id}` device references for pickers |
| `internal/device/address_history_view.go` | `addressHistoryBase` — the shared FROM + WHERE builder (not just a WHERE) over `addressHistoryEnriched`'s derived table, where `event_kind`/`ttl_risk`/`renewal_gap_seconds` are computed unbounded by time so classification stays stable as the caller pans the window; both `GetAddressHistoryEvents` and `GetAddressHistoryHistogram` build on it. SQLite will not push the `from`/`to` predicate past the CTE's window function, so each query materializes the renewal-gap CTE over the whole table before filtering — `event_kind`'s per-row lookup is what `idx_address_events_address_id_id` (migration 000033) exists to serve |
| `internal/rule/device_rules_view.go` | `RulesForDevices` — configured rules keyed by device id; unparseable configs are skipped, not fatal |
| `internal/devicepairing/device_pairings_view.go` | `PairingsForDevices` — latest pairing per device, keyed by device id |
| `internal/devicepairing/service.go` | Pairing create/claim; mints a device API key on claim |
| `internal/policy/service.go` | `Service`, constructor, provider interfaces; cache state |
| `internal/policy/cache.go` | In-memory IP + network-policy cache rebuild; deny-wins intersection |
| `internal/policy/access.go` | `Decide`, `VerifyAccess` — access decision entry points |
| `internal/policy/lifecycle.go` | `RunListener` + observer callbacks; change-signal handling |
| `internal/policy/handler.go` | `HandleForwardAuthIP` (Bearer + client IP), `SimulatePolicyAccess`, `GetPolicyUserMap` |
| `internal/hosts/service.go` | Known host/group management; notifies policy on change |
| `internal/useraccess/service.go` | Per-user bypass + group grants; observes users, notifies policy |
| `internal/networkpolicies/service.go` | CIDR network-policy CRUD; `CacheEntry` source for policy |
| `internal/accesslog/sink.go` | `Sink` — implements `policy.DecisionObserver`; batch-inserts decision events |
| `internal/accesslog/access_log_view.go` | `accessLogConditions` — the one WHERE builder behind the list, its `COUNT` and the histogram; plus the registry, the slim list row and the by-id detail read. Cursor and limit attach to the page builder only |
| `internal/accesslog/access_log_histogram_view.go` | The filtered allow/deny series. Always aggregates `access_log` itself, at every window width — the hourly rollups carry no attribution and omit the in-flight hour, so they cannot reconcile with the table. Folds onto `timebucket.Sequence`, so every bucket in the window is present |
| `internal/queries/repository.go` | Cross-domain read repository backing the list/filter views |
| `internal/queries/host_suggestions.go` | `pendingHostSuggestions` — shared raw/aggregate-dispatching implementation behind both the suggestions page and the dashboard's pending-suggestion count |
| `internal/filterx/filterx.go` | Column-allowlist registry for filter/sort/keyset pagination (ADR-007) |
| `internal/rollup/job.go` | `RollupJob` catch-up scheduler for hourly aggregates |
| `internal/lease/expiry_job.go` | `ExpiryJob` — disables addresses with an expired lease |
| `internal/scheduler/service.go` | Generic `Job` runner; `AddJob`, `RunSchedule` |
| `internal/scheduler/retention_runner.go` | `NewRetentionJob` — prunes access logs, address events, aggregates |
| `internal/logging/ctx.go` | `FromCtx`, `Enrich` — logger-in-context |
| `internal/logging/attribute_keys.go` | Canonical slog attribute key constants used package-wide |
| `internal/testutils/server.go` | `SetupIntegrationServer` for handler tests |
| `api/openapi.yaml` | API schema source of truth (contract owned by `docs/ARCHITECTURE.md`) |

---

## App Wiring (internal/app/app.go)

**Construction order:** DB → auth → device → devicepairing → geoip → hosts → useraccess →
networkpolicies → policy → rule → rollup → accesslog → queries → lease → maxaddr → buildmeta →
scheduler → HTTP server. After construction: `ExecuteScheduledRules` (disable stale addresses before serving),
`BootstrapAdmin`, then `policyService.Initialize` (warm the IP cache).

**Observer registrations:**
- `deviceService.AddAddressObserver`: lease, policy, maxaddr
- `deviceService.AddDeviceObserver`: policy (ownership changes → cache refresh)
- `authService.AddUserObserver`: useraccess, device
- `hostsService.AddObserver`: policy
- `userAccessService.AddObserver`: policy
- `networkPoliciesService.AddObserver`: policy
- `ruleService.AddRuleObserver`: lease, maxaddr
- `policyService.AddDecisionObserver`: accessLogSink

**Scheduler jobs (`AddJob`):** `lease.NewExpiryJob(deviceService)`, `rollupRepo.NewRollupJob`,
`scheduler.NewRetentionJob(accessLogRepo, deviceRepo, rollupRepo, …)`.

**Goroutines started in `RunBackground`:** `policy.RunListener`, `lease.RunListener`,
`maxaddr.RunListener`, `scheduler.RunSchedule`, `accessLogSink.Run`, `geoip.RunUpdater`. `Run` adds
`httpserver.StartAndWait` and the build-tag-gated `StartPprofServer` on top.

---

## Test Infrastructure

| Piece | Purpose |
|-------|---------|
| `internal/testdb.Setup` | In-memory SQLite (`database.SQLite`) for a single test; runs migrations, returns a teardown func |
| `internal/testutils/server.go` — `SetupIntegrationServer` | Builds a full `*app.App` (same DI graph as `app.New`) against `testdb`, the only sanctioned setup entry point for handler/integration tests |
| `internal/testutils/seeder*.go` — `NewSeeder(t)` | Declares only the entities a test needs, resolves dependency order, applies defaults; `SeedFullWorld` seeds a rich cross-domain dataset and is reserved for the data-complex read models — `internal/queries` and `internal/accesslog` |
| DSN `_time_format=sqlite&_texttotime=1` | Makes `modernc.org/sqlite` scan `DATETIME` columns straight into `time.Time` and format bound `time.Time` args the way SQLite's date functions (`strftime`, used by rollups) expect; see `internal/database/dbtime.go` — `DBTime` exists only for aggregate functions (`MAX`/`MIN`) that return `TEXT` even over `DATETIME` columns despite this flag |

DI wiring for tests mirrors production: `SetupIntegrationServer` calls the same constructors as
`app.New`/`app.NewWithConfigAndLogger`, so a test server has the identical observer graph and
scheduler jobs described above — no service is ever constructed by hand in a test.
