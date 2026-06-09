import { useMemo, useState } from "react";
import { Chip, CloseButton, Group, Skeleton, Stack, Text, TextInput } from "@mantine/core";
import { IconDevices, IconSearch } from "@tabler/icons-react";
import { useDeviceList } from "@/features/devices/hooks/useDeviceList";
import { OwnerCard } from "@/features/devices/OwnerCard";
import { EmptyState } from "@/components/EmptyState";
import { ErrorState } from "@/components/ErrorState";
import { DeviceState } from "@/lib/api";

function LoadingSkeleton() {
  return (
    <Stack gap="md">
      {[0, 1].map((i) => (
        <Stack key={i} gap="xs">
          <Skeleton height={48} radius="md" />
          <Skeleton height={40} radius="sm" />
          <Skeleton height={40} radius="sm" />
        </Stack>
      ))}
    </Stack>
  );
}

export function OwnerGroupList() {
  const { data: groups, isLoading, error, refetch } = useDeviceList();
  const [nameQuery, setNameQuery] = useState("");
  const [filterStale, setFilterStale] = useState(false);
  const [filterBypass, setFilterBypass] = useState(false);

  const filteredGroups = useMemo(() => {
    if (!groups) return [];
    const trimmed = nameQuery.trim().toLowerCase();
    const hasDeviceFilter = Boolean(trimmed) || filterStale;

    return groups
      .filter((g) => !filterBypass || g.owner.bypass_host_check)
      .map((g) => ({
        ...g,
        devices: hasDeviceFilter
          ? g.devices.filter((d) => {
              if (trimmed && !d.name.toLowerCase().includes(trimmed)) return false;
              if (filterStale && d.state !== DeviceState.STALE) return false;
              return true;
            })
          : g.devices,
      }))
      .filter((g) => !hasDeviceFilter || g.devices.length > 0);
  }, [groups, nameQuery, filterStale, filterBypass]);

  if (isLoading) return <LoadingSkeleton />;

  if (error) {
    return <ErrorState error={error} title="Could not load devices" onRetry={() => refetch()} />;
  }

  if (!groups || groups.length === 0) {
    return (
      <EmptyState
        icon={IconDevices}
        title="No devices yet"
        description="Devices appear here, grouped by owner, once they have been registered."
      />
    );
  }

  return (
    <Stack gap="md">
      <Group gap="sm" align="center">
        <TextInput
          placeholder="Filter devices by name…"
          value={nameQuery}
          onChange={(e) => setNameQuery(e.currentTarget.value)}
          leftSection={<IconSearch size={14} />}
          rightSection={
            nameQuery ? (
              <CloseButton size="sm" aria-label="Clear name filter" onClick={() => setNameQuery("")} />
            ) : null
          }
          style={{ flex: 1 }}
        />
        <Chip checked={filterStale} onChange={setFilterStale} size="sm">Stale</Chip>
        <Chip checked={filterBypass} onChange={setFilterBypass} size="sm">All hosts</Chip>
      </Group>

      {filteredGroups.length === 0 ? (
        <Text c="dimmed" size="sm">No devices match the current filters.</Text>
      ) : (
        <Stack gap="md">
          {filteredGroups.map((group) => (
            <OwnerCard
              key={group.owner.id}
              owner={group.owner}
              devices={group.devices}
            />
          ))}
        </Stack>
      )}
    </Stack>
  );
}
