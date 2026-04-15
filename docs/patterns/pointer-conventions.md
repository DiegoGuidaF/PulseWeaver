# Pointer Conventions (Go 1.26)

## Zero-value pointer allocation

Use `new(T)` instead of `&T{}` when allocating a zero-value struct:

```go
// correct
user := new(User)
if err := r.db.GetContext(ctx, user, query, id); err != nil { ... }
return user, nil

// incorrect — Go 1.26 lint warning
user := &User{}
```

Composite literals with field values stay as-is: `&User{Name: "alice"}` is fine.

## `new(expr)` for pointer fields

When populating a `*T` field with a copy of an existing value:

```go
TTLSeconds: new(config.TTLSeconds)  // *int pointing to a copy — correct
TTLSeconds: &config.TTLSeconds      // *int aliasing the struct field — wrong
```

## When to use pointers

| Context | Use `*T` when | Use `T` when |
|---------|---------------|--------------|
| **Receivers** | Struct has channels, mutexes, connections, or mutable state | Small, purely data structs with no mutable state |
| **Return values** | Filled by DB scanner; value is optional (`nil` = absent) | Constructor/converter; caller would immediately dereference |
| **Parameters** | Function must mutate the argument; struct is large | Small structs (≤~64 bytes) that the function only reads |
| **Struct fields** | Genuinely nullable (`*DenyReason`, `*int` for optional) | Not merely to avoid copying |

---
**Verified against:** `internal/device/repository.go`, `internal/rule/service.go`
**Applies to:** all Go code
**Known gaps:** none
**Last verified:** 2026-04-15
