import React from "react";
import { useNavigate } from "react-router-dom";
import { Box, Group, Text, ThemeIcon, UnstyledButton } from "@mantine/core";
import { IconChevronRight, IconDevices } from "@tabler/icons-react";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import type { DeviceListEntry } from "@/lib/api";
import { ICON_PICKER_OPTIONS } from "@/features/devices/deviceTypeConfig";
import { DeviceState } from "@/lib/api";
import { RuleChips } from "@/features/devices/RuleChips";

dayjs.extend(relativeTime);

const ICON_MAP = new Map(ICON_PICKER_OPTIONS.map(({ name, icon }) => [name, icon]));

function getDeviceListIcon(icon?: string | null) {
  if (icon) {
    const resolved = ICON_MAP.get(icon);
    if (resolved) return resolved;
  }
  return IconDevices;
}

const MAX_PIPS = 3;

function LivePips({ count }: { count: number }) {
  if (count === 0) return null;
  const pips = Math.min(count, MAX_PIPS);
  const overflow = count > MAX_PIPS ? count - MAX_PIPS : 0;
  return (
    <Group gap={4} wrap="nowrap">
      {Array.from({ length: pips }).map((_, i) => (
        <Box key={i} w={8} h={8} bg="orange.4" style={{ borderRadius: "50%", flexShrink: 0 }} />
      ))}
      {overflow > 0 && (
        <Text size="xs" c="dimmed">+{overflow}</Text>
      )}
    </Group>
  );
}

function getRowContainerStyle(state: DeviceState): React.CSSProperties {
  if (state === DeviceState.STALE) {
    return {
      border: "1px dashed var(--mantine-color-default-border)",
      borderRadius: 6,
    };
  }
  if (state === DeviceState.PENDING_CLAIM) {
    return {
      border: "1px dashed var(--mantine-color-indigo-5)",
      borderRadius: 6,
    };
  }
  return {
    border: "1px solid var(--mantine-color-default-border)",
    borderRadius: 6,
  };
}

interface Props {
  entry: DeviceListEntry;
  ownerId: number;
}

export function DeviceRow({ entry, ownerId }: Props) {
  const navigate = useNavigate();
  const Icon = getDeviceListIcon(entry.icon);
  const lastSeenText = entry.last_seen_at
    ? dayjs(entry.last_seen_at).fromNow()
    : "never seen";

  const isStale = entry.state === DeviceState.STALE;

  return (
    <Box style={getRowContainerStyle(entry.state)}>
      <UnstyledButton
        w="100%"
        onClick={() => navigate(`/user-devices/${ownerId}?device=${entry.id}`)}
        style={{ display: "block" }}
      >
        <Group gap="sm" align="center" wrap="nowrap" px="xs" py={8}>
          <ThemeIcon variant="transparent" size="md" c={isStale ? "dimmed" : undefined}>
            {React.createElement(Icon, { size: 18, stroke: 1.5 })}
          </ThemeIcon>

          <Box style={{ flex: 1, minWidth: 0 }}>
            <Text fw={500} size="sm" c={isStale ? "dimmed" : undefined} truncate>
              {entry.name}
            </Text>
            <Group gap={6} wrap="nowrap">
              <Text size="xs" c="dimmed" truncate>{lastSeenText}</Text>
              {entry.key_prefix && (
                <Text size="xs" c="dimmed" ff="monospace">{entry.key_prefix}</Text>
              )}
            </Group>
          </Box>

          <Group gap={4} wrap="nowrap">
            <RuleChips entry={entry} />
          </Group>

          <LivePips count={entry.live_address_count} />

          <IconChevronRight size={14} color="var(--mantine-color-dimmed)" />
        </Group>
      </UnstyledButton>
    </Box>
  );
}
