import { Link, Navigate, useParams } from "react-router-dom";
import { Badge, Group, Stack, Skeleton, Tabs, Text, Title, Anchor } from "@mantine/core";
import { IconChevronLeft } from "@tabler/icons-react";
import dayjs from "dayjs";
import { useDeviceDetail } from "@/features/devices/hooks/useDeviceDetail";
import { useDeviceAddressLeaseRule } from "@/features/devices/hooks/useDeviceAddressLeaseRule";
import { DeviceAddressesTab } from "@/features/devices/DeviceAddressesTab";
import { DeviceSettingsTab } from "@/features/devices/DeviceSettingsTab";
import { DeviceHistoryTab } from "@/features/devices/DeviceHistoryTab";
import { toErrorMessage } from "@/lib/api-client";

type DeviceDetailRouteParams = {
  deviceId?: string;
};

export function DeviceDetailPage() {
  const params = useParams<DeviceDetailRouteParams>();
  const deviceIdParam = params.deviceId;
  const deviceId = deviceIdParam
    ? Number.parseInt(deviceIdParam, 10)
    : Number.NaN;

  const { data: device, isLoading, isError, error } = useDeviceDetail(deviceId, 10_000);
  const { data: leaseRule } = useDeviceAddressLeaseRule(deviceId);

  if (!deviceIdParam || Number.isNaN(deviceId)) {
    return <Navigate to="/devices" replace />;
  }

  const liveIPs = device?.address_count ?? 0;
  const lastSeenAt = device?.last_seen_at;
  const expirySeconds = leaseRule?.ttl_seconds;

  function formatExpiry(seconds: number): string {
    if (seconds < 60) return `${seconds}s`;
    if (seconds < 3600) return `${Math.round(seconds / 60)}m`;
    return `${Math.round(seconds / 3600)}h`;
  }

  let headerContent: React.ReactNode;

  if (isLoading && !device) {
    headerContent = (
      <Stack gap={8}>
        <Skeleton height={28} width={192} />
        <Skeleton height={16} width={200} />
      </Stack>
    );
  } else if (device) {
    headerContent = (
      <Stack gap={4}>
        <Title order={2}>{device.name}</Title>
        <Group gap="md">
          <Text size="sm" c={liveIPs > 0 ? "orange.4" : "dimmed"} fw={liveIPs > 0 ? 500 : undefined}>
            {liveIPs} Live {liveIPs === 1 ? "IP" : "IPs"}
          </Text>
          <Text size="sm" c="dimmed">·</Text>
          <Text size="sm" c={lastSeenAt ? "orange.4" : "dimmed"}>
            {lastSeenAt ? `Last seen ${dayjs(lastSeenAt).fromNow()}` : "Never seen"}
          </Text>
          <Text size="sm" c="dimmed">·</Text>
          {expirySeconds != null ? (
            <Badge variant="light" color="green" size="sm">Auto-expiry: {formatExpiry(expirySeconds)}</Badge>
          ) : (
            <Text size="sm" c="dimmed">No auto-expiry</Text>
          )}
        </Group>
      </Stack>
    );
  } else if (isError) {
    headerContent = (
      <Text size="sm" c="red">
        Error loading device: {toErrorMessage(error)}
      </Text>
    );
  } else {
    headerContent = (
      <Text size="sm" c="dimmed">
        Device not found.{" "}
        <Anchor component={Link} to="/devices">Back to devices</Anchor>
      </Text>
    );
  }

  return (
    <Stack maw={1024} gap="xl">
      <Stack gap="md">
        <Anchor
          component={Link}
          to="/devices"
          c="dimmed"
          size="sm"
          style={{ display: "inline-flex", alignItems: "center", gap: 4 }}
        >
          <IconChevronLeft size={16} stroke={1.5} />
          <span>Back to devices</span>
        </Anchor>
        {headerContent}
      </Stack>

      <Tabs defaultValue="addresses" keepMounted={false}>
        <Tabs.List>
          <Tabs.Tab value="addresses">Addresses</Tabs.Tab>
          <Tabs.Tab value="settings">Settings</Tabs.Tab>
          <Tabs.Tab value="history">History</Tabs.Tab>
        </Tabs.List>
        <Tabs.Panel value="addresses" pt="md">
          <DeviceAddressesTab deviceId={deviceId} />
        </Tabs.Panel>
        <Tabs.Panel value="settings" pt="md">
          <DeviceSettingsTab deviceId={deviceId} device={device} />
        </Tabs.Panel>
        <Tabs.Panel value="history" pt="md">
          <DeviceHistoryTab deviceId={deviceId} />
        </Tabs.Panel>
      </Tabs>
    </Stack>
  );
}
