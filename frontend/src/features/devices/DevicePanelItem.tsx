import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import { Box, Group, Text, ThemeIcon, UnstyledButton } from "@mantine/core";
import type { FleetDevice } from "@/lib/api";
import { DeviceState } from "@/lib/api";
import { resolveDeviceIcon } from "@/features/devices/deviceTypeConfig";
import { isInactiveState } from "@/features/devices/constants";

dayjs.extend(relativeTime);

function deviceStatusText(entry: FleetDevice): string {
  const ago = entry.last_seen_at ? ` · ${dayjs(entry.last_seen_at).fromNow()}` : "";

  if (entry.state === DeviceState.DISABLED) return `disabled${ago}`;
  if (entry.live_address_count > 0) return `${entry.live_address_count} live${ago}`;
  if (entry.state === DeviceState.STALE) return `stale${ago}`;
  return entry.last_seen_at ? `seen ${dayjs(entry.last_seen_at).fromNow()}` : "never seen";
}

interface Props {
  entry: FleetDevice;
  isSelected: boolean;
  onSelect: () => void;
}

/** One row of the owner workspace's device sidebar. */
export function DevicePanelItem({ entry, isSelected, onSelect }: Props) {
  const renderIcon = resolveDeviceIcon(entry.icon);
  const isMuted = isInactiveState(entry.state);
  const isLive = entry.live_address_count > 0;

  return (
    <UnstyledButton
      onClick={onSelect}
      style={{
        display: "block",
        width: "100%",
        borderRadius: 6,
        borderLeft: `3px solid ${isSelected ? "var(--mantine-color-orange-5)" : "transparent"}`,
        background: isSelected ? "var(--mantine-color-default-hover)" : undefined,
      }}
    >
      <Group px="sm" py={6} gap="sm" align="center" wrap="nowrap">
        <ThemeIcon variant="transparent" size="sm" c={isMuted ? "dimmed" : undefined}>
          {renderIcon({ size: 16 })}
        </ThemeIcon>
        <Box style={{ flex: 1, minWidth: 0 }}>
          <Text
            size="sm"
            c={isMuted ? "dimmed" : undefined}
            fw={isSelected ? 500 : undefined}
            truncate
          >
            {entry.name}
          </Text>
          <Text size="xs" c="dimmed" truncate>
            {deviceStatusText(entry)}
          </Text>
        </Box>
        {isLive ? (
          <Box w={8} h={8} bg="orange.4" style={{ borderRadius: "50%", flexShrink: 0 }} />
        ) : (
          <Box
            w={8}
            h={8}
            style={{
              borderRadius: "50%",
              border: "1.5px solid var(--mantine-color-default-border)",
              flexShrink: 0,
            }}
          />
        )}
      </Group>
    </UnstyledButton>
  );
}
