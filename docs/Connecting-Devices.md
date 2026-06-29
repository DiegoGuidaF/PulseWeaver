# Connecting devices

A device "connects" to PulseWeaver by keeping its current IP address registered.
It does that with a **heartbeat**: a periodic `POST /api/v1/heartbeat` carrying
the device's API key. PulseWeaver reads the request's source IP and registers it
as one of the device's addresses. As long as heartbeats keep arriving, that
address stays active; when they stop, address rules (below) retire it.

This page covers how to keep a device heartbeating — from the dedicated client to
a one-line `curl` on a cron timer — and the settings that make roaming behave.

> A heartbeat is just an authenticated HTTP POST. The clients below automate it,
> but nothing about them is privileged — anything that can send a POST on a timer
> works.

---

## How a device gets its key: pairing

A device has to exist in PulseWeaver before it can heartbeat, and it needs an API
key to authenticate. The easy path is **device pairing**:

1. In the UI, create the device (**Devices → New device**), then create a
   **pairing** for it — set the heartbeat server URL, interval, and an expiry.
2. PulseWeaver generates a single-use **pairing code** (shown as a QR or a
   copyable string).
3. The user pastes/scans the code into
   [PulseWeaver Companion](https://github.com/DiegoGuidaF/pulseweaver-heartbeat-client)
   ([downloads](https://github.com/DiegoGuidaF/pulseweaver-heartbeat-client/releases)).
   The client claims the code, receives the device's configuration **and a freshly
   generated API key**, and starts heartbeating — no manual URL/key entry.

The API key is created at claim time and returned exactly once; PulseWeaver only
ever stores its hash. See [device pairing](../README.md#device-pairing) in the
README for the full flow, and the
[PulseWeaver Companion docs](https://github.com/DiegoGuidaF/pulseweaver-heartbeat-client/blob/main/docs/app.md#device-pairing)
for the client side.

You can also skip pairing and configure a client manually with the server URL and
the device's API key (regenerate it from **Devices → the device → Settings**).

---

## The endpoints must be reachable without the gate

Heartbeat and pairing happen *before* a device has a registered IP, so they must
**not** sit behind the forward-auth gate — otherwise a device on a new network
could never register that network's IP. Two endpoints need to be publicly
reachable:

| Endpoint | Credential |
|---|---|
| `POST /api/v1/heartbeat` | Device API key (`X-API-Key` header) |
| `POST /api/v1/device-pair` | One-time pairing code (in the body) |

The recommended pattern exposes only these two on a dedicated device domain and
404s everything else, keeping the admin UI off the public internet. The full
configuration — including the required `X-Real-IP` directive — is in the
[Caddy setup guide](Caddy-Setup.md#step-1--device-endpoints-public-domain).

---

## Choosing a client

| Client | Best for |
|---|---|
| [**PulseWeaver Companion app**](https://github.com/DiegoGuidaF/pulseweaver-heartbeat-client) ([downloads](https://github.com/DiegoGuidaF/pulseweaver-heartbeat-client/releases)) | Android and desktop (Linux/macOS/Windows). Background scheduling, network-awareness, system tray, QR pairing. The default choice for Android phones and laptops. |
| [**Docker container**](https://github.com/DiegoGuidaF/pulseweaver-heartbeat-client/blob/main/docs/docker.md) | A server or NAS already running Docker. A tiny container that runs the `curl` heartbeat on a timer — set the URL, key, and interval and it looks after itself. |
| **systemd timer** (below) | Headless Linux servers. Zero dependencies beyond `curl`. |
| **launchd agent** (below) | Headless macOS servers. |
| **Tasker / any HTTP scheduler** | Android DIY, or any tool that can POST on a timer / network-change event. The heartbeat is just `curl -X POST -H "X-API-Key: …" <url>` — see the systemd timer and launchd agent recipes below. |
| **Manual** | Devices with a stable IP — add the address by hand in the UI, no heartbeat needed (below). |

> [!NOTE]
> There is no dedicated iOS client yet. The heartbeat is a standard HTTP POST, so an iOS setup needs a Shortcuts
> automation or another tool that can POST on a schedule/network change, using the same URL and `X-API-Key` header.

### Android reliability notes

When PulseWeaver Companion prompts you to disable battery optimization, do it. Android Doze/App Standby can otherwise
delay background heartbeats for hours, long enough for the device's address lease to expire and access to become
intermittent.

### Linux — systemd timer

A **timer + oneshot service** is the idiomatic way to schedule this on Linux.

`/etc/systemd/system/pulseweaver-heartbeat.service`

```ini
[Unit]
Description=PulseWeaver heartbeat ping
Wants=network-online.target
After=network-online.target

[Service]
Type=oneshot
ExecStart=curl -sf -X POST -H "X-API-Key: wdk_YOUR_KEY_HERE" https://pw-device.example.com/api/v1/heartbeat
```

`/etc/systemd/system/pulseweaver-heartbeat.timer`

```ini
[Unit]
Description=Send PulseWeaver heartbeat periodically

[Timer]
OnBootSec=30s
OnUnitActiveSec=30min
AccuracySec=30s

[Install]
WantedBy=timers.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now pulseweaver-heartbeat.timer
systemctl list-timers pulseweaver-heartbeat.timer   # verify it's scheduled
```

> **Security note:** the example inlines the API key for brevity. For production,
> use systemd's credential management (`LoadCredential=` /
> `LoadCredentialEncrypted=`, systemd ≥ 247) to inject it at runtime via
> `$CREDENTIALS_DIRECTORY` — see the [systemd credentials docs](https://systemd.io/CREDENTIALS/).

### macOS — launchd agent

`~/Library/LaunchAgents/com.pulseweaver.heartbeat.plist`

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.pulseweaver.heartbeat</string>
    <key>ProgramArguments</key>
    <array>
        <string>curl</string>
        <string>-sf</string>
        <string>-X</string>
        <string>POST</string>
        <string>-H</string>
        <string>X-API-Key: wdk_YOUR_KEY_HERE</string>
        <string>https://pw-device.example.com/api/v1/heartbeat</string>
    </array>
    <key>StartInterval</key>
    <integer>1800</integer>
    <key>RunAtLoad</key>
    <true/>
    <key>StandardErrorPath</key>
    <string>/tmp/pulseweaver-heartbeat.err</string>
</dict>
</plist>
```

```bash
launchctl load ~/Library/LaunchAgents/com.pulseweaver.heartbeat.plist
launchctl list | grep pulseweaver   # verify it's loaded
```

> **Security note:** store the key in the macOS Keychain
> (`security find-generic-password` in a wrapper script) rather than inlining it.

### Manual / static-IP devices

A device with a stable IP (a home server, a desktop on a reserved DHCP lease)
needs no heartbeat at all:

1. Open the device in the UI → **Addresses** tab.
2. Switch to **Custom IP**, enter the address, and **Register**.
3. Enable/disable addresses as needed.

For these devices, consider **removing the API key entirely** once the address is
set (see below).

---

## Verifying a device is connected

Whatever client you use, a clean startup on the client side is **not** proof — the only
authoritative check is that the *server* is registering the heartbeats. PulseWeaver records every
address event in the **address history**, reachable two ways:

- **Address history** in the navigation panel — the full log across all devices, with filters.
- A device's own page — the same events scoped to that one device.

For a healthy client you should see a new entry roughly every heartbeat interval, carrying the
device's current IP. The **`Δ prev`** column shows the gap since that device's previous heartbeat and
is colour-coded against its address lease (TTL): it turns **yellow** as the gap approaches the lease
and **red** once it meets or exceeds it — an early warning to raise the heartbeat frequency or the
lease before access actually drops.

> 📸 _Screenshot needed: the Address history view showing several entries for one device, with the
> `Δ prev` column visible and at least one yellow/red (near- or over-TTL) gap._

If no new entries appear, the heartbeat isn't reaching the server — check that the client is running,
that the URL bypasses the forward-auth gate, and that the API key is valid.

---

## Recommended settings for roaming devices

Heartbeats only ever *add or refresh* addresses. Two per-device
[address rules](../README.md#key-concepts) keep the set of active addresses tight
so a device that moves networks doesn't leave stale IPs allowed:

- **Address lease** — a TTL after which an address auto-disables if no heartbeat
  refreshes it.
- **Max active addresses** — a cap; enabling a new address disables the oldest
  over the cap.

Disabling the max-active-addresses rule later does not re-enable addresses it already evicted; those addresses stay
disabled until they are manually enabled or refreshed by a new heartbeat from that network.

**A good default for phones and laptops: a ~1 hour lease with max 2 active
addresses.** This handles roaming cleanly — when you change networks the new IP
registers immediately, and the old one ages out within the hour (or is evicted
sooner once a third network pushes past the cap of 2). The overlap is brief and
bounded, and a missed heartbeat or two won't lock you out mid-session.

You *can* go tighter (a shorter lease for faster expiry), but mind the client's
real cadence: the Android app's minimum interval is **15 minutes**, and under
Android Doze the effective gap can stretch to **~30 minutes**. A lease shorter
than the worst-case interval will flap. Headless servers on a fixed network can
use a much shorter lease, or none at all.

> **Keep API keys only where they're needed.** A device's API key exists to
> authenticate heartbeats. If a device has a static IP and never heartbeats,
> remove its API key after registering the address — it's one less credential to
> leak, and one less device that can register an IP. Keep keys on roaming devices
> only.

---

## Related

- Exposing the endpoints safely: [Caddy setup guide](Caddy-Setup.md).
- What "active address" means for access decisions: [How It Works](How-It-Works.md).
- The address-lease lifecycle in depth: [How It Works](How-It-Works.md#address-lease-lifecycle).
