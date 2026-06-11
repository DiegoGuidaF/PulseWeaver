# Network Policies

A network policy grants access to clients by **IP range** instead of by registered device: "anything on
`192.168.1.0/24` may reach these hosts." Use them for networks you trust as a whole — your home LAN, a VPN subnet, a
small office range — where registering every device individually would be busywork.

Network policies are the fallback mechanism: they are only consulted when the client IP is **not** a registered device
address. Registered devices are always governed by their user's allowlist (see
[Host Access Control](Host-Access-Control.md)).

## Creating a policy

Under **Access → Network Policies**, a policy is:

| Field               | Meaning                                                                                  |
|---------------------|------------------------------------------------------------------------------------------|
| **Name**            | A label for you, e.g. "Home LAN" or "WireGuard clients".                                 |
| **CIDR**            | The IP range it covers, IPv4 or IPv6, e.g. `192.168.1.0/24`.                             |
| **Host groups**     | Which [host groups](Host-Access-Control.md#concepts) clients in this range may reach.    |
| **Bypass host check** | If on, clients in the range may reach **any** host — use sparingly.                    |
| **Enabled**         | Disabled policies are ignored entirely.                                                  |

As with users, hosts are granted through host groups — define your groups first, then attach them to the policy.

## How overlapping ranges behave

When a client IP falls inside several policies, **the most specific range decides — alone**:

- Policies are checked narrowest prefix first (`/28` before `/24` before `/16`).
- The first range containing the IP gives the final answer. If that policy doesn't allow the requested host, the
  request is denied even if a broader policy would have allowed it. There is no fall-through.

This makes carve-outs easy to reason about: a narrow policy for one corner of your network overrides whatever the
surrounding range says — in either direction.

## Guard against over-broad ranges

To keep a typo from allowing half the internet, PulseWeaver refuses ranges at the scale of an entire network operator
and warns about merely large ones:

| Range size                          | Behaviour                                  |
|-------------------------------------|--------------------------------------------|
| IPv4 `/8` or wider, IPv6 `/32` or wider | **Rejected** — cannot be saved.          |
| IPv4 `/9`–`/16`, IPv6 `/33`–`/47`   | Saved, but flagged with a warning.         |
| Narrower                            | No warning.                                |

There is no override for the rejected band — ranges that broad have no legitimate use in front of a home or small-team
proxy.

## Good to know

- Changes take effect automatically within a few seconds — no restart needed.
- Denied requests look exactly like every other denial: a uniform HTTP 403 that reveals nothing about why.
- Test a policy with **Auditing → Access Verification** ("would IP X reach host Y?") before relying on it.
- A policy grants access to **everything** in its range — anyone joining that network is covered. For untrusted or
  shared networks, prefer registered devices per user.
