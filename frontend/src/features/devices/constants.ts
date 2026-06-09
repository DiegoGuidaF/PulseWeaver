import type { BadgeProps } from "@mantine/core";
import { DeviceState } from "@/lib/api";

interface DeviceStateBadge {
  color: string;
  label: string;
  variant?: BadgeProps["variant"];
  tooltip?: string;
}

/**
 * Display config for the non-healthy device states. `healthy` is intentionally
 * absent — liveness is already conveyed by the live-IP pips, so a badge appears
 * exactly when a device needs a second look.
 */
export const DEVICE_STATE_BADGE: Partial<Record<DeviceState, DeviceStateBadge>> = {
  [DeviceState.STALE]: { color: "gray", label: "Stale", tooltip: "No live IPs" },
  [DeviceState.DISABLED]: { color: "gray", label: "Disabled", variant: "filled", tooltip: "Device is disabled and will not receive new IPs" },
  [DeviceState.PENDING_CLAIM]: { color: "indigo", label: "Pairing pending" },
  [DeviceState.EXPIRED_CLAIM]: { color: "red", label: "Code expired" },
};

/** States where the device is not currently reachable, so it reads as muted. */
export function isInactiveState(state: DeviceState): boolean {
  return state === DeviceState.STALE || state === DeviceState.DISABLED;
}
