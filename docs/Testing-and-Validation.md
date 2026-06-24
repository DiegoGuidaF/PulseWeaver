# Testing & Validation

Beyond the unit and integration suites in this repository, PulseWeaver is validated
**externally against a production-like deployment** — the same release container fronted
by a reverse proxy over HTTPS, exercised end to end so that container networking,
middleware wiring, rate limiters, cookie flags, and browser security policies are all in
play. The goal is a posture that is verified rather than asserted, which matters for a
component that sits on the authentication path.

These checks are runnable and repeatable, not one-off audits.

## Performance at a glance

The critical path is `GET /api/policy-engine/verify-ip` — the forward-auth gate the
reverse proxy calls for every protected request. It is served from an in-memory cache, so
per-request authorization is negligible overhead, and the whole service runs in a small
memory footprint. The figures below are measured under a **sustained ~50 requests/second**
— far heavier than a typical self-hosted deployment, where the gate sees well under one
request/second on average — with a realistic ~1% deny mix.

| Metric | Value | Notes |
|--------|-------|-------|
| `verify-ip` latency — via reverse proxy (p50 / p99) | ~1.6 ms / ~5 ms | End-to-end: Caddy + PulseWeaver |
| `verify-ip` latency — PulseWeaver only (p50 / p99) | ~0.3 ms / ~0.5 ms | The policy decision itself, no proxy |
| CPU under that load | ~1.5% of one core | Application process |
| Memory usage | ~24 MB | Resident set; well under the 256 MB container limit |
| Rate limiting | None on this path, by design | Constant-time token check over an in-memory cache |

The two latency rows isolate the proxy. PulseWeaver's own decision is **sub-millisecond**,
measured directly against the application listener; with Caddy in front the figure a real
client sees is ~1.6 ms. Measuring an identical request with and without the proxy, Caddy
accounts for roughly 1 ms of the total — TLS termination, HTTP forwarding, and an extra
network hop (the TLS handshake itself is excluded, since a proxy amortizes it with
keep-alive). CPU and memory are the application process alone: CPU is sampled from the
binary via `pprof`, and memory is its resident footprint.

These are order-of-magnitude figures on commodity hardware — absolute numbers depend on
the host — but they reflect the design intent: authorization on the proxy path should be
fast and cheap, and the service should stay lean. The ~1.5% CPU at this rate is the
headline. The path is **deliberately not rate-limited**: the decision is a constant-time
bearer-token check over an in-memory cache, so it absorbs load rather than rejecting it.
Stress testing has driven it to roughly **15,000 requests/second** sustained with no
connection failures — far past any plausible self-hosted load — so that headroom is
measured, not just inferred.

---

## Security

Security is validated as a structured pentest against the live stack, organised by
**actor** — what an unauthenticated client, an admin session, and a device API key can
each attempt — plus a container/infrastructure pass. The primary target is the
`verify-ip` forward-auth gate, since a bypass there means an unauthorized client reaches
a protected service.

Practices applied:

- **Forward-auth gate** — IP/host spoofing, CIDR-matching edge cases, bearer-token
  enumeration, and timing-oracle probes, confirming the gate fails closed.
- **Session & browser surface** — cookie flags, CSRF, stored XSS, and session
  fixation/invalidation, driven through a real browser.
- **Device API keys** — scope isolation, cache-poisoning attempts, and per-key rate
  limits.
- **Input handling** — SQL-injection and malformed-request probes across the API.
- **Container & image** — non-root/unprivileged runtime, resource limits, an unpublished
  application port, security headers, and automated image/Dockerfile vulnerability and
  misconfiguration scanning.
- **Whole surface** — an automated active scan (OWASP ZAP) across the API.

Tooling includes Playwright (browser-driven CSRF/XSS), Trivy (image and configuration
scanning), a load/latency driver, and OWASP ZAP. The audit is treated as a feedback
loop: findings are remediated and re-verified, and any accepted residual risk is
documented rather than hidden.

---

## Frontend accessibility & quality

The frontend is audited against the **production build** over HTTPS — development-server
performance is unrepresentative, so the optimized embedded SPA is the target. Several
complementary instruments are used:

- **Structural accessibility** — an axe-core sweep against WCAG 2.1 AA across every page.
- **Keyboard focus contracts** — focus-in, focus-trap, and focus-return behaviour for
  modals and drawers.
- **Performance & accessibility scores** — Lighthouse, run on the production build.
- **Responsive layout** — a viewport sweep that catches horizontal overflow and
  undersized tap targets at phone and tablet widths.

These tools raise the floor; they do not replace manual review. Lighthouse's
accessibility score in particular is only a subset of WCAG, so axe-core carries the
fuller automatable sweep, and the viewport sweep covers the "doesn't scale at narrow
widths" cases neither of the others detects.

---

## Backend performance

Performance work uses two complementary instruments:

- **Microbenchmarks** on the hot paths — chiefly the in-memory forward-auth decision and
  the access-log write path — driven through the real production code path and gated for
  regressions with `benchstat`.
- **Profiling** (`pprof` and execution traces) for discovery under realistic load. The
  profiling surface is compiled in only under a dedicated build tag and bound to an
  in-process loopback listener, so a release binary contains none of it — there is
  nothing to misconfigure in production.

The critical path, `GET /api/policy-engine/verify-ip`, is benchmarked and profiled most
closely, since the reverse proxy calls it for every protected request; its headline
numbers are in [Performance at a glance](#performance-at-a-glance) above. Memory is
tracked alongside — the live heap from the same profiles, and the resident footprint of
the running container — confirming the service stays well within its container limit.
