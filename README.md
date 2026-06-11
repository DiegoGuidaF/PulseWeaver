<picture>
  <source media="(prefers-color-scheme: dark)" srcset=".github/assets/wordmark-dark.svg">
  <img src=".github/assets/wordmark-light.svg" alt="PulseWeaver" height="48">
</picture>

[![CI](https://github.com/diegoguidaf/pulseweaver/actions/workflows/ci.yml/badge.svg)](https://github.com/diegoguidaf/pulseweaver/actions/workflows/ci.yml)
[![Docker](https://img.shields.io/badge/docker-ghcr.io%2Fdiegoguidaf%2Fpulseweaver-2496ED?logo=docker&logoColor=white)](https://github.com/diegoguidaf/pulseweaver/pkgs/container/pulseweaver)
[![Go 1.26](https://img.shields.io/badge/go-1.26-00ADD8?logo=go&logoColor=white)](go.mod)
[![License: AGPL v3](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](LICENSE)

**PulseWeaver** is a self-hosted forward-auth gate for reverse proxies — per-user, IP-based access control over which devices reach which services.

It keeps an up-to-date registry of your devices' current IP addresses via heartbeats, and answers one question for your
reverse proxy on every incoming request: **may this client reach this host?** Each user gets an explicit allowlist of
services; everything else is denied. No config file reloads, no static IP lists, and no identity provider bolted onto
apps that can't handle one.

It exists for the services that break behind SSO proxies — Home Assistant, Jellyfin, Nextcloud, IoT dashboards.
Instead of changing how an application authenticates, PulseWeaver simply keeps everyone except known devices of
permitted users from reaching it at all. The whole thing is **one binary** with the web UI embedded and a single
SQLite file — no database server, no separate frontend to deploy.

> [!NOTE]
> PulseWeaver is not an authentication system. It is an **IP gate with per-user host authorization**: it never
> verifies *who* sends a request — it checks whether the request's IP belongs to a registered device (or trusted
> network range) and whether that device's owner is allowed to reach the requested host. Think of it as a
> network-layer bouncer with a guest list per door, not a login system.

---

## Features

- **Forward-auth gate** — your reverse proxy asks PulseWeaver on every request; answered from an in-memory cache, no
  per-request database work.
- **Heartbeat-tracked device IPs** — phones and laptops keep their changing addresses registered automatically;
  address leases expire devices that go quiet.
- **Per-user host access control** — deny-by-default allowlists over an admin-curated set of known hosts, organised
  into groups: "Mom can watch Jellyfin" is one checkbox. ([docs](docs/Host-Access-Control.md))
- **Network policies** — CIDR-range grants for networks you trust as a whole, like your LAN or a VPN subnet.
  ([docs](docs/Network-Policies.md))
- **Access logs & analytics** — every allow/deny decision recorded and filterable; dashboard with traffic over time,
  per-service splits, top denied IPs, and GeoIP enrichment. ([docs](docs/Observability.md))
- **Suggested hosts** — PulseWeaver proposes hostnames it sees in real traffic, so building the known-hosts list takes
  minutes, not an audit.
- **QR device provisioning** — one registration code (or QR scan) configures a device end-to-end.
- **Simulate tool** — ask "would IP X reach host Y?" and see exactly why, without sending real traffic.
- **Single binary** — embedded web UI, SQLite storage; one container, one volume, done.

---

## Screenshots

| Dashboard                                  | Host access control                                              |
|--------------------------------------------|-------------------------------------------------------------------|
| ![Dashboard](screenshots/03-dashboard.png) | ![Host access control](screenshots/09-host-access-control.png)   |

| Devices                                | Device addresses                                                |
|----------------------------------------|------------------------------------------------------------------|
| ![Devices](screenshots/02-devices.png) | ![Device addresses](screenshots/07-device-detail-addresses.png) |

---

## How it works

Two flows work together. Your **reverse proxy** calls `GET /api/policy-engine/verify-ip` on every request, asking
*"may the client at this IP reach this host?"* — PulseWeaver answers 200 (allow) or 403 (deny) from an in-memory
cache. A request is allowed through one of two grants: the IP is an active address of a registered device whose user
is allowed that host, or the IP falls inside a [network policy](docs/Network-Policies.md) range that allows it.
Everything else — including known devices asking for hosts their user was never granted — is denied.

Your **devices** send periodic heartbeats (`POST /api/v1/heartbeat` with an `X-API-Key` header) to keep their current
IP registered. As long as heartbeats keep coming, the address stays active; with an address lease configured, it
expires automatically when the TTL runs out.

📖 [Detailed flow diagrams →](docs/How-It-Works.md)

---

## Key concepts

| Concept            | Description                                                                                                                |
|--------------------|------------------------------------------------------------------------------------------------------------------------------|
| **User**           | A person. Devices belong to users, and access is granted to users — "admin" is a role on a user, not a separate account type. |
| **Device**         | A logical endpoint (phone, laptop, server…) with a unique API key, owned by a user.                                       |
| **Address**        | An IP address (v4 or v6) linked to a device. Can be enabled or disabled.                                                  |
| **Heartbeat**      | A device call to `/api/v1/heartbeat` that enables the caller's current IP as the device's active address.                 |
| **Address lease**  | A TTL* rule per device. When the TTL expires, the address is automatically disabled by PulseWeaver's background scheduler. |
| **Known host**     | An admin-curated hostname that can be granted to users, e.g. `jellyfin.example.org`.                                      |
| **Host group**     | A named bundle of known hosts ("media", "storage") — the unit in which access is granted.                                 |
| **Network policy** | A CIDR-range grant for clients that are not registered devices — e.g. "the whole home LAN may reach these hosts."         |
| **Forward Auth**   | The `GET /api/policy-engine/verify-ip` endpoint. Your reverse proxy calls this on every request.                          |

> **TTL**: Time-To-Live

---

## Quick start

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
CADDY_IP=172.20.0.2   # Fixed IP for Caddy — keep it in the lower half of the subnet
TZ=Europe/Madrid
```

> [!TIP]
> Generate strong values for the two secrets with OpenSSL:
> ```bash
> openssl rand -base64 32   # PULSEWEAVER_POLICY_ENGINE_API_SECRET
> openssl rand -base64 24   # PULSEWEAVER_ADMIN_PASSWORD
> ```

> [!NOTE]
> `TRUSTED_PROXY` takes a single IP, not a CIDR range, and the compose file above reserves the lower half of the
> subnet so nothing can accidentally claim Caddy's address. Why both of these matter:
> [Understanding TRUSTED_PROXY](docs/Understanding-TRUSTED_PROXY.md#choosing-the-proxy-ip-in-docker-compose).

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
| `DATA_RETENTION_DAYS`      | No                 | `30`                       | Days to keep access-log and address-history entries; `0` disables pruning. See [Observability](docs/Observability.md).                                 |
| `GEOIP_ENABLED`            | No                 | `true`                     | Resolve client IPs to country/ASN for logs and dashboard. See [Observability](docs/Observability.md).                                                  |
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

PulseWeaver's verify-ip endpoint is **fail-closed**: missing header, wrong secret, unregistered IP, or a host the
user was never granted → the same `403`. Remember that access is deny-by-default — after adding the block, add
`your-service.example.com` as a known host and grant it to the users who should reach it
([Host Access Control](docs/Host-Access-Control.md)).

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

If using the heartbeat-client, instead of manually entering a server URL and API key, an admin can generate a
**registration code** that fully configures the heartbeat client in a single paste (or QR scan on mobile).

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

PulseWeaver is an **IP gate with per-user host authorization** — it blocks unknown IPs before they reach any service,
and decides per user which hosts the known ones may reach. It is **not** a user authentication system or a replacement
for identity providers: who is behind a request is inferred from its IP, never verified. When several users share one
IP, only hosts that **all** of them may reach are allowed — the strictest grant wins.

| ✅ Works well                                    | ⚠️ Not enough on its own                       |
|-------------------------------------------------|------------------------------------------------|
| Services that break behind SSO proxies          | Verifying who a user is (identity is IP-inferred) |
| Homelab with a small set of trusted networks    | Clients on ISP-level CGNAT                     |
| Reducing blast radius of unpatched CVEs         | Compromised active networks                    |
| Zero-config travel access via heartbeat + lease | Replacing TLS or app-level auth                |

PulseWeaver should be **one layer** in a defence-in-depth strategy, not the only one.

📖 **Deep dives:
** [Security Model](docs/Security-Model.md) · [Shared-IP Model](docs/Shared-IP-Model.md) · [Understanding TRUSTED_PROXY](docs/Understanding-TRUSTED_PROXY.md)

---

## Project status & support

PulseWeaver is in **beta**. The core gate, host access control, and observability surfaces are stable and in daily
use, but expect occasional rough edges and breaking changes before a 1.0 release.

- 🐛 **Bug reports** → [GitHub Issues](https://github.com/diegoguidaf/pulseweaver/issues)
- 💬 **Questions & ideas** → [GitHub Discussions](https://github.com/diegoguidaf/pulseweaver/discussions)
- 🔀 **Working nginx / Traefik config?** Contributions are very welcome — see [Proxy integration](#proxy-integration).

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
