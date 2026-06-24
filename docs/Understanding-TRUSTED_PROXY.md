# Understanding TRUSTED_PROXY

## Why it matters

When a reverse proxy like Caddy sits in front of PulseWeaver, your device never connects to
PulseWeaver directly. The proxy receives your device's request, then forwards it on your behalf.
From PulseWeaver's point of view, every forwarded request appears to arrive from the proxy's own
IP address — not from your device.

To solve this, reverse proxies attach a header to each forwarded request that carries the original
client IP. PulseWeaver reads this header to know the real source of the request.

The catch: PulseWeaver cannot trust that header from just anyone. If it did, any client could
send a fake header claiming to be a trusted IP and walk straight through the gate. `TRUSTED_PROXY`
tells PulseWeaver exactly one IP address it will believe. Headers arriving from any other peer are
ignored — the request still proceeds using the peer's own IP rather than the spoofed value, and a
warning is written to the server log.

> [!WARNING]
> If you are running behind a reverse proxy and do not set `TRUSTED_PROXY`, PulseWeaver will see
> the proxy's IP for every request. If any device sends a heartbeat through that proxy, the proxy's
> IP gets registered — and from that point every proxied request passes the gate. PulseWeaver logs
> a warning at startup when `TRUSTED_PROXY` is not configured.

### Why X-Real-IP and not X-Forwarded-For?

`X-Forwarded-For` is an older header that builds up a comma-separated chain as a request passes
through multiple proxies: `X-Forwarded-For: device-ip, proxy1-ip, proxy2-ip`. A client can inject
a fake first entry before any proxy adds theirs, and there is no universal rule for which entry in
the chain to trust.

`X-Real-IP` is a single value **overwritten** by the immediate upstream proxy. There is no chain to
manipulate and no ambiguity about which value to read. Caddy's `forward_auth` subrequest is
constructed internally by Caddy — the client cannot inject headers into it at all — and we
explicitly set `X-Real-IP` to `{http.request.remote.host}`, making the intent clear.

## Technical deep-dive

### Why single IP, not a CIDR range

`TRUSTED_PROXY` accepts only a single IP address by design. Accepting a subnet
(e.g. `172.20.0.0/24`) would extend trust to every address in that range, including the Docker
network's gateway — typically the first address in the subnet (`172.20.0.1`). The gateway address
is reachable from the Docker host itself: any process on the host could send a request from that
address with an arbitrary `X-Real-IP` header and have it accepted as authoritative. A pinned single
IP (e.g. `172.20.0.2` for the Caddy container) avoids this entirely. If your proxy IP ever changes,
update `TRUSTED_PROXY` explicitly — the friction is intentional.

### Choosing the proxy IP in Docker Compose

Pinning the proxy's IP with `ipv4_address` (as in the README's compose file) is the simplest way to
keep `TRUSTED_PROXY` stable across container restarts. Two guidelines when picking the value:

- **Pick an IP in the lower half of the subnet** (e.g. `172.20.0.2` in `172.20.0.0/24`). The
  compose file's `ip_range: 172.20.0.128/25` entry restricts Docker's auto-assigned addresses to
  the upper half, so no container joining the network without a fixed IP can accidentally receive
  the proxy's address and silently become a trusted proxy.
- **Avoid the gateway address** (typically the first address in the subnet, `172.20.0.1`) — it is
  reachable from the Docker host itself, as described above.

### Using an existing Docker network

If Caddy already runs on a Docker network you manage elsewhere, do not copy the README's network
definition as-is. Attach PulseWeaver to that existing network and mark it as external:

```yaml
services:
  pulseweaver:
    networks:
      - proxy

networks:
  proxy:
    external: true
    name: your-existing-caddy-network
```

`external: true` only tells Compose not to create the network. The security requirement is unchanged:
Caddy must have a stable IP on that same network, and `TRUSTED_PROXY` must be set to that exact IP.
If your existing Caddy service currently gets a dynamic address, update its network settings with an
`ipv4_address` and recreate the container so Docker applies the change.

### Defense-in-depth against proxy IP registration

The proxy IP is protected at two independent layers:

1. **Middleware** — `X-Real-IP` is only read when the direct peer exactly matches `TRUSTED_PROXY`.
   Any other source's `X-Real-IP` header is ignored and a warning is logged.
2. **Address registry** — PulseWeaver refuses to register the `TRUSTED_PROXY` IP as a device
   address, even if it is explicitly submitted via the API or the heartbeat body. This means that
   even in a misconfigured deployment where the proxy IP ends up as the apparent client IP, it
   cannot enter the IP registry and trigger a universal pass.

### Direct-access deployments (no proxy)

If your devices connect directly to PulseWeaver without a proxy in between, leave `TRUSTED_PROXY`
unset. PulseWeaver will use each connection's source IP directly and will never read `X-Real-IP`
from any source. The startup warning can be safely ignored in this case.
