# ID Types

Every entity identifier is a **typed Go alias** over `int64`, and every response field that carries an entity ID uses the shared `common.yaml#/ID` OpenAPI component. Request body fields that accept a foreign-key ID use a plain `type: integer` (not the `$ref`) to keep the TypeScript form types as `number`.

## Go — typed ID aliases

```go
// Each package defines its own alias.
type UserID   int64   // internal/auth/user.go
type DeviceID int64   // internal/device/device.go
```

Use the alias everywhere in domain structs, service signatures, and repository row structs:

```go
type Device struct {
    ID      DeviceID    `db:"id"`
    OwnerID auth.UserID `db:"owner_id"`
    // ...
}
```

The alias has two helpers: `.Int64()` and `.String()`, used when crossing package boundaries or serialising.

## OpenAPI — schema for entity IDs

### Response / read-only fields → `$ref: 'common.yaml#/ID'`

```yaml
# In PendingRegistration (response schema)
owner_id:
  $ref: 'common.yaml#/ID'
  description: ID of the user who owns this device.
```

`common.yaml#/ID` is `type: integer, format: int64`. The TypeScript generator resolves this to `export type Id = number`, which is safe for JSON transport.

### Request body / writable fields → `type: integer`

When a request body carries a foreign-key ID (e.g., `owner_id` in a create request), **do not use the `$ref`**. Use a plain `type: integer` instead:

```yaml
# In CreateRegistrationRequest (request schema)
owner_id:
  type: integer
  description: ID of the user who will own the registered device.
  example: 1
```

**Why:** `$ref: 'common.yaml#/ID'` generates `zId = z.coerce.bigint()` in the Zod schema. Using `zId` as the type for a form field makes the inferred TypeScript type `bigint`, which is incompatible with `NumberInput` and JSON-serialised payloads. A plain `type: integer` generates `z.int()` → `number`, which is form-bindable.

## Handler — crossing the boundary

```go
// Incoming request (int → typed alias):
OwnerID: auth.UserID(body.OwnerId)

// Outgoing response (typed alias → int64 for the DTO):
OwnerId: device.OwnerID.Int64()
```

For **optional** owner fields (pointer in the generated DTO):
```go
var ownerID *auth.UserID
if body.OwnerId != nil {
    ownerID = new(auth.UserID(*body.OwnerId))
}
```

## Frontend — displaying entity IDs

Response `Id` fields are `number` in TypeScript — display them directly. No special handling needed.

---
**Verified against:** `api/components/schemas/common.yaml`, `api/components/schemas/devices.yaml`, `internal/auth/user.go`, `internal/device/device.go`, `internal/device/handler.go`
**Applies to:** any field that carries an entity primary key or foreign key
**Known gaps:** none
**Last verified:** 2026-04-16
