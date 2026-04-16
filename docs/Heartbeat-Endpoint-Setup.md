# Heartbeat Endpoint Setup

The heartbeat endpoint (`POST /api/v1/heartbeat`) must be **reachable from your devices without going through the
forward-auth gate** — if it were gated behind the IP check, a device with a new IP could never activate that IP in the
first place.

If you use [device provisioning](../README.md#device-provisioning), the registration endpoint
(`POST /api/v1/register`) also needs to be publicly reachable. The examples below show how to expose both.

## Exposing device endpoints with Caddy

The recommended approach exposes only the two device-facing endpoints on a single public domain — everything else
returns 404:

```caddy
device.example.com {
    @device-endpoints path /api/v1/heartbeat /api/v1/register
    handle @device-endpoints {
        reverse_proxy pulseweaver:8080
    }
    respond 404
}
```

Alternatively, create a dedicated site that routes only to the heartbeat endpoint (if you don't use provisioning or
expose the registration endpoint separately):

```caddy
device-heartbeat.example.com {
    # Rewrite all requests to the heartbeat path
    rewrite * /api/v1/heartbeat

    # Proxy directly to PulseWeaver, bypassing forward_auth
    reverse_proxy pulseweaver:8080
}
```

Authentication is handled by the device's `X-API-Key`, which PulseWeaver validates for every heartbeat request. No
additional auth layer is needed.

**Optional extra obscurity:** If you want the endpoint to be harder to discover, you can add a random path segment to
the public URL and rewrite it. This should be treated as a noise-reduction measure, not as security, the real security
is provided by the API key in the header.

```caddy
device-heartbeat.example.com {
    handle /your-random-secret {
        rewrite * /api/v1/heartbeat
        reverse_proxy pulseweaver:8080
    }

    handle {
        respond "Not Found" 404
    }
}
```

The API key provides the real security. The secret path is only a small discovery barrier, but it helps limit casual
probing when the domain becomes visible through public certificate logs.


<!--
TODO: This sections below don't quite fit here. Probably the Android Tasker should move to the heartbeat-client repository.
There should be a separate rules section maybe? Also, make clear the recommendations regarding devices with dynamic IPs 
(there a brief mention of this in the heartbeat client documentation)
-->

## Android (Tasker)

Create a Tasker profile that triggers on:

- A periodic timer (e.g. every 4 minutes, take into account the address lease TTL for that device)
- WiFi connected / disconnected
- Mobile data connected / disconnected

The action is an HTTP Request (or a Shell action with `curl`):

```bash
curl -s -X POST https://device-heartbeat.example.com \
  -H "X-API-Key: your-device-api-key"
```

This keeps your phone's current IP active at all times as you move between networks.

> **Tip:** For a dedicated heartbeat app with background scheduling, network-awareness, and system tray support,
> see the [PulseWeaver Heartbeat Client](https://github.com/DiegoGuidaF/pulseweaver-heartbeat-client).

## Address lease recommendations

Set your device's address lease TTL to **5 minutes** (a bit more than the heartbeat interval). This means:

- After each heartbeat, the address is active for 5 more minutes.
- If no heartbeat arrives (e.g. you turned off your phone), the address is automatically deactivated after 5 minutes.
- Old addresses (previous network) expire shortly after you change networks.
- The overlap window — the time two addresses are simultaneously active — is at most one heartbeat interval.

You can tune the TTL shorter (e.g. 2 minutes) for tighter expiry, or longer if you prefer more tolerance for missed
heartbeats.

## Laptop / static device

For a device with a mostly stable IP (e.g. a home laptop), manual management is usually sufficient:

1. Open the PulseWeaver UI and navigate to your device.
2. On the **Addresses** tab, switch to **Custom IP**, enter the address, and click **Register**.
3. Enable or disable addresses as needed from the assigned addresses list.

No heartbeat automation required unless you want it.
