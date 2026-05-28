import { Alert, Skeleton, Stack } from "@mantine/core";
import { IconAlertCircle, IconDevices } from "@tabler/icons-react";
import { useDeviceList } from "@/features/devices/hooks/useDeviceList";
import { OwnerCard } from "@/features/devices/OwnerCard";
import { EmptyState } from "@/components/EmptyState";
import { toErrorMessage } from "@/lib/api-client";

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
  const { data: groups, isLoading, error } = useDeviceList();

  if (isLoading) return <LoadingSkeleton />;

  if (error) {
    return (
      <Alert color="red" icon={<IconAlertCircle size={16} />} title="Could not load devices">
        {toErrorMessage(error)}
      </Alert>
    );
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
      {groups.map((group) => (
        <OwnerCard
          key={group.owner.id}
          owner={group.owner}
          devices={group.devices}
        />
      ))}
    </Stack>
  );
}
