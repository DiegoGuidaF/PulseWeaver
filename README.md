<picture>
  <source media="(prefers-color-scheme: dark)" srcset=".github/assets/wordmark-dark.svg">
  <img src=".github/assets/wordmark-light.svg" alt="PulseWeaver" height="48">
</picture>

[![CI](https://github.com/diegoguidaf/pulseweaver/actions/workflows/ci.yml/badge.svg)](https://github.com/diegoguidaf/pulseweaver/actions/workflows/ci.yml)
[![Docker](https://img.shields.io/badge/docker-ghcr.io%2Fdiegoguidaf%2Fpulseweaver-2496ED?logo=docker&logoColor=white)](https://github.com/diegoguidaf/pulseweaver/pkgs/container/pulseweaver)
[![Go 1.26](https://img.shields.io/badge/go-1.26-00ADD8?logo=go&logoColor=white)](go.mod)
[![License: AGPL v3](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](LICENSE)

**PulseWeaver** is a self-hosted closed-door access layer for reverse proxies: unknown IPs are stopped before they reach
your apps, while known devices get per-user access to only the services you grant.

It keeps an up-to-date registry of your devices' current IP addresses and answers one question for your reverse proxy
on every incoming request: **may this client reach this host?** Each user gets an explicit allowlist of services;
everything else is denied. No config-file reloads, no static IP lists, and no identity provider bolted onto apps that
can't handle one.

> [!NOTE]
> PulseWeaver is not an authentication system. It never verifies *who* sends a request; it checks whether the
> request's IP belongs to a registered device (or trusted network range) and whether that device's owner is allowed
> to reach the requested host.

Want to see the allow/deny model before touching your proxy? [Try it locally in a few minutes](#try-it-first--local-only-no-caddy).

**Quick links:** [is this for you?](#is-this-for-you) · [trial](#try-it-first--local-only-no-caddy) ·
[deployment](#deploy-for-real) · [security model](#security-model) ·
[troubleshooting](#troubleshooting-access-problems)

## Is this for you?

PulseWeaver fits best when you run a small self-hosted environment for people you already know: a household, club,
friend group, small team, office, or community. Instead of leaving every app's login page reachable by the internet, it
keeps the door closed by default and lets known devices in when their current IP is active and their owner has a grant.

It is a different posture from Authelia, authentik, or a mesh VPN, not a weaker copy of them:

| If you already use... | PulseWeaver's role |
|-----------------------|--------------------|
| Authelia / authentik / SSO | Complement: they verify identity after a request reaches a login flow; PulseWeaver keeps unknown IPs from reaching the app at all. |
| Tailscale / WireGuard | Optional complement: VPNs are often enough for your own devices; PulseWeaver can gate VPN subnets as trusted networks and help when selected non-VPN devices or family members need specific services. |
| Static IP allowlists | Replacement: heartbeats keep roaming phones and laptops current without editing proxy config. |

It may **not** be the right first layer if you need identity verification, per-person separation on a shared ISP/CGNAT
address, or a fully validated nginx/Traefik deployment today. IP is not identity; keep app-level auth where identity
matters.

---

## Why PulseWeaver

PulseWeaver exists because small self-hosted communities are not miniature enterprises. Enterprise auth tools are built
to keep a login door open to many unknown people and reject the wrong credentials. A self-hoster often wants the reverse:
do not expose the door unless the request comes from a known device or trusted network.

That is why PulseWeaver stays out of the app's authentication path. It is a **closed-door forward-auth target** behind
your reverse proxy, so it can protect HTTP-family traffic your proxy can run through forward auth; validate
protocol-specific behaviour for your proxy before relying on it.

A device registers its IP with a **heartbeat** — an authenticated `POST` whose *source IP* becomes active for that
device. Android phones and laptops on changing networks can use the
[Heartbeat Client](https://github.com/DiegoGuidaF/pulseweaver-heartbeat-client), a stable server can be entered by hand,
and `curl` on a timer works too. This makes temporary access a small-community operation — create or pair a device,
grant a host group, revoke it later — without teaching every self-hosted app a new identity model. The whole deployment
is **one binary** with the web UI embedded and a single SQLite file — no database server, no separate frontend to deploy.

---

## Features

- **Forward-auth gate** — your reverse proxy asks PulseWeaver on every request; answered from an in-memory cache, no
  per-request database work.
- **Heartbeat-tracked device IPs** — Android phones and laptops can keep their changing addresses registered
  automatically with a periodic POST; old addresses expire on their own when a device goes quiet or moves networks.
- **Per-user host access control** — deny-by-default grants over an admin-curated set of hosts, bundled into groups:
  "Tom can watch Jellyfin" is one checkbox. ([docs](docs/Host-Access-Control.md))
- **Per-device address rules** — an address lease (TTL) and a max-active-addresses cap keep each device's set of
  allowed IPs tight as it roams, with no manual cleanup. ([docs](docs/Connecting-Devices.md#recommended-settings-for-roaming-devices))
- **Network policies** — CIDR-range grants for networks you trust as a whole, like your LAN or a VPN subnet; overly
  broad ranges are rejected so a typo cannot allow the wider internet. ([docs](docs/Network-Policies.md))
- **Access logs & anomaly surface** — every decision the engine evaluates is recorded and filterable; dashboard with
  traffic over time, per-service splits, shared-IP flags, top denied IPs, and GeoIP enrichment. ([docs](docs/Observability.md))
- **Suggested hosts** — PulseWeaver proposes hostnames it sees in real traffic, so building the hosts list takes
  minutes, not an audit.
- **Device pairing** — one code (or QR scan) configures a device automatic heartbeat end-to-end, no manual URL/key entry.
- **Access verification** — ask "would IP X reach host Y?" and see exactly why, without sending real traffic.
  ([docs](docs/Host-Access-Control.md#testing-your-configuration))
- **Minimal non-root container** — the official image runs as a non-root user on a distroless runtime with no shell,
  package manager, or SQLite CLI. This reduces post-exploit tooling; it is not a sandbox.
- **Single binary** — embedded web UI, SQLite storage; one container, one volume, done.

---

## Screenshots

| Dashboard                                           | Host access control                                                     |
|-----------------------------------------------------|-------------------------------------------------------------------------|
| ![Dashboard](screenshots/03-dashboard-viewport.png) | ![Host access control](screenshots/09-host-access-control-viewport.png) |

| Devices                                           | Device addresses                                                         |
|---------------------------------------------------|--------------------------------------------------------------------------|
| ![Devices](screenshots/02-devices-viewport.png)   | ![Device addresses](screenshots/07-device-detail-addresses-viewport.png) |

---

## How it works

Two flows work together. Your **reverse proxy** calls `GET /api/policy-engine/verify-ip` on every request, asking
*"may the client at this IP reach this host?"* — PulseWeaver answers 200 (allow) or 403 (deny) from an in-memory
cache. A request is allowed two ways: the IP is an active address of a registered device whose user is allowed that
host, or — **as a fallback, only when the IP belongs to no registered device** — the IP falls inside a
[network policy](docs/Network-Policies.md) range that allows it. Everything else, including known devices asking for
hosts their user was never granted, is denied.

Before either check, the proxy's own IP is denied outright when it equals `TRUSTED_PROXY`; the proxy address can never
become a registered device or satisfy a network policy.

Your **devices** keep their current IP registered by sending periodic heartbeats (`POST /api/v1/heartbeat` with an
`X-API-Key` header). Each heartbeat adds or refreshes an address;
per-device [address rules](docs/Connecting-Devices.md#recommended-settings-for-roaming-devices)
expire the stale ones so the allowed set stays tight as a device moves between networks.

📖 [Detailed flow diagrams →](docs/How-It-Works.md)

---

## Quick start

### Try it first — local-only, no Caddy

If you only want to see the model before touching your real proxy, run PulseWeaver by itself. This **does not protect
anything**; it is just a local trial so you can create a host, register one IP, and see an allow/deny decision. The
container binds to `127.0.0.1` only, so the trial UI is not reachable from your LAN.

```bash
ADMIN_PW="$(openssl rand -base64 18)"
echo "PulseWeaver trial admin password: ${ADMIN_PW}"

docker run --rm --name pulseweaver-trial \
  -d \
  -p 127.0.0.1:8080:8080 \
  -e ADMIN_PASSWORD="${ADMIN_PW}" \
  -e POLICY_ENGINE_API_SECRET='trial-policy-secret-change-me' \
  -v pulseweaver-trial-data:/data \
  ghcr.io/diegoguidaf/pulseweaver:latest
```

The container runs in the background; `docker stop pulseweaver-trial` ends the trial when you're done.

Then open <http://127.0.0.1:8080> and log in as `admin` with the generated password above:

1. Add a host such as `jellyfin.local.test` under **Access → Hosts**.
2. Create a host group such as `trial` under **Access → Host Groups**, then add the host to it from the group's available-hosts list.
3. Grant yourself access under **Access → Users → admin** by assigning the `trial` group to the `admin` user.
4. Click **Save changes** whenever PulseWeaver shows the save bar; changes are staged until you save them.
5. Create a device owned by `admin` under **Devices**. In the **New device** dialog, set **Credential** to **API key**.
   Copy it when shown — API keys are displayed once, and PulseWeaver stores only their hash.
6. Register your local trial IP:

   ```bash
   curl -i -X POST http://127.0.0.1:8080/api/v1/heartbeat \
     -H "X-API-Key: wdk_YOUR_DEVICE_KEY"
   ```

   Both `curl` commands run from your machine, so the same source IP registered by the heartbeat is checked by
   `verify-ip`. In Docker this may look like a bridge address such as `172.17.0.1`; that is the source IP PulseWeaver
   sees from inside the container. You can see it under **Devices → your device → the Addresses tab**.

7. Verify the gate directly, without DNS or a real protected service:

   ```bash
   curl -i http://127.0.0.1:8080/api/policy-engine/verify-ip \
     -H "Authorization: Bearer trial-policy-secret-change-me" \
     -H "X-Forwarded-Host: jellyfin.local.test"
   ```

   The granted host should return `200`. A host you did not add or grant should return `403`:

   ```bash
   curl -i http://127.0.0.1:8080/api/policy-engine/verify-ip \
     -H "Authorization: Bearer trial-policy-secret-change-me" \
     -H "X-Forwarded-Host: never-added.local.test"
   ```

   You can also use **Auditing → Access Verification** with the source IP shown on the device after the heartbeat.

You may see a `TRUSTED_PROXY` warning in the trial logs. That is expected when running without a reverse proxy and is
safe to ignore for this local-only trial. If port `8080` is already in use, change both sides of the `-p` mapping and
the `127.0.0.1` URLs above to another local port.

If the trial returns an unexpected `403`, open **Auditing → Access Logs** first: it will usually show whether the source
IP was never registered, the host was not granted, or the trial secret was mistyped.

Clean up when done (the container self-removes because of `--rm`; the named volume is the only persistent piece):

```bash
docker stop pulseweaver-trial
docker volume rm pulseweaver-trial-data
docker image rm ghcr.io/diegoguidaf/pulseweaver:latest  # optional; removes the downloaded image
```

Ready to deploy for real? Keep the model, but use fresh secrets, a pinned release tag, Caddy, and the public/private
endpoint split below — do not reuse the trial secret.

### Deploy for real

Getting to a working deployment is **three steps**:

1. **Deploy** PulseWeaver behind Caddy.
2. **Configure the server** — log in, add your hosts, bundle them into groups, grant users, create devices.
3. **Connect each device** so its IP stays registered.

When all three are done, protected services are reachable only from active registered addresses or trusted network ranges
with matching grants. On shared NAT or CGNAT, anyone sharing the visible IP may inherit reachability, so keep app-level
auth where identity matters. The rest of this section walks each step; deeper material is linked out so this stays a
quickstart.

> [!NOTE]
> PulseWeaver is in **beta**. For production-ish deployments, pin a release tag, read release notes before updating, and
> expect occasional breaking changes before 1.0.

### Step 1 — Deploy behind Caddy

> [!NOTE]
> The example below uses **Caddy**, currently the only reverse proxy configuration tested and validated by this project.
> Other proxies that support forward auth should work on the same principles; see [Proxy integration](#proxy-integration).

The easiest way to run PulseWeaver is alongside Caddy with Docker Compose. Three values must agree so PulseWeaver can
tell which incoming connection is your proxy (and therefore read the *real* client IP from it, instead of seeing
Caddy's own address on every request):

- **`ipv4_address`** pins Caddy to a fixed IP on the shared Docker network.
- **`CADDY_IP`** in `.env` holds that same address.
- **`TRUSTED_PROXY`** in PulseWeaver's environment is set to `${CADDY_IP}`.

Why this matters in one line: a device behind a proxy never connects to PulseWeaver directly, so PulseWeaver has to be
told which peer is the proxy in order to trust the forwarded client IP. The full reasoning is in
[Understanding TRUSTED_PROXY](docs/Understanding-TRUSTED_PROXY.md). `POLICY_ENGINE_API_SECRET` is defined once in
`.env` and injected into both containers.

> [!TIP]
> Already have Caddy running? Keep your existing Caddy service and add only the `pulseweaver` service below. The
> important parts are that both containers share a Docker network, Caddy has a fixed IP on that network, and your
> existing Caddy service receives `PULSEWEAVER_POLICY_ENGINE_API_SECRET`.
> If that network already exists, declare it as external instead of letting this compose file create it:
> `networks: { proxy: { external: true, name: your-existing-caddy-network } }`. Caddy still needs a stable
> `ipv4_address` on that network, and changing a running service's network settings requires recreating that container.

```yaml
# docker-compose.yml
services:
  caddy:
    image: caddy:2.11.1 # Example version, ensure you're running latest
    container_name: caddy # Omit or change if you already run a container named "caddy".
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
        ipv4_address: ${CADDY_IP} # Fixed IP so we can wire it to PulseWeaver's TRUSTED_PROXY
    depends_on:
      - pulseweaver

  pulseweaver:
    image: ghcr.io/diegoguidaf/pulseweaver:<release-tag> # Pick a published tag; avoid :latest for production-ish use.
    container_name: pulseweaver
    restart: unless-stopped
    expose: # Do not publish 8080 on the host; Caddy reaches PulseWeaver on the internal proxy network.
      - 8080
    environment:
      ADMIN_PASSWORD: ${PULSEWEAVER_ADMIN_PASSWORD}
      SERVER_PORT: 8080
      TRUSTED_PROXY: ${CADDY_IP}      # Required behind a proxy; single Caddy container IP only, no CIDR.
      POLICY_ENGINE_API_SECRET: ${PULSEWEAVER_POLICY_ENGINE_API_SECRET} # PulseWeaver's key; .env uses the longer name so Caddy can read it too.
      TZ: ${TZ}
    volumes:
      - ./pulseweaver/data:/data   # Bind mount; ensure writable by non-root (chown 65532:65532). Named volumes work too.
    networks:
      - proxy

networks:
  proxy:
    driver: bridge
    ipam:
      config:
        # ⚠️ Example subnet only — NOT a recommendation. Pick a range that does not
        # collide with anything on your network, then update CADDY_IP to match.
        - subnet: 172.20.0.0/24
          gateway: 172.20.0.1
          ip_range: 172.20.0.128/25  # Restrict auto-assigned IPs to upper half so nothing claims Caddy's address
```

A minimal `.env` alongside it:

```dotenv
PULSEWEAVER_POLICY_ENGINE_API_SECRET=a-very-long-random-secret-at-least-16-chars
PULSEWEAVER_ADMIN_PASSWORD=a-strong-admin-password
CADDY_IP=172.20.0.2   # Must be inside the subnet above, in its lower half — keep it off the upper, auto-assigned range
TZ=Europe/Madrid
```

> [!TIP]
> Generate strong values for the two secrets with OpenSSL:
> ```bash
> openssl rand -base64 32   # PULSEWEAVER_POLICY_ENGINE_API_SECRET
> openssl rand -base64 24   # PULSEWEAVER_ADMIN_PASSWORD
> ```

> [!NOTE]
> The `172.20.0.0/24` subnet above is **an example, not advice** — copy-pasting it blindly is a common source of
> "works on the tutorial, breaks on my network" issues. `TRUSTED_PROXY` takes a single IP, not a CIDR range, and the
> compose file reserves the upper half of the subnet so nothing can accidentally claim Caddy's address. The full
> reasoning: [Understanding TRUSTED_PROXY](docs/Understanding-TRUSTED_PROXY.md#choosing-the-proxy-ip-in-docker-compose).

> [!TIP]
> Back up `/data` regularly and before updates. PulseWeaver stores its SQLite database there; see
> [Data Persistence](docs/Data-Persistence.md) for safe backup methods and storage caveats.

**Exposure summary:**

| Surface | Should be public? | Notes |
|---------|-------------------|-------|
| Caddy ports `80` / `443` | Yes | Caddy is the only container publishing host ports. |
| Device domain, e.g. `pw-device.example.com` | Yes, narrowly | Only `/api/v1/heartbeat` and `/api/v1/device-pair`; every other path should return 404. |
| Admin UI, e.g. `pulseweaver.example.com` | No | Keep it on a private network/VPN, behind an auth middleware, or self-gated after bootstrapping. |
| PulseWeaver container `:8080` | No | It should stay reachable only from Caddy on the Docker network; do not replace `expose` with `ports`. |

After first start, check the container logs before sending real device heartbeats through the proxy:

```bash
docker logs pulseweaver | grep -i TRUSTED_PROXY || true
```

If this prints the startup warning that `TRUSTED_PROXY is not configured` while you are running behind Caddy, fix that
before continuing. No such warning means this specific check passed.

Then add the gate to any site you want to protect. The `forward_auth` block asks PulseWeaver before forwarding:

```caddy
your-service.example.com {
    # 'pulseweaver' is the Compose service name; update it if you rename the service.
    forward_auth pulseweaver:8080 {
        uri /api/policy-engine/verify-ip
        header_up X-Real-IP {http.request.remote.host}
        header_up Authorization "Bearer {$PULSEWEAVER_POLICY_ENGINE_API_SECRET}"
    }
    reverse_proxy your-service:port
}
```

The endpoint is **fail-closed**: a missing header, wrong secret, unregistered IP, or a host the user was never granted
all return the same `403`. If PulseWeaver is unreachable, Caddy returns an upstream error and protected services become
temporarily unreachable; they do not become open.

Protecting more than a service or two? Define the gate once as a reusable Caddy snippet and import it from each site:

```caddy
(pulseweaver) {
    forward_auth pulseweaver:8080 {
        uri /api/policy-engine/verify-ip
        header_up X-Real-IP {http.request.remote.host}
        header_up Authorization "Bearer {$PULSEWEAVER_POLICY_ENGINE_API_SECRET}"
    }
}

your-service.example.com {
    import pulseweaver
    reverse_proxy your-service:port
}
```

> [!IMPORTANT]
> The heartbeat and pairing endpoints also need their own public device-domain block **without** the forward-auth gate.
> Set that up before distributing pairing codes, otherwise devices on new networks cannot register their IP:
>
> ```caddy
> pw-device.example.com {
>     @device-endpoints path /api/v1/heartbeat /api/v1/device-pair
>     handle @device-endpoints {
>         reverse_proxy pulseweaver:8080 {
>             header_up X-Real-IP {http.request.remote.host}
>         }
>     }
>     respond 404
> }
> ```

📖 [Full Caddy setup guide →](docs/Caddy-Setup.md) — the recommended two-domain split (keep the admin UI off the public
internet), reusable snippets, device endpoints, and other proxy support.

### Step 2 — Configure the server

On first startup PulseWeaver bootstraps an `admin` user from the `ADMIN_PASSWORD` variable. Log in at your admin URL,
then set up access — admins think in *people and services*, so the UI does too:

1. **Add your hosts** under **Access → Hosts** — the hostnames you proxy (`jellyfin.example.org`, …). Once traffic
   flows, PulseWeaver suggests hosts it has actually seen, so you can promote them in a click.
2. **Bundle them into groups** under **Access → Host Groups** — *media*, *storage*, *home automation*. Groups are the
   unit access is granted in.
3. **Grant users** under **Access → Users** — assign each user the groups they should reach. Deny-by-default means a
   user reaches nothing until you do this. Users are admin-managed identities; end users do not log in to PulseWeaver.
4. **Create devices** under **Devices** — one per Android phone/laptop/server that needs access, each owned by a user.
5. **Save staged changes** whenever the bottom save bar appears.

> [!NOTE]
> Deny-by-default applies to your own account too. Assign your admin user a host group, or deliberately enable
> **Bypass host check**, before you expect to reach protected services yourself.
> The bootstrap `admin` account is the only superadmin; promoted admins can manage access and devices, but not create,
> promote, demote, or delete users.
> If you gate the admin UI with PulseWeaver itself, follow the
> [Caddy bootstrapping note](docs/Caddy-Setup.md#option-c--protect-with-pulseweavers-own-gate) before turning the gate on.

📖 [Host Access Control →](docs/Host-Access-Control.md)

### Step 3 — Connect each device

A device gets access by keeping its current IP registered. The simplest path is **pairing**: create a pairing for the
device, and the user scans/pastes the code into the heartbeat client, which configures itself and starts heartbeating.
Devices with a stable IP can instead have their address entered by hand — no heartbeat needed.

The heartbeat and pairing endpoints must be reachable from devices **without** the forward-auth gate (a device on a new
network has to be able to register that network's IP). Treat this as a prerequisite before distributing pairing codes:
the [Caddy setup guide](docs/Caddy-Setup.md) shows the recommended two-domain configuration that exposes only those two
endpoints publicly.

📖 [Connecting devices →](docs/Connecting-Devices.md) — pairing, the heartbeat client, lightweight `curl`/systemd/launchd
setups, and the recommended lease settings for roaming.

### Configuration reference

| Variable                   | Required           | Default                    | Description                                                                                                                                                                                                                                                                                        |
|----------------------------|--------------------|----------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `ADMIN_PASSWORD`           | Yes (8–72 chars)   | —                          | Password for the bootstrap `admin` account, applied **once** when that account is first created and stored as a bcrypt hash. Change it from the UI afterward; it is ignored on later startups and can be removed from `.env` after first login.                                                     |
| `POLICY_ENGINE_API_SECRET` | Yes (min 16 chars) | —                          | Shared secret that authenticates your proxy's calls to `verify-ip`, so callers without the secret cannot ask PulseWeaver for decisions. Keep `verify-ip` network-private anyway. Minimum 16 characters; use a long random value.                                                                    |
| `SERVER_PORT`              | No                 | `8080`                     | Port PulseWeaver listens on.                                                                                                                                                                                                                                                                       |
| `TRUSTED_PROXY`            | Yes behind a proxy | —                          | Single IP of your reverse proxy. Leave unset only for direct-access, no-proxy trials. PulseWeaver is designed to run behind a trusted proxy, and without it the proxy's own IP can be mistaken for the client. See [Understanding TRUSTED_PROXY](docs/Understanding-TRUSTED_PROXY.md).              |
| `RULE_CHECK_INTERVAL`      | No                 | `1m`                       | How often the background scheduler runs (lease expiry, rollups, retention). Set it at or below the lowest address-lease TTL you'll use.                                                                                                                                                            |
| `DATA_RETENTION_DAYS`      | No                 | `30`                       | Days to keep **per-request detail** — the access log and device address history. `0` disables pruning. Aggregated traffic is kept longer (see below), so old detail ages out without losing dashboard history. See [Observability](docs/Observability.md).                                         |
| `AGGREGATE_RETENTION_DAYS` | No                 | `365`                      | Days to keep the **hourly traffic aggregates** behind long dashboard windows. `0` = keep forever. Must be `0` or ≥ `DATA_RETENTION_DAYS`. See [Observability](docs/Observability.md).                                                                                                              |
| `GEOIP_ENABLED`            | No                 | `true`                     | Resolve client IPs to country/ASN for logs and dashboard. Uses free DB-IP databases, checks daily for stale files, downloads the current monthly DB-IP files when needed, and does no per-request external lookup. See [Observability](docs/Observability.md#geoip).                                |
| `DB_DIR`                   | No                 | `./data` (Docker: `/data`) | Directory for the SQLite database. The file is plaintext at rest except for fields PulseWeaver hashes before storing them; protect the volume with host permissions/backups/full-disk encryption as appropriate. See [Data Persistence](docs/Data-Persistence.md).                                  |
| `TZ`                       | No                 | `UTC`                      | Application timezone for explicit wall-clock operations. Persisted timestamps are UTC; API timestamps are serialized as UTC RFC3339.                                                                                                                                                               |
| `LOG_LEVEL`                | No                 | `info`                     | Log level: `debug`, `info`, `warn`, `error`.                                                                                                                                                                                                                                                       |
| `LOG_FORMAT`               | No                 | `text`                     | Log format: `text` (human-readable) or `json`.                                                                                                                                                                                                                                                     |
| `LOG_COLOR`                | No                 | `true`                     | Use coloured output for `text` format.                                                                                                                                                                                                                                                             |

> [!WARNING]
> If a device heartbeats through a reverse proxy while `TRUSTED_PROXY` is unset, PulseWeaver may register the proxy's
> IP as that device's address. Every request coming through that proxy would then look like the same allowed device. Set
> `TRUSTED_PROXY` before sending heartbeats through a proxy.

### Updates

For production-ish deployments, pin a specific image tag instead of `:latest`, back up `/data` before updating, and read
the [release notes](https://github.com/diegoguidaf/pulseweaver/releases). Database migrations run automatically at
startup. Before 1.0, assume downgrade safety is not guaranteed unless a release note explicitly says otherwise.

---

## Key concepts

| Concept            | Description                                                                                                                                                               |
|--------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **User**           | A person. Devices belong to users, and access is granted to users — "admin" is a role on a user, not a separate account type.                                             |
| **Device**         | A logical endpoint (phone, laptop, server…) with a unique API key, owned by a user.                                                                                       |
| **Address**        | An IP address (v4 or v6) linked to a device. A device holds a *list* of addresses, each enabled or disabled.                                                              |
| **Heartbeat**      | A device call to `/api/v1/heartbeat` that registers the caller's current IP as an active address of the device.                                                           |
| **Address lease**  | A per-device TTL\*. When no heartbeat refreshes an address before the TTL expires, PulseWeaver's background scheduler disables it.                                        |
| **Host**           | An admin-curated hostname that can be granted to users, e.g. `jellyfin.example.org`.                                                                                      |
| **Host group**     | A named bundle of hosts ("media", "storage"). **Groups are the only way hosts are granted** — you assign groups to users and to network policies, never individual hosts. |
| **Network policy** | A CIDR-range grant, used as a fallback for clients that are not registered devices — e.g. "the whole home LAN may reach these hosts."                                     |
| **Forward auth**   | The `GET /api/policy-engine/verify-ip` endpoint. Your reverse proxy calls this on every request.                                                                          |

> \***TTL**: Time-To-Live

---

## Performance

PulseWeaver sits on **every request** your reverse proxy forwards, so its overhead matters. Each decision is a
constant-time lookup served from an **in-memory cache** — no database work per request — so the measured cost in the
Caddy setup is small:

| Metric | Figure |
|--------|--------|
| Latency added per request | **sub-millisecond** decision in the app; ~1.6 ms end-to-end through a reverse proxy (~5 ms p99) |
| CPU under load | ~1.5% of one core at ~50 req/s — roughly 100× a typical self-hosted load |
| Memory | ~24 MB resident |
| Stress-tested to | ~15,000 req/s sustained, zero connection failures |

Order-of-magnitude figures on commodity hardware. See **[Testing & Validation](docs/Testing-and-Validation.md)**
for the method, the proxy-vs-direct breakdown, and the security/a11y assessment approach.

---

## Device pairing

Pairing onboards a device onto a heartbeat client with no manual URL/key entry — an admin generates a code, the user
scans or pastes it, and the client configures itself.

1. Create the device under **Devices** and assign it to the user, then open that device and create a **pairing** for it
   — choosing the heartbeat server URL, interval, biometric settings, and an expiry window.
2. PulseWeaver generates a single-use **pairing code** (an opaque string that encodes the server URL and a random
   token). Share it as a QR (rendered client-side in the browser) or copyable text.
3. The user pastes the code into the
   [Heartbeat Client](https://github.com/DiegoGuidaF/pulseweaver-heartbeat-client) ([downloads](https://github.com/DiegoGuidaF/pulseweaver-heartbeat-client/releases)).
   The app decodes the server URL, calls the claim endpoint, and receives the device's full configuration —
   **including a freshly generated API key**.
4. The code is invalidated and the app starts heartbeating — no manual setup.

Pairing configures an **existing** device; it does not create one. The API key is generated at claim time and returned
exactly once — PulseWeaver only ever stores its hash. This separation means you can pre-configure a device before
pairing, and re-pair a device later (rotating its key) without recreating it. The code is retrievable by the admin
until claimed or expired.

> [!TIP]
> The heartbeat and pairing endpoints must be reachable by devices without the gate **before you share pairing codes**.
> Rather than exposing the whole admin UI, the recommended setup publishes **only those two endpoints** under a separate
> device domain (e.g. `pw-device.example.com`) and keeps the panel private. See the
> [Caddy setup guide](docs/Caddy-Setup.md) and [Connecting devices](docs/Connecting-Devices.md).
>
> Heartbeats still require a per-device API key and are rate-limited by client IP. Pairing requires a single-use code
> before it expires and is rate-limited by client IP. If you need stricter throttling for your public device domain, add
> it at the reverse proxy; keep the admin API private.

Client-specific installation, background scheduling, and runtime caveats live in the
[Heartbeat Client documentation](https://github.com/DiegoGuidaF/pulseweaver-heartbeat-client/blob/main/docs/app.md);
this repository documents the server-side endpoints and access model.

---

## Proxy integration

PulseWeaver's model should translate to reverse proxies with forward-auth/auth-request support, but **Caddy** is
currently the only tested and validated configuration. If you have a working setup with nginx, Traefik, or another proxy,
[open an issue or PR](https://github.com/diegoguidaf/pulseweaver/issues) and community-validated configurations will be
added to the documentation.

For a configuration to be considered validated, it must demonstrate all of these: the forward-auth call, real client IP
forwarding, `X-Forwarded-Host`, the shared `Authorization` secret, device endpoints exposed without the gate, a private
admin-UI posture, and a single-IP `TRUSTED_PROXY` binding.

📖 [Full Caddy setup guide →](docs/Caddy-Setup.md) ·
[Generic proxy requirements →](docs/Caddy-Setup.md#other-reverse-proxies) ·
[Connecting devices →](docs/Connecting-Devices.md)

---

## Security model

PulseWeaver is an **IP gate with per-user host authorization** — it blocks unknown IPs before they reach any service,
and decides per user which hosts the known ones may reach. Who is behind a request is inferred from its IP, never
verified, so it complements rather than replaces app-level auth. When several users share one IP, only hosts that
**all** of them may reach are allowed — the strictest grant wins.

> [!WARNING]
> Do not expose the PulseWeaver admin UI directly to the public internet. Publish only the heartbeat/pairing endpoints
> publicly; keep the panel private, behind an identity proxy, or behind PulseWeaver's own gate after bootstrapping.

| ✅ Works well                                    | ⚠️ Not enough on its own                                                                                  |
|-------------------------------------------------|-----------------------------------------------------------------------------------------------------------|
| Services that break behind SSO proxies          | Verifying *who* a user is (identity is IP-inferred)                                                       |
| Homelab with a small set of trusted networks    | Telling co-tenants apart on ISP-level CGNAT — everyone behind the same ISP IP is indistinguishable        |
| Reducing blast radius of unpatched CVEs         | A compromised active network — an attacker on the same active IP inherits that IP's grants until it ages out |
| Zero-config travel access via heartbeat + lease | Replacing app-level authentication                                                                        |

Grant, address, and network-policy changes normally update the in-memory decision cache within milliseconds. A separate
periodic reconcile rebuilds the cache as a safety backstop if a change notification is missed or a refresh fails.
PulseWeaver should be **one layer** in a defence-in-depth strategy, not the only one.

Keep `verify-ip` network-private. It is intentionally not rate-limited because Caddy calls it on every protected request;
the bearer secret authenticates your proxy, but the endpoint should still be reachable only from that proxy.

The official container image runs as a non-root user and is based on a distroless runtime: it ships the PulseWeaver
binary, but no shell, package manager, or SQLite CLI. That reduces post-exploit tooling inside the container, but the
application still needs write access to `/data`, so continue to protect the mounted volume and host.

📖 **Deep dives:** [Security Model](docs/Security-Model.md) · [Shared-IP Model](docs/Shared-IP-Model.md) ·
[Understanding TRUSTED_PROXY](docs/Understanding-TRUSTED_PROXY.md)

---

## Troubleshooting access problems

First identify the layer. A plain `403` usually means PulseWeaver denied the request. A 5xx/upstream error or timeout
usually means PulseWeaver or the protected service is unreachable. An app-specific error after a successful allow is
inside the protected service.

PulseWeaver intentionally returns the same plain `403` for most denied requests, so the response does not reveal what
exists or who would be allowed. Use the admin UI to diagnose denials:

1. **Check Auditing → Access Logs.** Filter for denied requests. `ip_not_registered` means the source IP is not an active
   device address and did not match a network policy; `host_not_allowed` means the IP belongs to a known user/device but
   that user lacks a grant for the host; `invalid_token` means the proxy reached PulseWeaver with the wrong shared
   secret.
2. **Use Auditing → Access Verification.** Take the source IP and host from the denied row and verify the same decision
   path without sending real traffic.
3. **Check the matched device's Addresses tab.** If the row matched no device, look up the user's device by owner. If the
   device has no active address, its heartbeat stopped, expired, or is being sent from a different network than the one
   you're testing.
4. **If nothing appears in the access log, check the proxy.** A missing `Authorization` header is rejected before the
   decision engine runs, so it produces no log entry. Verify the `forward_auth` block and
   `POLICY_ENGINE_API_SECRET`.
5. **If the proxy reports an upstream error, check PulseWeaver itself.** `docker logs pulseweaver` is the first place to
   look; temporarily set `LOG_LEVEL=debug` if you need more detailed request/decision logs.

📖 [Observability →](docs/Observability.md) · [Connecting devices →](docs/Connecting-Devices.md) ·
[X-Real-IP registration error →](docs/Caddy-Setup.md#trusted-proxy-ip-addresses-cannot-be-registered)

---

## Project status & support

PulseWeaver is in **beta**. The core gate, host access control, and observability surfaces are stable and in daily
use, including the author's own deployment, but expect occasional rough edges and breaking changes before a 1.0 release.
Security-oriented checks are run against a production-like deployment, and the project is open source so the model and
implementation can be reviewed. Until beta is removed, evaluate it carefully for your own risk tolerance — see
[Testing & Validation](docs/Testing-and-Validation.md).

- 🐛 **Bug reports** → [GitHub Issues](https://github.com/diegoguidaf/pulseweaver/issues)
- 💬 **Questions & ideas** → [GitHub Discussions](https://github.com/diegoguidaf/pulseweaver/discussions)
- 🔒 **Security issues** → [SECURITY.md](SECURITY.md) — please use GitHub private vulnerability reporting.
- 🔀 **Working nginx / Traefik config?** Contributions are very welcome — see [Proxy integration](#proxy-integration).

---

## Development

PulseWeaver compiles to a **single binary** with the frontend SPA embedded. The frontend is built with Vite and
embedded at compile time — no separate web server needed in production.

### Quick start for local development

For local runs (no Docker), the app stores its database under `./data` relative to where you run it, and creates the
directory on first start — no root, no `/data` setup needed. Override with `DB_DIR` if you want it elsewhere.

```bash
# Backend (hot reload via Air)
make back-dev

# Frontend (Vite dev server, in a separate terminal)
make front-dev
```

### Useful make targets

Targets use a `back-*` / `front-*` prefix; run `make help` for the full list.

| Command           | Description                                                 |
|-------------------|-------------------------------------------------------------|
| `make build`      | Full production build → `bin/pulseweaver`                   |
| `make back-test`  | Run all Go tests                                            |
| `make back-lint`  | Format + lint                                               |
| `make front-lint` | ESLint + TypeScript type-check                              |
| `make check`      | Full validation: lint + type-check + tests (back + front)   |
| `make api`        | Regenerate backend + frontend types from `api/openapi.yaml` |

### Further reading

- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — high-level system map: layers, the API contract, request flow,
  single-binary build. Start here to understand how the pieces fit together.
- [`CODEBASE-Backend.md`](CODEBASE-Backend.md) — backend package structure, domain boundaries, service lifecycle,
  observer pattern.
- [`CODEBASE-Frontend.md`](CODEBASE-Frontend.md) — frontend directory structure, routing, hook conventions, UX surfaces.
- [`AGENTS.md`](AGENTS.md) — instructions for AI coding agents: conventions, commands, and hard rules.
- [`api/openapi.yaml`](api/openapi.yaml) — API schema; single source of truth for all endpoints and types.

### A note on AI usage

This project has not been vibe-coded. The author is a software developer with 9+ years of experience (primary stack:
Java/Kotlin). This is a first Go project and first React frontend. AI has been used extensively to accelerate tests and
frontend work, and as a learning tool — not as a replacement for understanding the code.
