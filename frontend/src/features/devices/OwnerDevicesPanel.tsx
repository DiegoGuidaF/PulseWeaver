import { useNavigate } from "react-router-dom";
import { buildRoute } from "@/lib/routes";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import {
  Autocomplete,
  Avatar,
  Badge,
  Box,
  Button,
  Divider,
  Group,
  Stack,
  Text,
  ThemeIcon,
  UnstyledButton,
} from "@mantine/core";
import { IconPlus } from "@tabler/icons-react";
import type { DeviceListEntry, DeviceListOwner } from "@/lib/api";
import { DeviceState } from "@/lib/api";
import { resolveDeviceIcon } from "@/features/devices/deviceTypeConfig";
import { GroupBadgeList } from "@/features/host-access/components/GroupBadgeList";
import { useDeviceList } from "@/features/devices/hooks/useDeviceList";

dayjs.extend(relativeTime);

function getInitials(name: string): string {
  return name
    .split(" ")
    .map((w) => w[0])
    .filter(Boolean)
    .slice(0, 2)
    .join("")
    .toUpperCase();
}

function deviceStatusText(entry: DeviceListEntry): string {
  const isLive = entry.live_address_count > 0;
  const isStale = entry.state === DeviceState.STALE;
  const ago = entry.last_seen_at ? ` · ${dayjs(entry.last_seen_at).fromNow()}` : "";

  if (isLive) return `${entry.live_address_count} live${ago}`;
  if (isStale) return `stale${ago}`;
  return entry.last_seen_at ? `seen ${dayjs(entry.last_seen_at).fromNow()}` : "never seen";
}

function DevicePanelItem({
  entry,
  isSelected,
  onSelect,
}: {
  entry: DeviceListEntry;
  isSelected: boolean;
  onSelect: () => void;
}) {
  const renderIcon = resolveDeviceIcon(entry.icon);
  const isStale = entry.state === DeviceState.STALE;
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
        <ThemeIcon variant="transparent" size="sm" c={isStale ? "dimmed" : undefined}>
          {renderIcon({ size: 16 })}
        </ThemeIcon>
        <Box style={{ flex: 1, minWidth: 0 }}>
          <Text
            size="sm"
            c={isStale ? "dimmed" : undefined}
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

export interface OwnerDevicesPanelProps {
  owner: DeviceListOwner;
  devices: DeviceListEntry[];
  selectedDeviceId: number | undefined;
  onSelectDevice: (id: number) => void;
  onAddDevice?: () => void;
}

export function OwnerDevicesPanel({
  owner,
  devices,
  selectedDeviceId,
  onSelectDevice,
  onAddDevice,
}: OwnerDevicesPanelProps) {
  const navigate = useNavigate();
  const { data: allGroups } = useDeviceList();

  const jumpData = (allGroups ?? [])
    .filter((g) => g.owner.id !== owner.id)
    .map((g) => g.owner.display_name);

  function handleJump(displayName: string) {
    const found = (allGroups ?? []).find(
      (g) => g.owner.display_name === displayName && g.owner.id !== owner.id,
    );
    if (found) navigate(buildRoute.userDevices(found.owner.id));
  }

  const ownerFirstName = owner.display_name.split(" ")[0];

  return (
    <Stack gap="md">
      {/* Owner card */}
      <Group gap="sm" align="flex-start">
        <Avatar radius="xl" size="md" color="indigo">
          {getInitials(owner.display_name)}
        </Avatar>
        <Stack gap={2}>
          <Group gap="xs" align="center">
            <Text fw={600} size="sm">{owner.display_name}</Text>
            {owner.role === "admin" && (
              <Badge size="xs" color="indigo" variant="light">admin</Badge>
            )}
          </Group>
          <Group gap={4} align="center" wrap="nowrap">
            {owner.bypass_host_check ? (
              <Badge size="xs" color="orange" variant="filled">bypass</Badge>
            ) : owner.host_groups.length > 0 ? (
              <GroupBadgeList groups={owner.host_groups} size="xs" />
            ) : null}
            <Text size="xs" c="dimmed">
              {owner.device_count} device{owner.device_count !== 1 ? "s" : ""}
              {owner.live_address_count > 0
                ? ` · ${owner.live_address_count} live`
                : ""}
            </Text>
          </Group>
        </Stack>
      </Group>

      <Divider />

      {/* Device list */}
      <Stack gap={0}>
        <Text size="xs" c="dimmed" fw={600} tt="uppercase" mb="xs" style={{ letterSpacing: "0.05em" }}>
          {ownerFirstName}&apos;s devices · {devices.length}
        </Text>
        {devices.map((entry) => (
          <DevicePanelItem
            key={entry.id}
            entry={entry}
            isSelected={entry.id === selectedDeviceId}
            onSelect={() => onSelectDevice(entry.id)}
          />
        ))}
      </Stack>

      <Button
        variant="subtle"
        size="xs"
        leftSection={<IconPlus size={14} />}
        onClick={onAddDevice}
        justify="flex-start"
        c="dimmed"
      >
        add device
      </Button>

      {/* Jump to another owner */}
      {jumpData.length > 0 && (
        <>
          <Divider />
          <Stack gap={4}>
            <Text size="xs" c="dimmed" fw={600} tt="uppercase" style={{ letterSpacing: "0.05em" }}>
              Jump
            </Text>
            <Autocomplete
              placeholder="other owner..."
              data={jumpData}
              onOptionSubmit={handleJump}
              size="xs"
              maxDropdownHeight={200}
            />
          </Stack>
        </>
      )}
    </Stack>
  );
}
