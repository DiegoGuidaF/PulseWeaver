# The Shared-IP Model

PulseWeaver gates by IP, not by individual identity: whoever appears to share your IP shares your access. What that
access *covers* is bounded by your [host access grants](Host-Access-Control.md) — an activated IP reaches the hosts its
user was granted, not everything behind the proxy.

The key word is **appears**. What matters is the IP *PulseWeaver actually sees* for a request, which depends on where
the client sits relative to the proxy:

- **On the same LAN as the PulseWeaver server, each device is seen by its own internal IP** (e.g. `192.168.1.231`).
  That traffic never leaves the local network, so there's no shared public IP and no sharing of access — your phone and
  your partner's phone at home are distinct IPs to PulseWeaver. (This assumes a typical homelab where the proxy runs on
  that LAN.)
- **Reaching the proxy from outside a NAT, every device behind that NAT shares its one public IP.** This is where
  sharing actually happens. If you heartbeat from a coffee shop, a hotel, or a friend's house, that network's *public*
  IP is what PulseWeaver registers — and everyone else on it can reach your granted hosts until the address ages out
  (an [address lease](Connecting-Devices.md#recommended-settings-for-roaming-devices) deactivates it shortly after you
  leave). For a controlled network this is the intended convenience; on an untrusted one, keep it short-lived.
- **ISP-level CGNAT:** some ISPs put hundreds or thousands of unrelated subscribers behind one public IP. Activating a
  CGNAT IP means those co-tenants *could* reach your granted hosts too. In practice this is a modest, not catastrophic,
  increase: it's still a small slice of the internet, and an attacker would have to already be one of those co-tenants,
  find the service, and get past whatever auth it has. PulseWeaver's job here is to **shrink the attack surface**, not
  eliminate it — so on CGNAT, make sure each exposed service keeps its own authentication. (Check with your ISP whether
  you're on CGNAT.)

## Several users behind one IP: the strictest wins

When one IP genuinely carries devices belonging to *different* PulseWeaver users — which, per above, means traffic
arriving through a shared NAT from outside, not devices on the proxy's own LAN — PulseWeaver cannot tell which person is
behind a given request, so it refuses to guess: that IP may only reach the hosts that **every** user on it is allowed to
reach. One restricted user narrows the whole IP — and a user with **bypass host check** enabled only keeps that bypass
on an IP where everyone else has it too.

PulseWeaver surfaces these situations for you rather than leaving them to be discovered by a confused 403: the
**dashboard** flags shared IPs (an IP claimed by multiple users), and **Auditing → Access Verification** shows, for a
given IP and host, exactly which grant or intersection produced the decision. If a request unexpectedly gets a 403,
that's the place to look — the usual cause is another user's device sharing the IP with a narrower grant.

## IPv6 changes the picture

With IPv6, devices typically receive globally unique, non-NATted addresses. This means:

- Each device has its own public IP rather than sharing one with an entire household.
- The "shared NAT" behaviour described above does not apply — an IPv6 heartbeat will activate a specific device's
  address, not a whole network.
- This makes the IP gate more precise and, in principle, more secure under IPv6.

However, **IPv6 support should be considered young**. The application handles IPv6 throughout — the address model
stores it and normalises it at validation time — but IPv6 brings edge cases that are inherently hard to cover until a
range of real deployments exercise them: prefix delegation, privacy/temporary addresses that rotate, and dual-stack
setups where a device flips between v4 and v6. Treat IPv6 as working, and expect it to harden as more people run it and
report what they hit.
