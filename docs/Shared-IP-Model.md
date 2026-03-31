# The Shared-IP Model

PulseWeaver gates by IP, not by individual identity. This means:

- **Multiple devices behind the same NAT (e.g. a home router) all share one public IP.** If your phone's heartbeat
  activates your home IP, everyone at home can access your services. This is usually the intended behaviour.
- **Hotel Wi-Fi / friend's house:** as soon as your phone sends a heartbeat from a new network, that network's public IP
  is activated. Everyone else on that network can also access your services during your stay. If you have an address
  lease configured, the IP is automatically deactivated shortly after you leave — without any manual action.
- **ISP-level CGNAT:** some ISPs share a single public IP across hundreds or thousands of unrelated subscribers.
  Understand that activating a CGNAT IP means allowing all those co-tenants to reach your services. **If your device
  is behind CGNAT (check with your ISP), treat heartbeat-activated IPs from that network with caution.**

In summary: private NAT (home router, hotel, friend's house) — controlled network, great fit. ISP CGNAT — allows
multiple unknown co-tenants, higher attack surface.

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
