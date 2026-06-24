# Reverse proxy setup — Caddy

This guide walks through the full Caddy configuration for a PulseWeaver deployment. For the
explanation of why `TRUSTED_PROXY` and `X-Real-IP` are required, see
[Understanding TRUSTED_PROXY](Understanding-TRUSTED_PROXY.md).

---

## Recommended domain split

This is the most important security decision in the setup, so it comes first.
Split PulseWeaver across two domains, two responsibilities:

| Domain | Purpose | Internet-facing? |
|---|---|---|
| `pw-device.example.com` | Heartbeat and device pairing — reachable by devices | ✅ Yes |
| `pulseweaver.example.com` | Admin UI | ❌ No — see [Admin UI](#step-3--admin-ui) |

Keeping them separate means the public internet can only reach the two device endpoints. The admin
panel, API, and everything else never appear on a publicly routable domain.

---

## The PulseWeaver gate

The `forward_auth` block below is the building block for protecting any service with PulseWeaver.
Before forwarding a request, it asks PulseWeaver whether the client may reach the requested host;
only clients with a matching grant — a registered device whose user is allowed the host, or a
[network policy](Network-Policies.md) covering the IP — pass. Caddy forwards the requested hostname
automatically (`X-Forwarded-Host`), so no extra directive is needed for the host check.

```caddy
forward_auth pulseweaver:8080 {
    uri /api/policy-engine/verify-ip
    header_up X-Real-IP {http.request.remote.host}
    header_up Authorization "Bearer {$PULSEWEAVER_POLICY_ENGINE_API_SECRET}"
}
```

The endpoint is **fail-closed**: a missing header, wrong secret, unregistered IP, or a host the
user was never granted all return the same `403`.

`PULSEWEAVER_POLICY_ENGINE_API_SECRET` must be the same value in both PulseWeaver's and Caddy's
environment. It authenticates the proxy-to-PulseWeaver call so external clients cannot query the
verify-ip endpoint directly.

---

## Step 1 — Device endpoints (public domain)

These two endpoints must be reachable from your devices — heartbeats and the initial device pairing
both call them from outside your network.

```caddy
pw-device.example.com {
    @device-endpoints path /api/v1/heartbeat /api/v1/device-pair
    handle @device-endpoints {
        reverse_proxy pulseweaver:8080 {
            header_up X-Real-IP {http.request.remote.host}
        }
    }
    respond 404
}
```

`header_up X-Real-IP {http.request.remote.host}` is **required** on this block. Without it,
PulseWeaver receives Caddy's IP for every heartbeat — since `TRUSTED_PROXY` prevents the proxy's IP
from being registered, all heartbeats fail. See [Troubleshooting](#troubleshooting) for details.

`respond 404` ensures every other path on this domain returns 404, keeping the attack surface to
exactly the two device endpoints.

> [!TIP]
> Set the **Heartbeat server URL** in the device invite to `https://pw-device.example.com`. The
> heartbeat client uses that URL for both the initial pairing and all subsequent heartbeats.

---

## Step 2 — Protecting your services

Any service you want to gate behind PulseWeaver gets the `forward_auth` block from above:

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

Requests without a matching grant receive a `403` before they reach `your-service`.

### Protecting many services without repeating yourself

A homelab usually fronts a dozen services, and copy-pasting the `forward_auth`
block into each site is how subtle mistakes creep in — a mistyped `uri`, a
forgotten `header_up`, a stale secret. Define the gate **once** as a Caddy
[snippet](https://caddyserver.com/docs/caddyfile/concepts#snippets) and `import`
it everywhere:

```caddy
# Define once, near the top of your Caddyfile.
(pulseweaver) {
    forward_auth pulseweaver:8080 {
        uri /api/policy-engine/verify-ip
        header_up X-Real-IP {http.request.remote.host}
        header_up Authorization "Bearer {$PULSEWEAVER_POLICY_ENGINE_API_SECRET}"
    }
}

# Then every protected site is two lines.
jellyfin.example.com {
    import pulseweaver
    reverse_proxy jellyfin:8096
}

nextcloud.example.com {
    import pulseweaver
    reverse_proxy nextcloud:80
}
```

Now the gate has a single definition: fix or audit it in one place, and every
site stays consistent. This is the recommended way to protect more than one or
two services.

> [!IMPORTANT]
> Adding the `forward_auth` block is only half the job. Access is **deny-by-default**: add
> `your-service.example.com` as a host and grant it to the users who should reach it — see
> [Host Access Control](Host-Access-Control.md). Until you do, *everyone* is denied — including you,
> unless your own user has a grant for it or **bypass host check** enabled. The safe way to confirm a
> setup is **Auditing → Access Verification** ("would IP X reach host Y?"), not your own browser.

---

## Step 3 — Admin UI

> [!IMPORTANT]
> **Do not expose the admin UI to the public internet.** This is not a concern about the security
> of the panel itself — it is a general principle: administrative interfaces should not be accessible
> to anonymous internet traffic when there is a better option. PulseWeaver is a security tool; apply
> the same posture to it that it applies to everything else. The options below are ordered from
> simplest and most reliable to most complex.

### Option A — VPN or private network *(recommended)*

The simplest and most robust option: don't publish the admin domain at all. Keep
`pulseweaver.example.com` as a private hostname — accessible on your home network or over a VPN
such as [Tailscale](https://tailscale.com/) — and no Caddy-level gate is needed because the domain
is never reachable from outside. No configuration to get wrong, no bootstrapping concerns, and
access is controlled entirely at the network layer.

### Option B — Use an existing auth middleware

If you already run [Authelia](https://www.authelia.com/), [Authentik](https://goauthentik.io/), or
a similar identity proxy, put PulseWeaver behind it. The configuration is specific to each product
but the concept is the same: add an authentication gate in front of the admin domain. This is a good
fit if you already have one of these running for other services and want a consistent access model.

### Option C — Protect with PulseWeaver's own gate

PulseWeaver can guard its own admin panel using the same gate it provides to other services. Only
registered devices whose user is allowed the panel's hostname get through; everyone else is denied —
which is usually exactly what you want for an admin UI.

Because the panel is gated like any other host, **you must grant your operator account access to it**:
either add `pulseweaver.example.com` as a host and put it in a group assigned to that user, or enable
**bypass host check** on that user so they reach every host. Bypass is **off by default** and is not
implied by the admin role, so the panel is not reachable for an admin until you grant it one of these
two ways.

```caddy
pulseweaver.example.com {
    encode zstd gzip
    forward_auth pulseweaver:8080 {
        uri /api/policy-engine/verify-ip
        header_up X-Real-IP {http.request.remote.host}
        header_up Authorization "Bearer {$PULSEWEAVER_POLICY_ENGINE_API_SECRET}"
    }
    reverse_proxy pulseweaver:8080 {
        header_up X-Real-IP {http.request.remote.host}
    }
}
```

Note the `header_up X-Real-IP` on the **`reverse_proxy`** block — not just the `forward_auth` block.
The admin UI's **Register my IP** button works by sending a heartbeat through this same proxy path, so
it needs the real client IP just as the gate does. Leave it off and PulseWeaver sees Caddy's IP
instead, the registration is refused, and the button appears broken — see
[Troubleshooting](#troubleshooting).

**The chicken-and-egg problem.** This option works, but it has a bootstrapping constraint worth
understanding: to access the admin UI you need a registered IP, and to register an IP through the
UI you need to access the admin UI. The way out is to never add the gate until you already have
at least one device registered through an alternative path:

1. Start without the `forward_auth` block — the admin UI is accessible without a gate.
2. Install the heartbeat client on your admin machine and configure it to send heartbeats.
3. Confirm your IP appears as active in PulseWeaver.
4. Only then add the `forward_auth` block to the Caddyfile and reload Caddy.

If you ever lock yourself out (the heartbeat client stops, your IP changes, etc.), you can recover
by temporarily accessing PulseWeaver directly — either from within the Docker network or via a
private network — to re-register your address, then remove the temporary access.

---

## Other reverse proxies

PulseWeaver has been tested with Caddy. Other reverse proxies that support forward auth should work
on the same principles — the table below maps each Caddy directive to its generic equivalent so you
can find the right option in your proxy's documentation.

| Caddy directive | What it does | What to look for |
|---|---|---|
| `reverse_proxy` + `header_up X-Real-IP {http.request.remote.host}` | Forwards the real client IP to PulseWeaver | "set request header", `proxy_set_header X-Real-IP` |
| `forward_auth` → `uri /api/policy-engine/verify-ip` | Sub-request auth check before forwarding | `auth_request` (nginx), `forwardAuth` (Traefik) |
| `header_up Authorization "Bearer …"` | Authenticates the proxy→PulseWeaver call | Pass a static bearer token to the auth endpoint |
| `respond 404` | Default-deny — block all unmatched paths | `return 404` / default location block |

Official configurations for nginx, Traefik, and other proxies will be added once they have been
tested and validated. If you have a working setup,
[open an issue or PR](https://github.com/diegoguidaf/pulseweaver/issues) and we'll add it here.

---

## Troubleshooting

### "Trusted proxy IP addresses cannot be registered"

This error appears when you click **Register my IP** in the admin UI, or when a heartbeat client
sends a heartbeat through the proxy, and PulseWeaver refuses to register the IP.

**What happened:** PulseWeaver detected that the IP it received matches the configured `TRUSTED_PROXY`
address. That IP is always blocked from registration — allowing it would grant every proxied request
a free pass regardless of the real client.

**Root cause:** A `reverse_proxy pulseweaver:8080` block is missing
`header_up X-Real-IP {http.request.remote.host}`. Caddy's `reverse_proxy` sets `X-Forwarded-For`
automatically but does **not** set `X-Real-IP`. PulseWeaver reads `X-Real-IP` and only trusts it
from `TRUSTED_PROXY`; without it the fallback is the raw connection peer — Caddy's own IP.

**Fix:** Add `header_up X-Real-IP {http.request.remote.host}` inside every
`reverse_proxy pulseweaver:8080` block in your Caddyfile — both the device endpoints domain and the
admin UI domain:

```caddy
reverse_proxy pulseweaver:8080 {
    header_up X-Real-IP {http.request.remote.host}
}
```

For a deeper explanation of how the trusted proxy check works, see
[Understanding TRUSTED_PROXY](Understanding-TRUSTED_PROXY.md).
