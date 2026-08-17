import type { BadgeProps } from "@mantine/core";
import { AddressEventSource, DeviceState } from "@/lib/api";

interface DeviceStateBadge {
  color: string;
  label: string;
  variant?: BadgeProps["variant"];
  tooltip?: string;
}

/**
 * Display config for the non-healthy device states. `healthy` is intentionally
 * absent — liveness is already conveyed by the live-IP pips, so a badge appears
 * exactly when a device needs a second look. Pairing state (pending/expired) is
 * conveyed by the RuleChips pairing badge from live pairing data, not here.
 */
export const DEVICE_STATE_BADGE: Partial<Record<DeviceState, DeviceStateBadge>> = {
  [DeviceState.STALE]: { color: "gray", label: "Stale", tooltip: "No live IPs" },
  [DeviceState.DISABLED]: { color: "gray", label: "Disabled", variant: "filled", tooltip: "Device is disabled and will not receive new IPs" },
};

/** States where the device is not currently reachable, so it reads as muted. */
export function isInactiveState(state: DeviceState): boolean {
  return state === DeviceState.STALE || state === DeviceState.DISABLED;
}

/** Compact source label for the address/history inline "updated_at · source" rows. */
export const ADDRESS_SOURCE_LABELS: Record<AddressEventSource, string> = {
  [AddressEventSource.HEARTBEAT]: "heartbeat",
  [AddressEventSource.WEB_UI]: "web UI",
  [AddressEventSource.EXPIRY]: "expired",
  [AddressEventSource.LIMIT_EXCEEDED]: "evicted",
};
