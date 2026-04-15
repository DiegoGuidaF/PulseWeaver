# Handler Tests (E2E)

Handler tests exercise the full HTTP stack. One test per endpoint for the happy path; extra tests only for handler-specific logic (auth, input validation).

## Setup

```go
package device_test // black-box: _test suffix

func TestHandler_CreateDevice(t *testing.T) {
    // Given — full app with in-memory DB
    srv := testutils.SetupIntegrationServer(t)
    // Seed state via service methods (never via repo or direct DB)

    // When — real HTTP request through the full middleware chain
    body := strings.NewReader(`{"name":"my-device"}`)
    req := httptest.NewRequest(http.MethodPost, "/api/v1/devices", body)
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Cookie", sessionCookie) // from auth setup
    w := httptest.NewRecorder()
    srv.ServeHTTP(w, req)

    // Then — assert on HTTP response
    is := is.New(t)
    is.Equal(w.Code, http.StatusCreated)
    var resp httpapi.DeviceResponse
    json.NewDecoder(w.Body).Decode(&resp)
    is.Equal(resp.Name, "my-device")
}
```

## Key rules

- **`SetupIntegrationServer(t)`** is the only setup entry point — never construct services manually.
- **Seed via services**: call `srv.DeviceService.CreateDevice(...)` etc. in the Given block. Never seed the DB directly.
- **Assert on HTTP**: status code + decoded body. Only reach into service/repo for side effects not visible in the response.
- **Auth paths**: test unauthenticated (no cookie → 401) and forbidden cases in short dedicated tests, not tables.
- **Do not repeat service-level tests**: handler tests confirm the HTTP integration, not business rules.

## Naming

```
TestHandler_<OperationName>          // happy path
TestHandler_<OperationName>_<Case>   // error/edge case
```

---
**Verified against:** `internal/device/handler_test.go`, `internal/testutils/server.go`
**Applies to:** `internal/*/handler_test.go`
**Known gaps:** none
**Last verified:** 2026-04-15
