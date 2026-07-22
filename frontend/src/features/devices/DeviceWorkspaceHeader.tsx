import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import { Group, Stack, Text, ThemeIcon, Title } from "@mantine/core";
import type { FleetDevice } from "@/lib/api";
import { resolveDeviceIcon } from "@/features/devices/deviceTypeConfig";
import { DeviceStatusBadge } from "@/features/devices/DeviceStatusBadge";
import { RuleChips } from "@/features/devices/RuleChips";

dayjs.extend(relativeTime);

function formatCreatedAt(iso: string): string {
  return dayjs(iso).format("D MMM YYYY");
}

interface Props {
  device: FleetDevice;
}

/** Title row of the owner device workspace: identity, state and the metadata line. */
export function DeviceWorkspaceHeader({ device }: Props) {
  const renderDeviceIcon = resolveDeviceIcon(device.icon);

  return (
    <Stack gap={4}>
      <Group gap="xs" align="center">
        <ThemeIcon variant="transparent" size="md" c="dimmed">
          {renderDeviceIcon({ size: 22 })}
        </ThemeIcon>
        <Title order={1} size="h3">{device.name}</Title>
        <DeviceStatusBadge state={device.state} size="sm" />
        <RuleChips
          rules={device.rules}
          pairing={device.pairing}
          liveAddressCount={device.live_address_count}
          size="xs"
        />
      </Group>
      <Group gap={6} wrap="wrap">
        {device.live_address_count > 0 && (
          <Text size="xs" c="var(--pw-amber-text)">
            live · {device.live_address_count} IP{device.live_address_count !== 1 ? "s" : ""}
          </Text>
        )}
        {device.last_seen_at && (
          <>
            <Text size="xs" c="dimmed">·</Text>
            <Text size="xs" c="dimmed">seen {dayjs(device.last_seen_at).fromNow()}</Text>
          </>
        )}
        {device.api_key_prefix && (
          <>
            <Text size="xs" c="dimmed">·</Text>
            <Text size="xs" c="dimmed" ff="monospace">{device.api_key_prefix}…</Text>
          </>
        )}
        {device.created_at && (
          <>
            <Text size="xs" c="dimmed">·</Text>
            <Text size="xs" c="dimmed">created {formatCreatedAt(device.created_at)}</Text>
          </>
        )}
      </Group>
    </Stack>
  );
}
