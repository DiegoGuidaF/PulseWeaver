# AI Overview

## Tech Stack

- Language: Go
- Architecture: layered / clean-ish
    - API layer: generated types from OpenAPI (`oapi-codegen`)
    - Service layer: business use cases (e.g. `auth.Service`, `device.Service`)
    - Domain layer: invariants and entity constructors (e.g. `auth.User`, `auth.NewUser`)
    - Repository layer: interfaces + `sqlx` implementations
- DB: SQLite
- Migrations / Schema: SQLite, explicit SQL using `golang-migrate`
- HTTP: chi + `oapi-codegen` routers + kin-openapi validation middleware

## High-level rules

- Use **standard library + `sqlx` + `oapi-codegen`**. Do not introduce ORMs or heavy frameworks.
- Keep validations and invariants in **domain constructors** (e.g. `auth.NewUser`, `validateUsername`).
- The **service layer orchestrates repos + domain**, but does not know SQL details.
- The **repository layer owns SQL** (including error mapping from DB errors → domain errors).
- **Transaction Management:** Repositories provide `RunInTx` for transactional operations. Services use this to ensure atomicity when multiple repository calls must succeed or fail together (e.g., creating a user and session atomically).
- OpenAPI types are **transport only**; map them to/from domain types explicitly.
