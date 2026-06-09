import { useNavigate } from "react-router-dom";
import { buildRoute } from "@/lib/routes";
import { Box, Group, ThemeIcon, Tooltip, UnstyledButton, Text } from "@mantine/core";
import { IconChevronRight } from "@tabler/icons-react";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import type { DeviceListEntry } from "@/lib/api";
import { resolveDeviceIcon } from "@/features/devices/deviceTypeConfig";
import { DeviceState } from "@/lib/api";
import { RuleChips } from "@/features/devices/RuleChips";
import { DeviceStatusBadge } from "@/features/devices/DeviceStatusBadge";
import { isInactiveState } from "@/features/devices/constants";

dayjs.extend(relativeTime);

const MAX_PIPS = 3;

function LivePips({ count }: { count: number }) {
  if (count === 0) return null;
  const pips = Math.min(count, MAX_PIPS);
  const overflow = count > MAX_PIPS ? count - MAX_PIPS : 0;
  return (
    <Tooltip label={`${count} live IP address${count !== 1 ? "es" : ""}`} withArrow>
      <Group gap={4} wrap="nowrap">
        {Array.from({ length: pips }).map((_, i) => (
          <Box key={i} w={8} h={8} bg="orange.4" style={{ borderRadius: "50%", flexShrink: 0 }} />
        ))}
        {overflow > 0 && (
          <Text size="xs" c="dimmed">+{overflow}</Text>
        )}
      </Group>
    </Tooltip>
  );
}

function getRowContainerStyle(state: DeviceState): React.CSSProperties {
  if (state === DeviceState.DISABLED) {
    return {
      border: "1px dashed var(--mantine-color-default-border)",
      borderRadius: 6,
      background: "light-dark(color-mix(in srgb, var(--mantine-color-gray-5) 11%, transparent), color-mix(in srgb, var(--mantine-color-dark-3) 20%, transparent))",
    };
  }
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
  const renderIcon = resolveDeviceIcon(entry.icon);
  const lastSeenText = entry.last_seen_at
    ? dayjs(entry.last_seen_at).fromNow()
    : "never seen";

  const isMuted = isInactiveState(entry.state);

  return (
    <Box style={getRowContainerStyle(entry.state)}>
      <UnstyledButton
        w="100%"
        onClick={() => navigate(`${buildRoute.userDevices(ownerId)}?device=${entry.id}`)}
        style={{ display: "block" }}
      >
        <Group gap="sm" align="center" wrap="nowrap" px="xs" py={8}>
          <ThemeIcon variant="transparent" size="md" c={isMuted ? "dimmed" : undefined}>
            {renderIcon({ size: 22 })}
          </ThemeIcon>

          <Box style={{ flex: 1, minWidth: 0 }}>
            <Group gap={6} wrap="nowrap">
              <Text fw={500} size="sm" c={isMuted ? "dimmed" : undefined} truncate>
                {entry.name}
              </Text>
              <DeviceStatusBadge state={entry.state} />
            </Group>
            <Group gap={6} wrap="nowrap">
              <Text size="xs" c="dimmed" truncate>{lastSeenText}</Text>
              {entry.api_key_prefix && (
                <Text size="xs" c="dimmed" ff="monospace">{entry.api_key_prefix}</Text>
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
