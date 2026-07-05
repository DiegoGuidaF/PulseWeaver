# Backend Codebase Reference

> Last updated: 2026-07-04 (novelty & geo-velocity detectors)

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
| `device` | Core domain: devices, their addresses (IPs), and device API keys. Emits address lifecycle events; observes user lifecycle to cascade device deletion. Provides device-API-key auth middleware. | `service.go`, `addresses.go`, `device_repository.go`, `address_repository.go`, `events.go`, `middleware.go` |
| `auth` | User authentication (session cookies), users, and sessions. Emits user lifecycle events; `BootstrapAdmin` ensures one admin on startup. | `service.go`, `session.go`, `cookie.go`, `middleware.go`, `principal.go` |
| `devicepairing` | Device provisioning via short-lived pairing codes; the heartbeat client claims a code to receive a fresh device API key. Supersedes the former `registration` package. | `service.go`, `pairing.go`, `code.go`, `repository.go` |
| `policy` | Forward-auth hot path: answers "can this IP reach this host?" from an in-memory cache (exact IP → CIDR network policy → deny). Observes address/user/host/network-policy changes; emits decisions to `accesslog`. | `service.go`, `cache.go`, `access.go`, `lifecycle.go`, `handler.go`, `decision.go`, `request.go`, `audit.go`, `observer.go` |
| `hosts` | Known hosts (FQDNs) and host groups; bulk reconciliation of membership. Notifies policy on change. | `service.go`, `host.go`, `host_group.go`, `reconcile_hosts.go`, `reconcile_groups.go` |
| `useraccess` | Per-user host access: the bypass-host-check flag and host-group grants. Observes user lifecycle, notifies policy on grant changes. (Carved out of the former `hostaccess`.) | `service.go`, `user_access.go`, `repository.go`, `events.go` |
| `networkpolicies` | CIDR-based network policies (named ranges + own bypass flag + grants); the second match tier in `policy`. Exposes `CacheEntry` to the policy cache; notifies policy on change. | `service.go`, `network_policy.go`, `repository.go`, `events.go` |
| `lease` | Address lease TTL: disables addresses whose lease expired. Reads per-device config from `rule`; runs a `RunListener` and exposes `NewExpiryJob`. | `service.go`, `address_lease.go`, `expiry_job.go`, `repository.go` |
| `maxaddr` | Enforces the max active addresses per device. Observes address + rule changes; runs a `RunListener`. | `service.go` |
| `rule` | Per-device rules (lease TTL, max active address count). Emits rule change events. | `service.go`, `rule.go`, `repository.go`, `events.go` |
| `accesslog` | Async batch logging of policy decisions: `Sink` implements `policy.DecisionObserver`; serves audit reads (e.g. deny-reason list). | `sink.go`, `handler.go`, `repository.go` |
| `queries` | Cross-domain read side (lite CQRS) for the frontend's list/filter views; one view + handler file per surface. Folds join rows via `collate`. | `repository.go`, `*_view.go`, `handler_*.go`, `filterx/` |
| `rollup` | Hourly traffic + attribution aggregate tables; catch-up `RollupJob`; serves the dashboard read API (raw vs aggregate on `RawWindowThreshold`). | `job.go`, `traffic_rollup.go`, `traffic_reads.go`, `attribution_rollup.go`, `attribution_reads.go`, `handler.go`, `types.go` |
| `geoip` | IP → location/ASN enrichment from an MMDB (db-ip.com); background `RunUpdater` refresh. | `lookup.go`, `updater.go`, `result.go` |
| `anomaly` | Periodic detection scan over the access log + traffic aggregates. Detectors are pure readers behind a `Detector` interface; the `ScanJob` owns the incremental watermark and the deduplicated finding upsert. The novelty family additionally reports `ProfileObservation`s via the optional `ProfileLearner` interface, which the job persists to `device_profiles` inside the same scan transaction. Findings persist to `anomalies` for review (no verify-path impact). Owns the single-domain acknowledge endpoint (`handler.go`) and the retention prune (`DeleteAnomaliesOlderThan`); the cross-domain list read model lives in `queries` (`anomaly_view.go`). | `anomaly.go`, `detector.go`, `job.go`, `repository.go`, `reads.go`, `rules.go`, `probing.go`, `volume.go`, `volume_reads.go`, `baseline.go`, `novelty.go`, `novelty_reads.go`, `travel.go`, `handler.go`, `errors.go` |
| `health` | `GET /health` → `{"status":"ok","timestamp":…}`. | `health.go` |
| `timebucket` | Shared time-bucket granularity settings + parsing (rollup, dashboard, …). | `granularity.go` |

`policy` and `rollup` are the read-heavy hot paths; everything else flows handler → service →
repository. Cross-domain reads go through `queries`, never across domain repositories.

## Shared / Utility Packages (`internal/`)

| Package | Owns | Key files |
|---------|------|-----------|
| `ids` | Typed `int64` ID newtypes (`DeviceID`, `UserID`, `HostID`, `HostGroupID`, `NetworkPolicyID`, …) shared across domains for type-safe boundaries. | `types.go` |
| `collate` | Generic `Collapse`: folds flat parent×child SQL rows (LEFT JOINs) into nested DTOs in first-seen order. Replaces the hand-written "seen map" idiom in `queries`. | `collate.go` |
| `slicex` | Generic slice helpers absent from the stdlib `slices` (`Dedup`, sorted `Intersect`). | `slicex.go` |

## Infrastructure Packages (`internal/`)

| Package | Owns | Key files |
|---------|------|-----------|
| `app` | Dependency injection, startup, and observer wiring (see **App Wiring** below). | `app.go` |
| `config` | Env var parsing (`caarlos0/env/v11`); optional `.env` (godotenv). | `config.go` |
| `database` | Single SQLite connection (sqlx, WAL, `MaxOpenConns=1`); migrations embedded via `embed.FS`. | `sqlite.go`, `db.go`, `transactor.go`, `migrations/` |
| `httpserver` | Chi router + global middleware chain; `/api/v1` sub-router; graceful shutdown; OpenAPI security-scheme validation; build-tag-gated pprof. | `server.go`, `routes.go`, `lifecycle.go`, `authentication.go`, `middleware.go`, `contention.go` |
| `httpapi` | `oapi-codegen` output: DTOs + strict handler interface. The contract is owned by `ARCHITECTURE.md` (schema-first, `make api`); this is only the backend's generated side. **Do not modify.** | `server.gen.go` |
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
| `internal/device/events.go` | `AddressEvent`, `EventType` — domain events emitted to observers |
| `internal/devicepairing/service.go` | Pairing create/claim; mints a device API key on claim |
| `internal/policy/service.go` | `Service`, constructor, provider interfaces; cache state |
| `internal/policy/cache.go` | In-memory IP + network-policy cache rebuild; deny-wins intersection |
| `internal/policy/access.go` | `Decide`, `VerifyAccess` — access decision entry points |
| `internal/policy/lifecycle.go` | `RunListener` + observer callbacks; change-signal handling |
| `internal/policy/handler.go` | `HandleForwardAuthIP` (Bearer + client IP), `SimulatePolicyAccess` |
| `internal/hosts/service.go` | Known host/group management; notifies policy on change |
| `internal/useraccess/service.go` | Per-user bypass + group grants; observes users, notifies policy |
| `internal/networkpolicies/service.go` | CIDR network-policy CRUD; `CacheEntry` source for policy |
| `internal/accesslog/sink.go` | `Sink` — implements `policy.DecisionObserver`; batch-inserts decision events |
| `internal/queries/repository.go` | Cross-domain read repository backing the list/filter views |
| `internal/queries/filterx/filterx.go` | Column-allowlist registry for filter/sort/keyset pagination (ADR-007) |
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
networkpolicies → policy → rule → accesslog → queries → lease → maxaddr → rollup → scheduler → HTTP
server. After construction: `ExecuteScheduledRules` (disable stale addresses before serving),
`BootstrapAdmin`, then `policyService.Initialize` (warm the IP cache).

**Observer registrations:**
- `deviceService.AddAddressObserver`: lease, policy, maxaddr
- `authService.AddUserObserver`: useraccess, device
- `hostsService.AddObserver`: policy
- `userAccessService.AddObserver`: policy
- `networkPoliciesService.AddObserver`: policy
- `ruleService.AddRuleObserver`: lease, maxaddr
- `policyService.AddDecisionObserver`: accessLogSink

**Scheduler jobs (`AddJob`):** `lease.NewExpiryJob(deviceService)`, `rollupRepo.NewRollupJob`,
`scheduler.NewRetentionJob(accessLogRepo, deviceRepo, rollupRepo, …)`, and `anomaly.NewScanJob(…)`
when `Anomaly.Enabled`.

**Goroutines started in `RunBackground`:** `policy.RunListener`, `lease.RunListener`,
`maxaddr.RunListener`, `scheduler.RunSchedule`, `accessLogSink.Run`, `geoip.RunUpdater`. `Run` adds
`httpserver.StartAndWait` and the build-tag-gated `StartPprofServer` on top.
