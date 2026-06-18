import { useMemo, useState } from "react";
import {
  Avatar,
  Badge,
  Card,
  Checkbox,
  Group,
  Progress,
  SegmentedControl,
  Select,
  Stack,
  Table,
  Text,
  TextInput,
  Tooltip,
} from "@mantine/core";
import {
  IconChevronRight,
  IconSearch,
  IconShield,
  IconShieldOff,
  IconUsers,
  IconWifi,
  IconWifiOff,
} from "@tabler/icons-react";
import type { PolicyUserEntry, PolicyUserMapAudit } from "@/lib/api";
import { PolicyUserStatus } from "@/lib/api";
import { AllHostsBypassPill } from "@/features/subjects/components/AllHostsBypassPill";

const MAX_VISIBLE_IPS = 3;

// NO_LIVE_IPS and NO_ACCESS render identically (a dimmed "Offline" row), so they
// collapse into a single filter rather than two the user can't tell apart.
const OFFLINE_FILTER = "offline";

type StatusFilter = "all" | typeof OFFLINE_FILTER | PolicyUserStatus;

function isOfflineStatus(status: PolicyUserStatus): boolean {
  return status === PolicyUserStatus.NO_LIVE_IPS || status === PolicyUserStatus.NO_ACCESS;
}

function matchesSearch(user: PolicyUserEntry, q: string): boolean {
  if (!q) return true;
  const lower = q.toLowerCase();
  return (
    user.display_name.toLowerCase().includes(lower) ||
    user.ips.some(
      (ip) =>
        ip.ip.toLowerCase().includes(lower) ||
        ip.addresses.some((a) => a.device_name.toLowerCase().includes(lower)),
    )
  );
}

/**
 * Reachability badge for the Status column. Host authorization is shown separately
 * in the Effective hosts column. Bypass users get the combined bypass pill since the
 * host check doesn't apply.
 */
function StatusBadge({ status }: { status: PolicyUserStatus }) {
  if (status === PolicyUserStatus.BYPASS) {
    return <AllHostsBypassPill />;
  }

  const hasLiveIps =
    status === PolicyUserStatus.LIVE_WITH_ACCESS || status === PolicyUserStatus.LIVE_NO_HOST_ACCESS;

  return (
    <Tooltip
      label={hasLiveIps ? "Device is online — at least one live IP in the cache" : "No live IPs in the cache — no current access"}
      withArrow
    >
      <Badge
        variant="light"
        color={hasLiveIps ? "orange" : "gray"}
        size="sm"
        leftSection={hasLiveIps ? <IconWifi size={11} /> : <IconWifiOff size={11} />}
      >
        {hasLiveIps ? "Live" : "Offline"}
      </Badge>
    </Tooltip>
  );
}

function IpBadgeList({ ips, onSelectIp }: { ips: string[]; onSelectIp: (ip: string) => void }) {
  const visible = ips.slice(0, MAX_VISIBLE_IPS);
  const overflow = ips.slice(MAX_VISIBLE_IPS);

  return (
    <Group gap={4} wrap="nowrap">
      {visible.map((ip) => (
        <Badge
          key={ip}
          variant="light"
          color="teal"
          size="sm"
          style={{ fontFamily: "monospace", cursor: "text" }}
          onClick={(e) => {
            e.stopPropagation();
            onSelectIp(ip);
          }}
        >
          • {ip}
        </Badge>
      ))}
      {overflow.length > 0 && (
        <Tooltip label={overflow.join(", ")} multiline maw={240} withArrow>
          <Badge variant="outline" color="gray" size="sm">
            +{overflow.length}
          </Badge>
        </Tooltip>
      )}
    </Group>
  );
}

function EffectiveHostsCell({
  user,
  totalHosts,
  status,
}: {
  user: PolicyUserEntry;
  totalHosts: number;
  status: PolicyUserStatus;
}) {
  if (status === PolicyUserStatus.BYPASS) {
    return (
      <Group gap={4} wrap="nowrap">
        <IconShield size={14} color="var(--mantine-color-orange-5)" />
        <Text size="sm" c="var(--pw-amber-text)" style={{ whiteSpace: "nowrap" }}>
          Any host
        </Text>
      </Group>
    );
  }

  if (status === PolicyUserStatus.LIVE_NO_HOST_ACCESS) {
    return (
      <Tooltip label="No host grants — all requests will be denied" withArrow>
        <Badge variant="light" color="red" size="sm" leftSection={<IconShieldOff size={11} />}>
          No host access
        </Badge>
      </Tooltip>
    );
  }

  // Offline users (no live IPs) have no current access; the row is dimmed and the cell empty.
  if (isOfflineStatus(status)) {
    return (
      <Text size="sm" c="dimmed">
        —
      </Text>
    );
  }

  const pct = totalHosts > 0 ? (user.allowed_host_count / totalHosts) * 100 : 0;
  return (
    <Group gap="xs" wrap="nowrap" style={{ minWidth: 90 }}>
      <Text size="sm" style={{ whiteSpace: "nowrap" }}>
        {user.allowed_host_count} / {totalHosts}
      </Text>
      <Progress.Root size="xs" style={{ flex: 1, minWidth: 40 }}>
        <Progress.Section
          value={pct}
          color="indigo"
          aria-label={`${user.allowed_host_count} of ${totalHosts} hosts accessible`}
        />
      </Progress.Root>
    </Group>
  );
}

function UserRow({
  user,
  totalHosts,
  onSelectIp,
  onSelect,
}: {
  user: PolicyUserEntry;
  totalHosts: number;
  onSelectIp: (ip: string) => void;
  onSelect: () => void;
}) {
  const status = user.status;
  const isOffline = isOfflineStatus(status);

  return (
    <Table.Tr
      style={{ cursor: "pointer", opacity: isOffline ? 0.55 : undefined }}
      onClick={onSelect}
    >
      <Table.Td>
        <Group gap="sm" wrap="nowrap">
          <Avatar size="md" color="indigo" variant="filled" radius="xl">
            {user.display_name[0]?.toUpperCase() ?? "?"}
          </Avatar>
          <Group gap="xs" wrap="nowrap">
            <Text size="sm" fw={500}>
              {user.display_name}
            </Text>
            {user.is_admin && (
              <Badge variant="light" color="indigo" size="xs">
                Admin
              </Badge>
            )}
          </Group>
        </Group>
      </Table.Td>

      <Table.Td>
        <Group gap="xs" wrap="nowrap">
          <StatusBadge status={status} />
          {user.on_shared_ip && (
            <Badge variant="light" color="yellow" size="sm" leftSection={<IconUsers size={12} />}>
              Shared IP
            </Badge>
          )}
        </Group>
      </Table.Td>

      <Table.Td>
        {user.ips.length > 0 ? (
          <IpBadgeList ips={user.ips.map((ip) => ip.ip)} onSelectIp={onSelectIp} />
        ) : (
          <Text size="sm" c="dimmed">
            —
          </Text>
        )}
      </Table.Td>

      <Table.Td>
        <EffectiveHostsCell user={user} totalHosts={totalHosts} status={status} />
      </Table.Td>

      <Table.Td>
        <Text size="sm" c={!isOffline && user.device_count > 0 ? undefined : "dimmed"}>
          {!isOffline && user.device_count > 0 ? user.device_count : "—"}
        </Text>
      </Table.Td>

      <Table.Td>
        <IconChevronRight size={16} color="var(--mantine-color-dimmed)" />
      </Table.Td>
    </Table.Tr>
  );
}

interface PolicyUserTableProps {
  data: PolicyUserMapAudit;
  totalHosts: number;
  onSelectIp: (ip: string) => void;
  onSelectUser: (user: PolicyUserEntry) => void;
}

export function PolicyUserTable({
  data,
  totalHosts,
  onSelectIp,
  onSelectUser,
}: PolicyUserTableProps) {
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("all");
  const [sharedOnly, setSharedOnly] = useState(false);

  const counts = useMemo(() => {
    const by = (s: PolicyUserStatus) => data.users.filter((u) => u.status === s).length;
    return {
      all: data.users.length,
      live_with_access: by(PolicyUserStatus.LIVE_WITH_ACCESS),
      live_no_host_access: by(PolicyUserStatus.LIVE_NO_HOST_ACCESS),
      bypass: by(PolicyUserStatus.BYPASS),
      offline: data.users.filter((u) => isOfflineStatus(u.status)).length,
    };
  }, [data.users]);

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    return data.users.filter((user) => {
      if (statusFilter !== "all") {
        const matchesStatus =
          statusFilter === OFFLINE_FILTER
            ? isOfflineStatus(user.status)
            : user.status === statusFilter;
        if (!matchesStatus) return false;
      }
      if (sharedOnly && !user.on_shared_ip) return false;
      return matchesSearch(user, q);
    });
  }, [data.users, search, statusFilter, sharedOnly]);

  const statusOptions = [
    { label: `All (${counts.all})`, value: "all" },
    { label: `Live + access (${counts.live_with_access})`, value: PolicyUserStatus.LIVE_WITH_ACCESS },
    { label: `Live, no access (${counts.live_no_host_access})`, value: PolicyUserStatus.LIVE_NO_HOST_ACCESS },
    { label: `Bypass (${counts.bypass})`, value: PolicyUserStatus.BYPASS },
    { label: `Offline (${counts.offline})`, value: OFFLINE_FILTER },
  ];

  return (
    <Stack gap="sm">
      <Stack gap="xs">
        <Group justify="space-between" wrap="wrap" gap="xs">
          <TextInput
            placeholder="Search by IP, user, or device..."
            value={search}
            onChange={(e) => setSearch(e.currentTarget.value)}
            leftSection={<IconSearch size={14} />}
            size="sm"
            style={{ flex: 1, minWidth: 220 }}
          />
          <Text size="xs" c="dimmed" style={{ whiteSpace: "nowrap" }}>
            {filtered.length} of {data.users.length}
          </Text>
        </Group>
        <Group gap="sm" wrap="wrap">
          <SegmentedControl
            visibleFrom="sm"
            size="xs"
            value={statusFilter}
            onChange={(v) => setStatusFilter(v as StatusFilter)}
            data={statusOptions}
          />
          <Select
            hiddenFrom="sm"
            size="sm"
            value={statusFilter}
            onChange={(v) => setStatusFilter((v ?? "all") as StatusFilter)}
            data={statusOptions}
            allowDeselect={false}
            style={{ flex: 1, minWidth: 160 }}
          />
          <Checkbox
            label="Shared IPs only"
            size="md"
            checked={sharedOnly}
            onChange={(e) => setSharedOnly(e.currentTarget.checked)}
          />
        </Group>
      </Stack>

      <Card withBorder p={0}>
        <Table.ScrollContainer minWidth={720}>
          <Table highlightOnHover>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>User</Table.Th>
                <Table.Th>Status</Table.Th>
                <Table.Th>Live IPs</Table.Th>
                <Table.Th>Effective hosts</Table.Th>
                <Table.Th>Devices</Table.Th>
                <Table.Th aria-label="Actions" />
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {filtered.length === 0 ? (
                <Table.Tr>
                  <Table.Td colSpan={6}>
                    <Text size="sm" c="dimmed" ta="center" py="xl">
                      {search || statusFilter !== "all" || sharedOnly
                        ? "No users match the current filters."
                        : "No users in the policy cache."}
                    </Text>
                  </Table.Td>
                </Table.Tr>
              ) : (
                filtered.map((user) => (
                  <UserRow
                    key={user.user_id}
                    user={user}
                    totalHosts={totalHosts}
                    onSelectIp={onSelectIp}
                    onSelect={() => onSelectUser(user)}
                  />
                ))
              )}
            </Table.Tbody>
          </Table>
        </Table.ScrollContainer>
      </Card>
    </Stack>
  );
}
