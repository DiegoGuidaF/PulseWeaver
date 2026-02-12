# Go Style Guidelines (for AI)

## General

- Prefer explicit, boring Go. Avoid magic.
- Handle errors inline with early returns (no error collections or custom monads).
- Use small, focused functions and methods.
- Keep dependencies flowing: handler → service → repo → db.

## Services

- Service structs (e.g. `auth.Service`) own:
    - Repositories (interfaces).
    - Logger.
    - Optional config.
- Service methods can use helper **functions** for pure logic:
    - `checkPassword`, `hashRawToken`, `generateSecurePassword` should be plain funcs.
- Methods on `Service` are only for operations that logically involve its dependencies.

## Domain

- Domain entities live under `internal/<bounded-context>` (e.g. `internal/auth/user.go`).
- Use constructors like `NewUser(...) (*User, error)` to enforce:
    - Username invariants.
    - DisplayName invariants.
    - Password validation + hashing.
- **Domain Separation:** Domain must never depend on HTTP or OpenAPI types.
    - **Handler Layer:** Extracts values from OpenAPI types (`api.CreateUserRequest`) and passes primitive values to services.
    - **Service Layer:** Uses domain constructors (`NewUser`) which perform all business validation.
    - **Repository Layer:** Maps domain models to/from database, maps DB errors to domain errors.
    - **Flow:** OpenAPI DTO → Handler extracts values → Service calls domain constructor → Repository persists domain model.

## Repositories

- Repository interfaces live in the domain package (`auth.Repository`, `device.Repository`).
- **Implementation Location:** Repository implementations (`repository.go`) are in the same package as their interfaces (e.g., `internal/device/repository.go` alongside `internal/device/`). This is acceptable and common in Go - it keeps related code together and simplifies package structure.
- **SQL Implementation:**
    - Use `sqlx` for database operations.
    - No ORM.
    - Map DB-specific errors (e.g. unique constraint violations) to domain errors **inside** repo methods.
- **Transaction Pattern:** Repositories implement `RunInTx(ctx context.Context, fn func(Repository) error) error` for transactional operations:
    - The callback receives a transactional repository instance.
    - If already in a transaction, the callback reuses it (prevents nested transactions).
    - Transactions auto-rollback on error, commit on success.
    - Services use `RunInTx` to ensure atomicity across multiple repository operations.
    - Example:
      ```go
      err := repo.RunInTx(ctx, func(tx DeviceRepository) error {
          addr, err := tx.CreateAddress(ctx, address)
          if err != nil {
              return err
          }
          return tx.EnableAddress(ctx, addr.ID)
      })
      ```

## Configuration

- Use `internal/config` to load environment variables.
- Pass config values (like `AdminPassword` or `TokenTTL`) into Service constructors (`NewService`) rather than accessing
  global config.

## Testing Strategy

### 1. Domain Layer (Unit Tests)

- **Scope:** Pure functions, Entity constructors (`NewUser`), and Validators (`validateUsername`).
- **Style:** Table-driven tests (`tests := []struct{...}`).
- **Dependencies:** None. These must be ultra-fast and run without external setup.

### 2. Service Layer (Unit Tests)

- **Scope:** Complex business logic, orchestration, and error handling.
- **Dependencies:**
    - Mock the Repository Interface (generate mocks with `mockery` or write simple hand-rolled fakes).
    - Do **not** use real databases here.
- **Style:** Table-driven tests (`tests := []struct{...}`) for related cases.
- **Pattern:** `Given(MockRepo behaves X) -> When(Call Service) -> Then(Expect Y)`.

### 3. Repository Layer (Integration Tests)

- **Scope:** SQL queries, Data mapping, Transaction logic, and Database Constraints (e.g., "Unique Username").
- **Dependencies:** Real SQLite database (in-memory `file::memory:`).
- **Lifecycle:** Use a test helper to spin up a fresh DB instance and run migrations for *each* test (or clear tables).
- **Goal:** Verify that what you save is what you read back, and that errors (`sqlite3.ErrConstraintUnique`) are
  correctly mapped to domain errors.

### 4. HTTP Handler Layer (End-to-End / Component Tests)

- **Scope:** Request binding (JSON -> DTO), Input validation errors, Response marshaling, and Status codes.
- **Dependencies:**
    - Use `httptest.NewServer` or call the handler directly with `httptest.NewRecorder`.
    - Use a **Real Service** backed by a **Real In-Memory DB** (sqlite) whenever possible.
    - Avoid mocking the Service layer unless strictly necessary (mocking obscures serialization bugs).
- **Strict Server Nuance:**
    - Since we use `StrictServer`, ensure tests assert against the *structured response* (e.g., parse the JSON body back
      into the generated Go types) rather than just regex matching the string body.
- **Coverage:** Happy paths + Critical error paths (401, 403, 400). Do not re-test deep business logic edge cases here (
  do that in Service layer).

### 5. Test Design Principles

- **No Logic in Setup:** Test setup (`Given`) should be flat and declarative. No loops or complex `if`s in test
  preparation.
- **Helpers:** Use "Test Builders" or helper functions (e.g., `createAuthenticatedUser(t, db)`) to reduce noise, but
  keep them dumb.
- **Assertions:** Use `stretchr/testify/assert` or `require` for readability.
- **State Isolation:** Every test must run in a clean state. No shared global DB state between tests.
