# Security Model

## What PulseWeaver is

An **IP gate**. It reduces the attack surface of your server by refusing connections from unknown IP addresses before
they reach any service. It is not:

- A user authentication system (no passwords, no sessions, no tokens for end users).
- A replacement for Authelia, authentik, Keycloak, or any identity provider.
- A guarantee of security on its own.

## When it works well

- Services that break when placed behind an SSO proxy (Home Assistant, many media servers, IoT dashboards, etc.).
- Homelab environments where the set of trusted networks is small and well-understood.
- Reducing the blast radius of unpatched CVEs: an attacker cannot even reach the service if their IP is not active.
- Travel and flexible remote access: heartbeat + address lease gives you zero-config access from wherever your device
  is.

## When it is not enough

- If you need to identify individual users (use app-level auth in addition).
- If clients are on ISP-level CGNAT (see [Shared-IP Model](Shared-IP-Model.md)).
- If an active network is compromised (VPN leak, shared Wi-Fi with a bad actor, etc.).
- As a substitute for TLS. Always use HTTPS.

## Pros and cons summary

**Pros**

- Drastically reduced attack surface — unknown IPs cannot even reach your services.
- Simple mental model: devices keep their address active, everything else is blocked.
- No changes to existing applications or their auth systems.
- Zero-config access from trusted locations via heartbeat + lease.
- The "whole-network" behaviour of NAT works in your favour for home, hotel, and friend's house scenarios.

**Cons / caveats**

- IP-based trust only — does not verify identity.
- CGNAT can expose your services to unrelated co-tenants if you are not careful.
- If an active network is compromised while the IP is active, the attacker gains access.
- Does not replace TLS or app-level authentication.
- PulseWeaver should be **one layer** in a broader defence-in-depth strategy, not the only one.
