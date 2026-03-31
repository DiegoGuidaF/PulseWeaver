# PulseWeaver

[![CI](https://github.com/diegoguidaf/pulseweaver/actions/workflows/ci.yml/badge.svg)](https://github.com/diegoguidaf/pulseweaver/actions/workflows/ci.yml)
[![Docker](https://img.shields.io/badge/docker-ghcr.io%2Fdiegoguidaf%2Fpulseweaver-2496ED?logo=docker&logoColor=white)](https://github.com/diegoguidaf/pulseweaver/pkgs/container/pulseweaver)
[![Go 1.26](https://img.shields.io/badge/go-1.26-00ADD8?logo=go&logoColor=white)](go.mod)
[![License: AGPL v3](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](LICENSE)

**PulseWeaver** is a self-hosted device address tracker and forward-auth gate for reverse proxies.

It keeps an up-to-date registry of your devices' current IP addresses and tells your reverse proxy whether to allow or
block each incoming request — no config file reloads, no static IP lists, and no complex identity providers required for
the services you want to protect.

> [!NOTE]
> PulseWeaver is not an authentication or authorization system. It is an **IP gate**. It does not verify who a user is;
> it only checks whether the IP a request comes from belongs to a registered device. Think of it as a network-layer
> bouncer, not a login system.

---

## Screenshots

| Dashboard | Devices |
|-----------|---------|
| ![Dashboard](screenshots/03-dashboard.png) | ![Devices](screenshots/02-devices.png) |

| Device addresses |
|-----------------|
| ![Device addresses](screenshots/07-device-detail-addresses.png) |

---

## How it works

Two flows work together. Your **reverse proxy** calls `GET /api/policy-engine/verify-ip` on every request — PulseWeaver answers 200 (allow) or 403 (block) from an in-memory cache. Your **devices** send periodic heartbeats (`POST /api/v1/heartbeat` with an `X-API-Key` header) to register their current IP. As long as heartbeats keep coming, the IP stays active. If a device has an address lease configured, the IP expires automatically when the TTL runs out.

📖 [Detailed flow diagrams →](docs/How-It-Works.md)

---

## Why use this?

Many self-hosted services — Home Assistant, Jellyfin, Nextcloud, Grafana, etc. — are not designed to work behind complex
identity proxies like Authelia or authentik. Adding OIDC/SSO to them is often painful and sometimes breaks the app
entirely.

PulseWeaver takes a different approach: **only accept connections from IP addresses you know**. This drastically reduces
the attack surface without touching how the application itself authenticates users.

- No OIDC configuration, no identity provider to maintain.
- Works with any service out of the box.
- Devices with changing IPs (phones, laptops on roaming) stay covered automatically via heartbeat.
- Travel and flexible remote access: heartbeat + address lease gives you zero-config access from wherever your device is.

---

## Key concepts

| Concept           | Description                                                                                                                |
|-------------------|----------------------------------------------------------------------------------------------------------------------------|
| **Device**        | A logical endpoint (phone, laptop, server…) with a unique API key.                                                         |
| **Address**       | An IP address (v4 or v6) linked to a device. Can be enabled or disabled.                                                   |
| **Heartbeat**     | A device call to `/api/v1/heartbeat` that enables the caller's current IP as the device's active address.                  |
| **Address lease** | A TTL* rule per device. When the TTL expires, the address is automatically disabled by PulseWeaver's background scheduler. |
| **Forward Auth**  | The `GET /api/policy-engine/verify-ip` endpoint. Your reverse proxy calls this on every request to check the client IP.    |

> **TTL**: Time-To-Live

---

## Setup

### Docker Compose (recommended)

The easiest way to run PulseWeaver alongside Caddy. The key points are:

- Both services must be on the same Docker network so `pulseweaver:8080` resolves.
- `POLICY_ENGINE_API_SECRET` is defined once in your `.env` and injected into both containers.
- `TRUSTED_PROXY` must be set to Caddy's container IP so PulseWeaver can correctly extract the real
  client IP on both the heartbeat and forward-auth endpoints.
  See [Understanding TRUSTED_PROXY](docs/Understanding-TRUSTED_PROXY.md)
  for the full explanation.

```yaml
# docker-compose.yml
name: proxy

services:
  caddy:
    image: caddy:2.11.1 # Example version, ensure you're running latest
    container_name: caddy
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
      - "443:443/udp"
    environment:
      PULSEWEAVER_POLICY_ENGINE_API_SECRET: ${PULSEWEAVER_POLICY_ENGINE_API_SECRET}
      TZ: ${TZ}
    volumes:
      - ./caddy/Caddyfile:/etc/caddy/Caddyfile
      - ./caddy/data:/data
      - ./caddy/config:/config
    networks:
      proxy:
        ipv4_address: ${CADDY_IP} # Set specific IP so we can wire it to PulseWeaver TRUSTED_PROXY
    depends_on:
      - pulseweaver

  pulseweaver:
    image: ghcr.io/diegoguidaf/pulseweaver:dev
    container_name: pulseweaver
    restart: unless-stopped
    expose: # No need to use "ports" if you access this via Caddy
      - 8080
    environment:
      ADMIN_PASSWORD: ${PULSEWEAVER_ADMIN_PASSWORD}
      SERVER_PORT: 8080
      TRUSTED_PROXY: ${CADDY_IP}      # Caddy's container IP on the shared network (single IP only, no CIDR)
      POLICY_ENGINE_API_SECRET: ${PULSEWEAVER_POLICY_ENGINE_API_SECRET}
      TZ: ${TZ}
    volumes:
      - ./pulseweaver/data:/data   # Bind mount; ensure writable by non-root (chown 65532:65532) or use a named volume
    networks:
      - proxy

networks:
  proxy:
    driver: bridge
    ipam:
      config:
        - subnet: 172.20.0.0/24
```

A minimal `.env` alongside it:

```dotenv
PULSEWEAVER_POLICY_ENGINE_API_SECRET=a-very-long-random-secret-at-least-32-chars
PULSEWEAVER_ADMIN_PASSWORD=a-strong-admin-password
CADDY_IP=172.20.0.2   # Caddy's fixed IP on the proxy network (single IP, no CIDR)
TZ=Europe/Madrid
```

> [!TIP]
> Give Caddy a fixed `ipv4_address` on the shared docker network and set `TRUSTED_PROXY` to that exact IP.
> `TRUSTED_PROXY` accepts a **single IP address only** — CIDR ranges are not supported. Pinning Caddy's IP is the
> simplest way to keep this stable.

### First-run admin account

On first startup, PulseWeaver creates an `admin` user using the `ADMIN_PASSWORD` environment variable. Use a strong,
unique password and store it securely (e.g. in your `.env` file with restricted permissions).

### Configuration reference

| Variable              | Required           | Default | Description                                                                                                                                                                                                  |
|-----------------------|--------------------|---------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `ADMIN_PASSWORD`      | Yes                | —       | Password for the `admin` UI account (bootstrapped on first run).                                                                                                                                             |
| `POLICY_ENGINE_API_SECRET`    | Yes (min 32 chars) | —       | Shared secret between Caddy and PulseWeaver. Minimum 32 characters.                                                                                                                                          |
| `SERVER_PORT`         | No                 | `8080`  | Port PulseWeaver listens on.                                                                                                                                                                                 |
| `TRUSTED_PROXY`       | No                 | —       | Single IP address of your reverse proxy. Required when running behind a proxy — see [Understanding TRUSTED_PROXY](docs/Understanding-TRUSTED_PROXY.md).                                                     |
| `RULE_CHECK_INTERVAL` | No                 | `1m`    | How often the scheduler checks for expired address leases. Set this to the lowest address lease TTL you'll use.                                                                                              |
| `DB_DIR`              | No                 | `./data` (Docker: `/data`) | Directory for the SQLite database. See [Data Persistence](docs/Data-Persistence.md).                                                                                                                        |
| `TZ`                  | No                 | `UTC`   | Application timezone for explicit wall-clock operations. Persisted timestamps are UTC; API timestamps are serialized as UTC RFC3339.                                                                         |
| `LOG_LEVEL`           | No                 | `info`  | Log level: `debug`, `info`, `warn`, `error`.                                                                                                                                                                 |
| `LOG_FORMAT`          | No                 | `text`  | Log format: `text` (human-readable) or `json`.                                                                                                                                                               |
| `LOG_COLOR`           | No                 | `true`  | Use coloured output for `text` format.                                                                                                                                                                       |

---

## Proxy integration

### Caddy (forward_auth)

Add the `forward_auth` block to any site you want to protect:

```caddy
your-service.example.com {
    forward_auth http://pulseweaver:8080 {
        uri /api/policy-engine/verify-ip
        header_up X-Real-IP {http.request.remote.host}
        header_up Authorization "Bearer {$PULSEWEAVER_POLICY_ENGINE_API_SECRET}"
    }

    reverse_proxy your-service:port
}
```

PulseWeaver's verify-ip endpoint is **fail-closed**: any missing header, invalid secret, or inactive IP returns `403`.

### Other reverse proxies

Any proxy that supports forward auth can work. The requirements are:

1. Call `GET http://pulseweaver:8080/api/policy-engine/verify-ip` before forwarding the request.
2. Pass the real client IP in the `X-Real-IP` header.
3. Pass `Authorization: Bearer <POLICY_ENGINE_API_SECRET>` to authenticate the proxy-to-PulseWeaver call.
4. Allow the request through on `200`; block on anything else.

---

## Keeping devices connected

Devices send periodic heartbeats to keep their current IP active. There are several ways to set this up:

| Method | Best for | Details |
|--------|----------|---------|
| **[Heartbeat Client](https://github.com/DiegoGuidaF/pulseweaver-heartbeat-client)** | Android, desktop (Linux/macOS/Windows) | Dedicated app with background scheduling, network-awareness, and system tray support. |
| **[systemd timer / launchd agent](docs/Lightweight-Heartbeat-Clients.md)** | Headless Linux & macOS servers | Zero-dependency `curl` + OS scheduler. No app needed. |
| **[Tasker](docs/Heartbeat-Endpoint-Setup.md#android-tasker)** | Android (DIY) | HTTP request on a timer or network change event. |
| **Manual** | Static IP devices | Add addresses directly in the PulseWeaver UI — no heartbeat needed. |

The heartbeat endpoint (`POST /api/v1/heartbeat`) must be exposed **without** the forward-auth gate — see
[Heartbeat Endpoint Setup](docs/Heartbeat-Endpoint-Setup.md) for the Caddy configuration.

---

## Security model

PulseWeaver is an **IP gate** — it reduces your attack surface by blocking unknown IPs before they reach any service.
It is **not** a user authentication system or a replacement for identity providers.

| ✅ Works well | ⚠️ Not enough on its own |
|--------------|--------------------------|
| Services that break behind SSO proxies | Identifying individual users |
| Homelab with a small set of trusted networks | Clients on ISP-level CGNAT |
| Reducing blast radius of unpatched CVEs | Compromised active networks |
| Zero-config travel access via heartbeat + lease | Replacing TLS or app-level auth |

PulseWeaver should be **one layer** in a defence-in-depth strategy, not the only one.

📖 **Deep dives:** [Security Model](docs/Security-Model.md) · [Shared-IP Model](docs/Shared-IP-Model.md) · [Understanding TRUSTED_PROXY](docs/Understanding-TRUSTED_PROXY.md)

---

## Development

PulseWeaver compiles to a **single binary** with the frontend SPA embedded. The frontend is built with Vite and
embedded at compile time — no separate web server needed in production.

### Quick start for local development

For local runs (no Docker), the app writes to `/data/data.db`. Create that directory and make it writable before
starting: `sudo mkdir -p /data && sudo chown $(whoami) /data`.

```bash
# Backend (hot reload via Air)
make dev-back

# Frontend (Vite dev server, in a separate terminal)
make dev-front
```

### Useful make targets

| Command               | Description                                                 |
|-----------------------|-------------------------------------------------------------|
| `make build`          | Full production build → `bin/pulseweaver`                      |
| `make test`           | Run all Go tests                                            |
| `make lint`           | Format + lint                                               |
| `make api`            | Regenerate backend + frontend types from `api/openapi.yaml` |
| `make migrate-up`     | Apply pending database migrations                           |
| `make migrate-create` | Create a new migration pair                                 |

### Further reading

- [`CODEBASE-Backend.md`](CODEBASE-Backend.md) — backend package structure, domain boundaries, service lifecycle,
  observer pattern.
- [`CODEBASE-Frontend.md`](CODEBASE-Frontend.md) — frontend directory structure, routing, hook conventions, UX surfaces.
- [`CLAUDE.md`](CLAUDE.md) — full reference for AI-assisted development, conventions, and testing patterns.
- [`api/openapi.yaml`](api/openapi.yaml) — API schema; single source of truth for all endpoints and types.

### A note on AI usage

This project has not been vibe-coded. The author is a software developer with 9+ years of experience (primary stack:
Java/Kotlin). This is a first Go project and first React frontend. AI has been used extensively to accelerate tests and
frontend work, and as a learning tool — not as a replacement for understanding the code.
