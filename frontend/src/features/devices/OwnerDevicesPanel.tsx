import { useMemo, useState } from "react";
import { useNavigate } from "react-router";
import { buildRoute } from "@/lib/routes";
import {
  Avatar,
  Badge,
  Button,
  CloseButton,
  Divider,
  Group,
  Select,
  Stack,
  Text,
  TextInput,
  Tooltip,
} from "@mantine/core";
import { IconPlus, IconSearch } from "@tabler/icons-react";
import type { FleetDevice, OwnerSummary } from "@/lib/api";
import { UserRole } from "@/lib/api";
import { DevicePanelItem } from "@/features/devices/DevicePanelItem";
import { getInitials } from "@/features/devices/ownerDisplay";
import { GroupBadgeList } from "@/features/host-access/components/GroupBadgeList";
import { useOwnerRefs } from "@/features/devices/hooks/useOwnerRefs";

export interface OwnerDevicesPanelProps {
  owner: OwnerSummary;
  devices: FleetDevice[];
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
  const { data: allOwners } = useOwnerRefs();

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
  const jumpData = (allOwners ?? [])
    .filter((o) => o.id !== owner.id)
    .map((o) => ({ value: String(o.id), label: o.display_name }));

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
          <Group gap="xs" align="center">
            <Text fw={600} size="sm">{owner.display_name}</Text>
            {owner.role === UserRole.ADMIN && (
              <Badge size="xs" color="indigo" variant="light">admin</Badge>
            )}
          </Group>
          <Stack gap={4}>
            {owner.bypass_host_check ? (
              <Tooltip label="Host check bypassed — this user's devices can access any configured host" withArrow>
                <Badge size="xs" color="orange" variant="filled" w="fit-content">All hosts</Badge>
              </Tooltip>
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
