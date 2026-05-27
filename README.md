<picture>
  <source media="(prefers-color-scheme: dark)" srcset=".github/assets/wordmark-dark.svg">
  <img src=".github/assets/wordmark-light.svg" alt="PulseWeaver" height="48">
</picture>

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

| Dashboard                                  | Devices                                |
|--------------------------------------------|----------------------------------------|
| ![Dashboard](screenshots/03-dashboard.png) | ![Devices](screenshots/02-devices.png) |

| Device addresses                                                |
|-----------------------------------------------------------------|
| ![Device addresses](screenshots/07-device-detail-addresses.png) |

---

## How it works

Two flows work together. Your **reverse proxy** calls `GET /api/policy-engine/verify-ip` on every request — PulseWeaver
answers 200 (allow) or 403 (block) from an in-memory cache. Your **devices** send periodic heartbeats (
`POST /api/v1/heartbeat` with an `X-API-Key` header) to register their current IP. As long as heartbeats keep coming,
the IP stays active. If a device has an address lease configured, the IP expires automatically when the TTL runs out.

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
- Travel and flexible remote access: heartbeat + address lease gives you zero-config access from wherever your device
  is.

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

The easiest way to run PulseWeaver alongside Caddy. Three values must stay in sync:

- **`ipv4_address`** pins Caddy to a fixed IP on the shared Docker network.
- **`CADDY_IP`** in `.env` holds that same address.
- **`TRUSTED_PROXY`** in PulseWeaver's environment is set to `${CADDY_IP}`.

Together they tell PulseWeaver which connection peer is the trusted proxy, so it reads the real
client IP from forwarded requests rather than treating Caddy's own IP as the source. See
[Understanding TRUSTED_PROXY](docs/Understanding-TRUSTED_PROXY.md) for the full explanation.
`POLICY_ENGINE_API_SECRET` is defined once in `.env` and injected into both containers.

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
          gateway: 172.20.0.1
          ip_range: 172.20.0.128/25  # Restrict auto-assigned IPs to upper half (.128–.254)
```

A minimal `.env` alongside it:

```dotenv
PULSEWEAVER_POLICY_ENGINE_API_SECRET=a-very-long-random-secret-at-least-32-chars
PULSEWEAVER_ADMIN_PASSWORD=a-strong-admin-password
CADDY_IP=172.20.0.2   # Fixed IP for Caddy — must be in the lower half of the subnet (see note below)
TZ=Europe/Madrid
```

> [!TIP]
> Generate strong values for the two secrets with OpenSSL:
> ```bash
> openssl rand -base64 32   # PULSEWEAVER_POLICY_ENGINE_API_SECRET
> openssl rand -base64 24   # PULSEWEAVER_ADMIN_PASSWORD
> ```

> [!NOTE]
> **Choosing `CADDY_IP`:** Pick an IP in the lower half of your subnet (e.g. `172.20.0.2`). The
> `ip_range` entry in the IPAM config above restricts Docker's auto-assigned IPs to the upper half
> (`172.20.0.128/25`), so no container joining the network without a fixed IP can accidentally
> receive Caddy's address and silently become a trusted proxy.
>
> **`TRUSTED_PROXY` accepts a single IP only** — CIDR ranges are not supported. Pinning Caddy's IP
> with `ipv4_address` is the simplest way to keep it stable across container restarts.

### First-run admin account

On first startup, PulseWeaver creates an `admin` user using the `ADMIN_PASSWORD` environment variable. Use a strong,
unique password and store it securely (e.g. in your `.env` file with restricted permissions).

### Configuration reference

| Variable                   | Required           | Default                    | Description                                                                                                                                             |
|----------------------------|--------------------|----------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------|
| `ADMIN_PASSWORD`           | Yes                | —                          | Password for the `admin` UI account (bootstrapped on first run).                                                                                        |
| `POLICY_ENGINE_API_SECRET` | Yes (min 32 chars) | —                          | Shared secret between Caddy and PulseWeaver. Minimum 32 characters.                                                                                     |
| `SERVER_PORT`              | No                 | `8080`                     | Port PulseWeaver listens on.                                                                                                                            |
| `TRUSTED_PROXY`            | No                 | —                          | Single IP address of your reverse proxy. Required when running behind a proxy — see [Understanding TRUSTED_PROXY](docs/Understanding-TRUSTED_PROXY.md). |
| `RULE_CHECK_INTERVAL`      | No                 | `1m`                       | How often the scheduler checks for expired address leases. Set this to the lowest address lease TTL you'll use.                                         |
| `DB_DIR`                   | No                 | `./data` (Docker: `/data`) | Directory for the SQLite database. See [Data Persistence](docs/Data-Persistence.md).                                                                    |
| `TZ`                       | No                 | `UTC`                      | Application timezone for explicit wall-clock operations. Persisted timestamps are UTC; API timestamps are serialized as UTC RFC3339.                    |
| `LOG_LEVEL`                | No                 | `info`                     | Log level: `debug`, `info`, `warn`, `error`.                                                                                                            |
| `LOG_FORMAT`               | No                 | `text`                     | Log format: `text` (human-readable) or `json`.                                                                                                          |
| `LOG_COLOR`                | No                 | `true`                     | Use coloured output for `text` format.                                                                                                                  |

---

## Proxy integration

PulseWeaver works with any reverse proxy that supports forward auth — the integration requirements
are the same regardless of which proxy you use. Currently only Caddy has a tested and validated
configuration. If you have a working setup with nginx, Traefik, or another proxy,
[open an issue or PR](https://github.com/diegoguidaf/pulseweaver/issues) and community-validated
configurations will be added to the documentation.

### Caddy

Add the `forward_auth` block to any site you want to protect:

```caddy
your-service.example.com {
    forward_auth pulseweaver:8080 {
        uri /api/policy-engine/verify-ip
        header_up X-Real-IP {http.request.remote.host}
        header_up Authorization "Bearer {$PULSEWEAVER_POLICY_ENGINE_API_SECRET}"
    }
    reverse_proxy your-service:port
}
```

PulseWeaver's verify-ip endpoint is **fail-closed**: missing header, wrong secret, or unregistered IP → `403`.

📖 [Full Caddy setup guide →](docs/Caddy-Setup.md) — device endpoints, admin UI configuration,
troubleshooting, and other proxy support.

---

## Keeping devices connected

Devices send periodic heartbeats to keep their current IP active. There are several ways to set this up:

| Method                                                                              | Best for                               | Details                                                                               |
|-------------------------------------------------------------------------------------|----------------------------------------|---------------------------------------------------------------------------------------|
| **[Heartbeat Client](https://github.com/DiegoGuidaF/pulseweaver-heartbeat-client)** | Android, desktop (Linux/macOS/Windows) | Dedicated app with background scheduling, network-awareness, and system tray support. |
| **[systemd timer / launchd agent](docs/Lightweight-Heartbeat-Clients.md)**          | Headless Linux & macOS servers         | Zero-dependency `curl` + OS scheduler. No app needed.                                 |
| **[Tasker](docs/Heartbeat-Endpoint-Setup.md#android-tasker)**                       | Android (DIY)                          | HTTP request on a timer or network change event.                                      |
| **Manual**                                                                          | Static IP devices                      | Add addresses directly in the PulseWeaver UI — no heartbeat needed.                   |

The heartbeat (`POST /api/v1/heartbeat`) and device pairing (`POST /api/v1/device-pairing`) endpoints
must be exposed **without** the forward-auth gate — see [Caddy setup guide](docs/Caddy-Setup.md) or
[Heartbeat Endpoint Setup](docs/Heartbeat-Endpoint-Setup.md) for configuration details.

---

## Device provisioning

If using the heartbeat-client, instead of manually entering a server URL and API key, an admin can generate a *
*registration code** that fully
configures the heartbeat client in a single paste (or QR scan on mobile).

### How it works

1. The admin opens **Devices → Provisioning** in the PulseWeaver UI and creates an invite — choosing a device name,
   owner, heartbeat interval, biometric settings, and an expiry window.
2. PulseWeaver generates a single-use registration code (an opaque base64 string that encodes the server URL and a
   random token). The admin shares the code as a QR (generated client-side in the browser) or copyable text.
3. The user pastes the code in the
   [Heartbeat Client](https://github.com/DiegoGuidaF/pulseweaver-heartbeat-client). The app decodes the server URL from
   the code, calls the server's claim endpoint, and receives the full device configuration + API key.
4. The device is created, the code is invalidated, and the app starts sending heartbeats — no manual setup required.

The code is retrievable by the admin until it is claimed or expires. After claim, both the code and the plaintext API
key are deleted from the database; an audit trail (timestamp, device link, key prefix) is retained.

> For details on how the heartbeat client handles provisioning, see the
> [Heartbeat Client documentation](https://github.com/DiegoGuidaF/pulseweaver-heartbeat-client/blob/main/docs/app.md#device-provisioning).

### Public endpoints

Device provisioning adds a second endpoint that must be reachable from devices without the forward-auth gate:

| Endpoint                        | Purpose                    | Auth                    |
|---------------------------------|----------------------------|-------------------------|
| `POST /api/v1/heartbeat`        | Ongoing heartbeats         | `X-API-Key` header      |
| `POST /api/v1/device-pairing`   | One-time device pairing    | Pairing code in body    |

📖 [Caddy setup guide →](docs/Caddy-Setup.md) — covers the recommended two-domain configuration that
exposes only these endpoints publicly while keeping the admin UI off the internet.

> [!TIP]
> Set the **Heartbeat server URL** in the device invite to `https://pw-device.example.com`. The
> heartbeat client uses that URL for both the initial pairing and all subsequent heartbeats.

---

## Security model

PulseWeaver is an **IP gate** — it reduces your attack surface by blocking unknown IPs before they reach any service.
It is **not** a user authentication system or a replacement for identity providers.

| ✅ Works well                                    | ⚠️ Not enough on its own        |
|-------------------------------------------------|---------------------------------|
| Services that break behind SSO proxies          | Identifying individual users    |
| Homelab with a small set of trusted networks    | Clients on ISP-level CGNAT      |
| Reducing blast radius of unpatched CVEs         | Compromised active networks     |
| Zero-config travel access via heartbeat + lease | Replacing TLS or app-level auth |

PulseWeaver should be **one layer** in a defence-in-depth strategy, not the only one.

📖 **Deep dives:
** [Security Model](docs/Security-Model.md) · [Shared-IP Model](docs/Shared-IP-Model.md) · [Understanding TRUSTED_PROXY](docs/Understanding-TRUSTED_PROXY.md)

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
| `make build`          | Full production build → `bin/pulseweaver`                   |
| `make test`           | Run all Go tests                                            |
| `make lint-back`      | Format + lint                                               |
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
