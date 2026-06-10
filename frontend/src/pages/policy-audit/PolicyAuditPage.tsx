import { useState } from "react";
import {
  Card,
  Center,
  Group,
  Loader,
  SimpleGrid,
  Stack,
  Tabs,
  Text,
  ThemeIcon,
  Title,
} from "@mantine/core";
import {
  IconArrowsExchange,
  IconBolt,
  IconClock,
  IconNetwork,
  IconUsers,
  IconWorld,
} from "@tabler/icons-react";
import type { PolicyUserMapAudit, PolicyUserEntry } from "@/lib/api";
import { useDateFormatter } from "@/contexts/useDateTimePrefs";
import { usePolicyMap } from "@/features/policy-audit/hooks/usePolicyMap";
import { ErrorState } from "@/components/ErrorState";
import { SimulateBar } from "@/features/policy-audit/components/SimulateBar";
import { PolicyUserTable } from "@/features/policy-audit/components/PolicyUserTable";
import { PolicyUserDrawer } from "@/features/policy-audit/components/PolicyUserDrawer";
import { NetworkPolicyCacheTab } from "@/features/policy-audit/components/NetworkPolicyCacheTab";

function relativeTime(dateStr: string): string {
  const seconds = Math.floor((Date.now() - new Date(dateStr).getTime()) / 1000);
  if (seconds < 60) return `${seconds}s ago`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
  return `${Math.floor(seconds / 3600)}h ago`;
}

interface StatTileProps {
  icon: React.ReactNode;
  label: string;
  value: string | number;
  sub?: string;
}

function StatTile({ icon, label, value, sub }: StatTileProps) {
  return (
    <Stack gap={4} style={{ minWidth: 0 }}>
      <Group gap={6}>
        {icon}
        <Text size="xs" c="dimmed" fw={500}>
          {label}
        </Text>
      </Group>
      <Text size="xl" fw={700} lh={1}>
        {value}
      </Text>
      {sub && (
        <Text size="xs" c="dimmed" truncate="end">
          {sub}
        </Text>
      )}
    </Stack>
  );
}

function CacheStatsHeader({ data }: { data: PolicyUserMapAudit }) {
  const formatDateTime = useDateFormatter();

  return (
    <Card withBorder p="lg">
      <SimpleGrid cols={{ base: 2, sm: 3, lg: 6 }} spacing="lg">
        <StatTile
          icon={<IconClock size={14} color="var(--mantine-color-dimmed)" />}
          label="Last refreshed"
          value={relativeTime(data.refreshed_at)}
          sub={formatDateTime(data.refreshed_at)}
        />
        <StatTile
          icon={<IconBolt size={14} color="var(--mantine-color-dimmed)" />}
          label="Regen time"
          value={`${data.refresh_duration_ms}ms`}
        />
        <StatTile
          icon={<IconWorld size={14} color="var(--mantine-color-dimmed)" />}
          label="Enabled IPs"
          value={data.total_ip_count}
          sub={`${data.total_device_count} device${data.total_device_count !== 1 ? "s" : ""}`}
        />
        <StatTile
          icon={<IconNetwork size={14} color="var(--mantine-color-dimmed)" />}
          label="Known hosts"
          value={data.total_host_count}
        />
        <StatTile
          icon={<IconUsers size={14} color="var(--mantine-color-dimmed)" />}
          label="Shared IPs"
          value={data.shared_ip_count}
        />
        <StatTile
          icon={<IconNetwork size={14} color="var(--mantine-color-dimmed)" />}
          label="Network policies"
          value={data.total_network_policy_count}
        />
      </SimpleGrid>
    </Card>
  );
}

export function PolicyAuditPage() {
  const { data, isPending, isError, refetch } = usePolicyMap();
  const [simulateIp, setSimulateIp] = useState("");
  const [selectedUser, setSelectedUser] = useState<PolicyUserEntry | null>(null);

  return (
    <Stack maw={1200} gap="md">
      <div>
        <Title order={1}>Access Verification</Title>
        <Text c="dimmed" mt={4}>
          Verify who can reach what. Check live IPs, effective host grants, and simulate a request
          using the same policy the proxy evaluates.
        </Text>
      </div>

      {data && <CacheStatsHeader data={data} />}

      <Card withBorder>
        <Stack gap="xs">
          <Group gap="xs">
            <ThemeIcon size="sm" variant="transparent" color="indigo">
              <IconArrowsExchange size={16} />
            </ThemeIcon>
            <Text size="sm" fw={500}>
              Test request
            </Text>
          </Group>
          <Text size="xs" c="dimmed">
            Send an (IP, host) pair to the policy endpoint. Same code path the proxy uses to allow
            or deny.
          </Text>
          <SimulateBar ip={simulateIp} onIpChange={setSimulateIp} />
        </Stack>
      </Card>

      {isPending && (
        <Center py="xl">
          <Loader />
        </Center>
      )}

      {isError && (
        <ErrorState
          title="Failed to load access data"
          message="Could not fetch the policy map snapshot. Make sure you have admin access."
          onRetry={() => refetch()}
        />
      )}

      {data && (
        <Tabs defaultValue="devices">
          <Tabs.List>
            <Tabs.Tab value="devices">Device entries</Tabs.Tab>
            <Tabs.Tab value="network-policies">Network policies</Tabs.Tab>
          </Tabs.List>
          <Tabs.Panel value="devices" pt="md">
            <PolicyUserTable
              data={data}
              totalHosts={data.total_host_count}
              onSelectIp={setSimulateIp}
              onSelectUser={setSelectedUser}
            />
          </Tabs.Panel>
          <Tabs.Panel value="network-policies" pt="md">
            <NetworkPolicyCacheTab
              entries={data.network_policies}
              totalHosts={data.total_host_count}
            />
          </Tabs.Panel>
        </Tabs>
      )}

      <PolicyUserDrawer
        user={selectedUser}
        totalHosts={data?.total_host_count ?? 0}
        onClose={() => setSelectedUser(null)}
        onSelectIp={setSimulateIp}
      />
    </Stack>
  );
}
