# Cross-Domain Queries

Each domain's repository must only query tables it owns. Cross-domain data access belongs in `internal/queries/`.

## The rule

A repository in `internal/device/` must not JOIN against `users`. A repository in `internal/hostaccess/` must not SELECT from `users`. If the SQL spans two or more domain table sets, it does not belong in either domain's repository.

## Where cross-domain reads go

`internal/queries/` is the designated package for read-only views that join across domains. It is intentionally not a domain — it has no business logic, no service, no events. Its repository is allowed to join any combination of tables.

```
internal/queries/repository.go   ← cross-domain SELECTs
internal/queries/handler.go      ← HTTP handler (no service layer)
internal/queries/<view>.go        ← one file per view type
```

Examples already in this codebase: `GetAllUsers` (users + user_host_settings), `GetKnownHostsWithStats` (known_hosts + access_log + user_allowed_hosts + user_allowed_host_groups), `GetDevices` (devices + addresses + users).

## Why not just query the other table?

The original `hostaccess.Repository.GetAllUserHostAccess` read `bypass_host_allowlist` directly from the `users` table. When the column moved to `user_host_settings`, every callsite in hostaccess had to change. A cross-domain join in a domain repository creates an invisible dependency: schema changes in another domain silently break unrelated repositories.

## Cross-domain writes: use the observer pattern

Reads go to `queries`. Cross-domain writes or lifecycle reactions go through domain events — see `observer-pattern.md`. `hostaccess.Service.OnUserEvent` is the canonical example: it reacts to `auth.UserEvent` to keep its own tables consistent without auth ever importing hostaccess.

---
**Verified against:** `internal/queries/`, `internal/hostaccess/repository.go`, `internal/auth/events.go`
**Applies to:** any new repository method that needs data from another domain's tables
**Last verified:** 2026-04-21
