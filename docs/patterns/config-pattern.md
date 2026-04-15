# Config Pattern

All configuration is loaded from environment variables at startup. No global config access — the config struct is passed into constructors.

## Pattern

```go
// config/config.go
type Conf struct {
    Server ConfServer
    DB     ConfDB
    Rules  ConfRules
    Policy ConfPolicy
}

type ConfServer struct {
    AdminPassword string `env:"ADMIN_PASSWORD,required"`
    ServerPort    int    `env:"SERVER_PORT" envDefault:"8080"`
    TrustedProxy  string `env:"TRUSTED_PROXY"`
    TZ            string `env:"TZ" envDefault:"UTC"`
}
```

## Adding a new env var

1. Add the field to the appropriate `Conf*` struct with `env:"VAR_NAME"` tag
2. Add validation in `Load()` if needed (e.g., minimum length, format check)
3. Pass it through constructors — never read `os.Getenv` directly in services
4. Document in `.env.example`

## Key rules

- **`caarlos0/env/v11`** for parsing — supports `required`, `envDefault`, nested structs.
- **Optional `.env` file** loaded via `godotenv` — useful for local dev.
- **Validation in `Load()`** — e.g., `POLICY_ENGINE_API_SECRET` must be ≥32 chars.
- **Test bypass**: tests construct `*config.Conf` directly — they never call `Load()`.
- **No global state**: the `Conf` struct is threaded through constructors, not stored in a package-level var.

---
**Verified against:** `internal/config/config.go`
**Applies to:** any new configuration option
**Known gaps:** none
**Last verified:** 2026-04-15
