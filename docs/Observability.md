# Observability

PulseWeaver records every allow/deny decision it makes, so you can always answer "who reached what?" and "what is
getting blocked?" — at a glance on the dashboard, or per request in the access log.

## Access Logs

**Auditing → Access Logs** shows one entry per decision: timestamp, client IP, requested host, outcome, and — when
relevant — which device matched, which network policy matched, and where the IP is located.

Filter by any combination of client IP, outcome, deny reason, device, network policy, host, country, or continent.
Typical questions it answers:

- *What just got denied, and why?* Filter outcome = denied. The deny reason tells you whether the IP wasn't registered
  at all (`ip_not_registered`) or the user simply lacked a grant for that host (`host_not_allowed`).
- *What is this device actually reaching?* Filter by device.
- *Is anything from outside the country hitting my proxy?* Filter by country or continent.

Logging is designed to never slow down request handling: under extreme load PulseWeaver drops log entries rather than
delaying decisions, so treat the log as an operational audit trail, not a billing-grade record.

## Dashboard

The **Dashboard** aggregates the same decisions into charts for a time window you pick (default: last 24 hours):

- **Traffic over time** — allowed vs denied request volume; the chart granularity adapts to the window, from per-minute
  for short windows up to per-day for multi-week ones.
- **Per-service split** — which hosts get the traffic.
- **Top denied IPs** — the addresses getting blocked most, a quick scan for scanners and misconfigured clients.

Windows up to 24 hours are computed live from the raw log. Longer windows use hourly summaries, so the most recent
hour fills in as it completes.

## GeoIP

Client IPs are resolved to **country, continent, and network operator (ASN)** so the access log and dashboard can show
and filter by geography. It works out of the box:

- Uses the free [DB-IP](https://db-ip.com) databases, downloaded automatically and refreshed monthly.
- Wholly self-contained — lookups happen locally; no per-request calls to any external service.
- Fail-open: if the data isn't available yet (or the IP is private), entries simply have no geo fields. GeoIP never
  blocks or fails a request.

| Setting          | Default        | Effect                          |
|------------------|----------------|---------------------------------|
| `GEOIP_ENABLED`  | `true`         | Master switch.                  |
| `GEOIP_DATA_DIR` | `./data/geoip` | Where the databases are stored. |

## Data retention

Old observability data is pruned automatically once per day:

| Setting               | Default | Effect                                                             |
|-----------------------|---------|----------------------------------------------------------------------|
| `DATA_RETENTION_DAYS` | `30`    | Entries older than this are deleted. `0` disables pruning entirely. |

The same setting also prunes the device address history shown under **Auditing → IP Address Logs** — one knob covers
both. The hourly summaries behind the dashboard are kept, so long-window charts can still show traffic volumes older
than the cutoff — only the per-request detail is gone.

## Related

- How decisions are made: [How It Works](How-It-Works.md), [Host Access Control](Host-Access-Control.md),
  [Network Policies](Network-Policies.md).
- What the logs can and cannot tell you about *who* made a request: [Security Model](Security-Model.md).
