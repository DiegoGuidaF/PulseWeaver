import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { buildRoute } from "@/lib/routes";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import {
  Avatar,
  Badge,
  Box,
  Button,
  CloseButton,
  Divider,
  Group,
  Select,
  Stack,
  Text,
  TextInput,
  ThemeIcon,
  Tooltip,
  UnstyledButton,
} from "@mantine/core";
import { IconPlus, IconSearch } from "@tabler/icons-react";
import type { DeviceListEntry, DeviceListOwner } from "@/lib/api";
import { DeviceState } from "@/lib/api";
import { resolveDeviceIcon } from "@/features/devices/deviceTypeConfig";
import { GroupBadgeList } from "@/features/host-access/components/GroupBadgeList";
import { useDeviceList } from "@/features/devices/hooks/useDeviceList";
import { isInactiveState } from "@/features/devices/constants";

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
  const ago = entry.last_seen_at ? ` · ${dayjs(entry.last_seen_at).fromNow()}` : "";

  if (entry.state === DeviceState.DISABLED) return `disabled${ago}`;
  if (entry.live_address_count > 0) return `${entry.live_address_count} live${ago}`;
  if (entry.state === DeviceState.PENDING_CLAIM) return "pairing pending";
  if (entry.state === DeviceState.EXPIRED_CLAIM) return "code expired";
  if (entry.state === DeviceState.STALE) return `stale${ago}`;
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

  const [query, setQuery] = useState("");
  const trimmed = query.trim().toLowerCase();
  const filtered = useMemo(
    () => (trimmed ? devices.filter((d) => d.name.toLowerCase().includes(trimmed)) : devices),
    [devices, trimmed],
  );
  // Only worth a search box once the list is long enough to scan poorly.
  const showFilter = devices.length > 3;

  // Option values are owner ids, not display names: names can collide across
  // owners and duplicate Select option values are rejected by Mantine.
  const jumpData = (allGroups ?? [])
    .filter((g) => g.owner.id !== owner.id)
    .map((g) => ({ value: String(g.owner.id), label: g.owner.display_name }));

  function handleJump(ownerId: string) {
    navigate(buildRoute.userDevices(Number(ownerId)));
  }

  const ownerFirstName = owner.display_name.split(" ")[0];

  return (
    <Stack gap="md">
      {/* Owner card */}
      <Group gap="sm" align="flex-start" wrap="nowrap">
        <Avatar radius="xl" size="md" color="indigo" style={{ flexShrink: 0 }}>
          {getInitials(owner.display_name)}
        </Avatar>
        <Stack gap={6}>
          <Stack gap={2}>
            <Text fw={600} size="sm">{owner.display_name}</Text>
            {owner.role === "admin" && (
              <Box>
                <Badge size="xs" color="indigo" variant="light">admin</Badge>
              </Box>
            )}
          </Stack>
          <Stack gap={4}>
            {owner.bypass_host_check ? (
              <Box>
                <Tooltip label="Host check bypassed — this user's devices can access any configured host" withArrow>
                  <Badge size="xs" color="orange" variant="filled">All hosts</Badge>
                </Tooltip>
              </Box>
            ) : owner.host_groups.length > 0 ? (
              <GroupBadgeList groups={owner.host_groups} size="xs" />
            ) : null}
            <Text size="xs" c="dimmed">
              {owner.device_count} device{owner.device_count !== 1 ? "s" : ""}
              {owner.live_address_count > 0
                ? ` · ${owner.live_address_count} IPs live`
                : ""}
            </Text>
          </Stack>
        </Stack>
      </Group>

      <Divider />

      {/* Device list */}
      <Stack gap={0}>
        <Text size="xs" c="dimmed" fw={600} tt="uppercase" mb="xs" style={{ letterSpacing: "0.05em" }}>
          {ownerFirstName}&apos;s devices · {devices.length}
        </Text>
        {showFilter && (
          <TextInput
            size="xs"
            mb="xs"
            placeholder="Filter by name…"
            value={query}
            onChange={(e) => setQuery(e.currentTarget.value)}
            leftSection={<IconSearch size={13} />}
            rightSection={
              query ? (
                <CloseButton
                  size="sm"
                  aria-label="Clear filter"
                  onClick={() => setQuery("")}
                />
              ) : null
            }
          />
        )}
        {filtered.length > 0 ? (
          filtered.map((entry) => (
            <DevicePanelItem
              key={entry.id}
              entry={entry}
              isSelected={entry.id === selectedDeviceId}
              onSelect={() => onSelectDevice(entry.id)}
            />
          ))
        ) : (
          <Text size="xs" c="dimmed" px="sm" py={6}>
            {trimmed ? `No devices match “${query}”.` : "No devices yet."}
          </Text>
        )}
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
            <Select
              value={null}
              onChange={(val) => val && handleJump(val)}
              placeholder="other owner..."
              data={jumpData}
              searchable
              size="xs"
              maxDropdownHeight={200}
            />
          </Stack>
        </>
      )}
    </Stack>
  );
}
