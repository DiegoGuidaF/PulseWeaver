# Security Model

## What PulseWeaver is

An **IP gate with per-user host authorization**. It reduces the attack surface of your server by refusing connections
from unknown IP addresses before they reach any service — and for known IPs, it only allows the hosts that the device's
user has been [granted](Host-Access-Control.md). It is not:

- A user authentication system (no passwords, no sessions, no tokens for end users — who a request belongs to is
  inferred from its IP, never verified).
- A replacement for Authelia, authentik, Keycloak, or any identity provider.
- A complete security solution by itself — it's designed to be **one layer** in defence-in-depth, dramatically
  shrinking what's reachable so your other layers face far less traffic.

## When it works well

- Services that break when placed behind an SSO proxy (Home Assistant, many media servers, IoT dashboards, etc.).
- Homelab environments where the set of trusted networks is small and well-understood.
- Reducing the blast radius of unpatched CVEs: an attacker cannot even reach the service if their IP is not active.
- Travel and flexible remote access: heartbeat + address lease gives you zero-config access from wherever your device
  is.

## When it is not enough

- If you need to *verify* who a user is. PulseWeaver has per-user access grants, but the user behind a request is
  inferred from the client IP — on a shared IP it cannot tell people apart, so it only allows hosts that **every**
  user on that IP may reach (see [Shared-IP Model](Shared-IP-Model.md)). For real identity, keep app-level auth on the
  services themselves.
- On ISP-level CGNAT, the IP you activate is shared with unrelated subscribers, so they inherit reachability to your
  granted hosts. PulseWeaver still keeps the wider internet out — but it can't tell co-tenants apart, so a service
  exposed this way should keep its own authentication (see [Shared-IP Model](Shared-IP-Model.md)).
- If an active network is compromised (VPN leak, shared Wi-Fi with a bad actor, etc.), the attacker on that network
  gains whatever reachability the active IP has.

## Pros and cons summary

**Pros**

- Drastically reduced attack surface — unknown IPs cannot even reach your services.
- Deny-by-default, per-user host access: a known IP only reaches the services its user was granted, and a newly proxied
  service is unreachable until explicitly granted to someone.
- Simple mental model: devices keep their address active, users reach what they were granted, everything else is
  blocked.
- No changes to existing applications or their auth systems.
- Zero-config access from trusted locations via heartbeat + lease.
- The "whole-network" behaviour of NAT works in your favour for home, hotel, and friend's house scenarios.

**Cons / caveats**

- IP-based trust only — per-user grants exist, but identity is inferred from the IP, never verified.
- On CGNAT, an activated IP is shared with unrelated co-tenants who inherit reachability to your granted hosts.
- If an active network is compromised while the IP is active, the attacker on it gains access.
- Does not authenticate users — keep app-level auth on services where identity actually matters.
- PulseWeaver should be **one layer** in a broader defence-in-depth strategy, not the only one.
