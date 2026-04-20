# How to: Add a cross-domain read query

Use the `queries` package when a read needs data from **more than one domain** (e.g. joining `known_hosts` with `access_log`, or aggregating across `devices` and `addresses`). Single-domain reads stay in their own repository.

## When to use `queries` vs a domain repository

| Situation | Where it goes |
|-----------|--------------|
| Read from one table / one domain | `internal/<domain>/repository.go` |
| Read joins two or more domain tables | `internal/queries/` |
| Aggregation for the dashboard | `internal/dashboard/repository.go` (already has its own cross-domain queries) |

## Files you will touch

| Step | File |
|------|------|
| 1 | `internal/queries/<view-name>.go` — new result type + query method on `Repository` |
| 2 | `internal/queries/handler.go` — new handler method |
| 3 | `api/openapi.yaml` + path/schema files — expose via HTTP (optional, follow `add-http-endpoint.md`) |

## Step 1 — Add the query to the queries repository

Create a new file (or add to an existing one) in `internal/queries/`:

```go
// internal/queries/my_view.go
package queries

import (
    "context"
    "fmt"
    "time"

    "github.com/DiegoGuidaF/PulseWeaver/internal/somedomain"
)

type MyViewRow struct {
    ID        somedomain.ThingID `db:"id"`
    Name      string             `db:"name"`
    LastSeen  *time.Time         `db:"last_seen"`   // nullable JOIN result
    HitCount  int                `db:"hit_count"`
}

func (r *Repository) GetMyView(ctx context.Context) ([]MyViewRow, error) {
    const q = `
        SELECT t.id, t.name,
               MAX(al.created_at) AS last_seen,
               COUNT(al.id)       AS hit_count
        FROM   things t
        LEFT JOIN access_log al ON LOWER(al.target_host) = LOWER(t.fqdn)
        GROUP BY t.id, t.name
        ORDER BY t.name
    `
    rows, err := r.db.QueryContext(ctx, q)
    if err != nil {
        return nil, fmt.Errorf("get my view: %w", err)
    }
    defer rows.Close()

    var out []MyViewRow
    for rows.Next() {
        var row MyViewRow
        if err := rows.Scan(&row.ID, &row.Name, &row.LastSeen, &row.HitCount); err != nil {
            return nil, fmt.Errorf("scan my view row: %w", err)
        }
        out = append(out, row)
    }
    return out, rows.Err()
}
```

Key rules:
- **Only reads** — the `queries` package never writes. Mutations stay in domain repositories.
- Import domain ID types (e.g. `somedomain.ThingID`) so the result is typed. Never use raw `int64` IDs in query results.
- Use `LOWER()` when joining against `access_log.target_host` — it is stored as-received, not normalized.
- `queries.Repository` is already wired in `app.go` — no wiring change needed for new methods on it.

## Step 2 — Add a handler method

```go
// internal/queries/handler.go
func (h *HTTPHandler) ListMyView(
    ctx context.Context,
    req httpapi.ListMyViewRequestObject,
) (httpapi.ListMyViewResponseObject, error) {
    ctx = logging.WithOperation(ctx, "ListMyView")

    rows, err := h.repo.GetMyView(ctx)
    if err != nil {
        h.logger.ErrorContext(ctx, "list my view failed", slog.Any(logging.AttrKeyError, err))
        return httpapi.ListMyView500JSONResponse(errResp("Failed to load view")), nil
    }

    resp := make([]httpapi.MyViewRow, len(rows))
    for i, r := range rows {
        resp[i] = toMyViewRowDTO(r)
    }
    return httpapi.ListMyView200JSONResponse(resp), nil
}
```

Because `queries.HTTPHandler` already implements `StrictServerInterface` methods, new methods on it are picked up automatically — no changes to `routes.go`, `server.go`, or `app.go` are needed.

## Step 3 — Expose via HTTP (if needed)

Follow `add-http-endpoint.md`. The only difference is that the new operation goes on `queries.HTTPHandler`, not a domain handler.

---
**Related patterns:** `repository-layer.md`, `schema-first-api.md`
**Real examples:** `internal/queries/host_access_view.go` (KnownHostStats, HostSuggestion queries), `internal/dashboard/repository.go` (traffic aggregation)
**Last verified:** 2026-04-20
