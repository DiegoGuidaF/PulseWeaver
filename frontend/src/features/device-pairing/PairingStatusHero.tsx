import { Group, Stack, Text } from "@mantine/core";
import { DeviceState } from "@/lib/api";
import type { DeviceState as DeviceStateType } from "@/lib/api";
import { useDateFormatter } from "@/contexts/useDateTimePrefs";
import classes from "./PairingStatusHero.module.css";

interface Props {
  deviceState: DeviceStateType;
  /** When the companion app claimed the active link. */
  claimedAt: string;
}

/**
 * Hero for an already-linked device. Carries two orthogonal facts: that the device
 * is linked (the title) and whether the link is alive (the heartbeat dot — amber and
 * pulsing when healthy, quiet when stale or disabled).
 */
export function PairingStatusHero({ deviceState, claimedAt }: Props) {
  const formatDateTime = useDateFormatter();
  const isHealthy = deviceState === DeviceState.HEALTHY;

  const health =
    deviceState === DeviceState.DISABLED
      ? "Device disabled — not receiving heartbeats"
      : isHealthy
        ? "Heartbeat active"
        : "No recent heartbeat";

  return (
    <Group gap="sm" align="center" wrap="nowrap">
      <span className={`${classes.dot} ${isHealthy ? classes.alive : classes.quiet}`} />
      <Stack gap={2}>
        <Text fw={600}>Linked to companion app</Text>
        <Text size="xs" c="dimmed">
          {health} · claimed {formatDateTime(claimedAt)}
        </Text>
      </Stack>
    </Group>
  );
}
