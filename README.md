# WallyDex

**WallyDex** is a self-hosted device IP address management service.
Its primary goal is keeping an updated list of your device's IPs in order to only allow those through your proxy.

__How it works:__
WallyDex acts as a Forward Auth sidecar for your reverse proxy (e.g. Caddy).
On every incoming request the proxy asks WallyDex `GET /api/authz/verify-ip`; WallyDex checks the client IP against its in-memory cache of enabled device IPs and responds `200 OK` or `403 Forbidden`.

# AI usage
This app has not been vibe coded. I am a software developer with 9+ years of experience.
However AI has been used extensively in some parts of the code to speed development as well as to learn from it. This is
my first time doing a project in Go (my usual stack is Java/Kotlin) as well as doing a React frontend.
AI has been used mostly for writing tests and the frontend.

## Features
- **Device Inventory:** Track hardware with metadata.
- **IP Management:** Assign multiple IPs to devices with IPv4/IPv6 validation.
- **Forward Auth:** Acts as an authorization sidecar — your reverse proxy delegates IP allow/deny decisions to WallyDex via `GET /api/authz/verify-ip`, no config reloads needed.
- **Production Grade:** Built with idiomatic Go (Chi, sqlx, SQLite WAL mode).

## Tech Stack
- **Backend:** Go 1.24+, Chi, SQLite, sqlx
- **Frontend:** React, TypeScript, Vite
