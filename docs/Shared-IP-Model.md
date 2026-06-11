# The Shared-IP Model

PulseWeaver gates by IP, not by individual identity: whoever shares your IP shares your access. What that access
*covers* is bounded by your [host access grants](Host-Access-Control.md) — an activated IP reaches the hosts its user
was granted, not everything behind the proxy. Concretely:

- **Multiple devices behind the same NAT (e.g. a home router) all share one public IP.** If your phone's heartbeat
  activates your home IP, everyone at home can reach the hosts *you* can reach. This is usually the intended
  behaviour.
- **Hotel Wi-Fi / friend's house:** as soon as your phone sends a heartbeat from a new network, that network's public IP
  is activated. Everyone else on that network can reach your allowed hosts during your stay. If you have an address
  lease configured, the IP is automatically deactivated shortly after you leave — without any manual action.
- **ISP-level CGNAT:** some ISPs share a single public IP across hundreds or thousands of unrelated subscribers.
  Activating a CGNAT IP means all those co-tenants inherit your access — your host grants shrink the blast radius, but
  they don't remove it. **If your device is behind CGNAT (check with your ISP), treat heartbeat-activated IPs from
  that network with caution.**

In summary: private NAT (home router, hotel, friend's house) — controlled network, great fit. ISP CGNAT — allows
multiple unknown co-tenants, higher attack surface.

## Several users behind one IP: the strictest wins

In a household, one IP usually carries devices belonging to *different* PulseWeaver users with different grants.
PulseWeaver cannot tell which person is behind a given request, so it refuses to guess: a shared IP may only reach the
hosts that **every** user on that IP is allowed to reach. One restricted user narrows the whole IP — and a user whose
allowlist check is bypassed (admins, by default) only keeps that bypass on an IP where everyone else has it too.

If a device unexpectedly gets a 403 at home but works elsewhere, this intersection is the usual cause: check whether
another user's device shares the IP and has a narrower grant.

## IPv6 changes the picture

With IPv6, devices typically receive globally unique, non-NATted addresses. This means:

- Each device has its own public IP rather than sharing one with an entire household.
- The "shared NAT" behaviour described above does not apply — an IPv6 heartbeat will activate a specific device's
  address, not a whole network.
- This makes the IP gate more precise and, in principle, more secure under IPv6.

However, **IPv6 support in PulseWeaver is not yet thoroughly tested**. The application handles IPv6 addresses in the
address model and normalises them at validation time, but real-world edge cases (prefix delegation, temporary addresses,
dual-stack setups) have not been fully exercised. Treat IPv6 as working but experimental until you have validated it in
your own setup.
