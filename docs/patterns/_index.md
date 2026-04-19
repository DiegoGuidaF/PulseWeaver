# Backend Pattern Index

> Before implementing a feature, scan this index and read every pattern that applies.
> After implementing, check the [self-improvement protocol](../../../project/workflow/WORKFLOW.md#pattern-maintenance).

| Pattern | File | Use when | Avoid when | Refs |
|---------|------|----------|------------|------|
| Handler structure | `handler-structure.md` | Adding or modifying an HTTP endpoint | Working on business logic (that's the service layer) | `device/handler.go`, `auth/handler.go` |
| Service layer | `service-layer.md` | Adding business logic, domain validation, cross-domain interfaces | Writing SQL or HTTP-level code | `device/service.go`, `auth/service.go` |
| Repository layer | `repository-layer.md` | Adding DB queries, transactions, constraint handling | Writing business logic (that's the service layer) | `device/repository.go`, `lease/repository.go` |
| Observer pattern | `observer-pattern.md` | Reacting to domain events (address changes, etc.) | Simple request-response flows with no side effects | `device/events.go`, `policy/service.go`, `lease/service.go` |
| Background service lifecycle | `background-service-lifecycle.md` | Adding a long-running goroutine, listener, or scheduler | One-shot operations that complete in a request | `policy/service.go`, `lease/service.go`, `scheduler/service.go` |
| Schema-first API | `schema-first-api.md` | Adding or modifying any API endpoint | Internal-only types not exposed via HTTP | `api/openapi.yaml`, `httpapi/server.gen.go` |
| Config pattern | `config-pattern.md` | Adding a new env var or configuration option | Runtime-mutable settings (not supported) | `config/config.go` |
| Logging context | `logging-context.md` | Adding log statements in handlers or services | Generated code or test helpers | `logging/ctx.go` |
| Pointer conventions | `pointer-conventions.md` | Declaring structs, return types, or struct fields with pointers | — | — |
| ID types | `id-types.md` | Defining or using entity/foreign-key ID fields in Go, OpenAPI, or TypeScript | — | `auth/user.go`, `device/device.go`, `api/components/schemas/common.yaml` |
| Database transactions | `database-transactions.md` | Wrapping multi-step operations atomically in services or repos | Single-query methods that need no tx | `internal/database/db.go`, `internal/database/transactor.go` |
| Migrations | `migrations.md` | Writing any migration file | — | `internal/database/migrations/` |
| Handler tests | `handler-tests.md` | Writing tests for HTTP endpoints | Testing business logic (use service tests) | `device/handler_test.go`, `testutils/server.go` |
| Service tests | `service-tests.md` | Writing tests for service methods | Testing SQL/HTTP (use repository/handler tests) | `device/service_test.go` |
| Repository tests | `repository-tests.md` | Writing tests for repository methods | Testing business logic (use service tests) | `device/repository_test.go`, `testdb/` |
| Access log supplemental tables | `access-log-supplemental.md` | Adding optional detail data to access log rows (sparse child tables) | Data that is always present — add a column instead | `accesslog/repository.go`, `access_log_geoip`, `access_log_ip_devices` |
