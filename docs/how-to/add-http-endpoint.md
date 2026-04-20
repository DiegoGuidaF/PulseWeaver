# How to: Add an HTTP endpoint end-to-end

This walks every file you must touch to ship a new API endpoint. The files must be changed in this order because each step depends on the previous one.

## 1. Define the endpoint in OpenAPI

`api/openapi.yaml` is the single source of truth. Never edit `internal/httpapi/` directly — it is generated.

**a. Add the schema** (if the endpoint returns or accepts a new type):

```yaml
# api/components/schemas/my-domain.yaml
MyThing:
  type: object
  required: [id, name]
  properties:
    id:
      $ref: './common.yaml#/ID'   # always use $ref for ID fields
    name:
      type: string
```

**b. Add the path** in a path file or inline in `openapi.yaml`:

```yaml
# api/paths/my-domain.yaml
listMyThings:
  get:
    operationId: ListMyThings
    summary: List things
    tags: [my-domain]
    responses:
      '200':
        content:
          application/json:
            schema:
              type: array
              items:
                $ref: '../components/schemas/my-domain.yaml#/MyThing'
```

**c. Register** both in `api/openapi.yaml`:

```yaml
paths:
  /api/v1/admin/my-things:
    $ref: './paths/my-domain.yaml#/listMyThings'

components:
  schemas:
    MyThing:
      $ref: './components/schemas/my-domain.yaml#/MyThing'
```

**d. Regenerate:**

```sh
make api   # runs GOWORK=off oapi-codegen + frontend codegen
```

After this, `internal/httpapi/server.gen.go` and `frontend/src/lib/api/` are updated.

## 2. Implement the handler method

The generated interface now includes your operation. Implement it in `internal/<domain>/handler.go`:

```go
func (h *HTTPHandler) ListMyThings(
    ctx context.Context,
    req httpapi.ListMyThingsRequestObject,
) (httpapi.ListMyThingsResponseObject, error) {
    ctx = logging.WithOperation(ctx, "ListMyThings")

    things, err := h.service.ListMyThings(ctx)
    if err != nil {
        h.logger.ErrorContext(ctx, "list my things failed", slog.Any(logging.AttrKeyError, err))
        return httpapi.ListMyThings500JSONResponse(errResp("Failed to list things")), nil
    }

    resp := make([]httpapi.MyThing, len(things))
    for i, t := range things {
        resp[i] = toMyThingDTO(t)
    }
    return httpapi.ListMyThings200JSONResponse(resp), nil
}
```

Rules:
- `logging.WithOperation` at entry
- Never pass `httpapi` types deeper than the handler
- Map domain sentinels to HTTP codes; unhandled errors → 500 (log first)
- `ErrBadRequest` (from Params constructors) → 400; see service-layer.md

## 3. (If new domain) Register the handler in routes and server

If the handler is on an **existing** `*HTTPHandler` that is already part of `CompositeHandler`, nothing to do here — oapi-codegen's strict interface compilation will catch missing methods.

If it is a **new domain handler**, add it in three files:

**`internal/httpserver/routes.go`:**

```go
type MyDomainHandler = mydomain.HTTPHandler

// Add to CompositeHandler
type CompositeHandler struct {
    ...
    *MyDomainHandler
}

// Add to addRoutes signature and body
func addRoutes(r *chi.Mux, ..., myDomainHandler *MyDomainHandler, ...) {
    routeHandler := &CompositeHandler{
        ...
        MyDomainHandler: myDomainHandler,
    }
```

**`internal/httpserver/server.go`:**

```go
func NewServer(..., myDomainHandler *MyDomainHandler, ...) http.Handler {
    ...
    addRoutes(r, ..., myDomainHandler, ...)
}
```

**`internal/app/app.go`:**

```go
myDomainRepo    := mydomain.NewRepository(db.DB())
myDomainService := mydomain.NewService(myDomainRepo, logger)
myDomainHandler := mydomain.NewHTTPHandler(myDomainService, logger)
...
handler := httpserver.NewServer(..., myDomainHandler, ...)
```

## 4. Verify

```sh
make check   # lint-all + all tests
```

The compiler will fail at `CompositeHandler` construction if any `StrictServerInterface` method is missing — this is the intended safety net.

---
**Related patterns:** `handler-structure.md`, `service-layer.md`, `schema-first-api.md`
**Real example:** `hostaccess/handler.go` wired in `httpserver/routes.go` and `app/app.go`
**Last verified:** 2026-04-20
