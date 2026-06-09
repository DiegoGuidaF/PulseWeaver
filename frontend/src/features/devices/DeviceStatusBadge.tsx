import { Badge, Tooltip } from "@mantine/core";
import type { BadgeProps } from "@mantine/core";
import type { DeviceState } from "@/lib/api";
import { DEVICE_STATE_BADGE } from "@/features/devices/constants";

interface DeviceStatusBadgeProps {
  state: DeviceState;
  size?: BadgeProps["size"];
}

/** Renders the device's lifecycle state, or nothing for the healthy/live state. */
export function DeviceStatusBadge({ state, size = "xs" }: DeviceStatusBadgeProps) {
  const cfg = DEVICE_STATE_BADGE[state];
  if (!cfg) return null;
  const badge = (
    <Badge size={size} color={cfg.color} variant={cfg.variant ?? "light"}>
      {cfg.label}
    </Badge>
  );
  return cfg.tooltip ? (
    <Tooltip label={cfg.tooltip} withArrow>
      {badge}
    </Tooltip>
  ) : badge;
}
