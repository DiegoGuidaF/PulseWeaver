import { useMemo, useState } from "react";
import {
  Avatar,
  Badge,
  Card,
  Checkbox,
  Group,
  Progress,
  SegmentedControl,
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
} from "@tabler/icons-react";
import type { PolicyUserEntry, PolicyUserMapAudit } from "@/lib/api";

const MAX_VISIBLE_IPS = 3;

type StatusFilter = "all" | "allowlisted" | "bypass" | "no_access";

function getUserStatus(user: PolicyUserEntry): "bypass" | "no_access" | "allowlisted" {
  if (user.bypass_allowlist) return "bypass";
  if (user.ips.length === 0) return "no_access";
  return "allowlisted";
}

function matchesSearch(user: PolicyUserEntry, q: string): boolean {
  if (!q) return true;
  const lower = q.toLowerCase();
  return (
    user.user_name.toLowerCase().includes(lower) ||
    user.ips.some(
      (ip) =>
        ip.ip.toLowerCase().includes(lower) ||
        ip.addresses.some((a) => a.device_name.toLowerCase().includes(lower)),
    )
  );
}

function StatusBadge({ status }: { status: "bypass" | "no_access" | "allowlisted" }) {
  if (status === "bypass") {
    return (
      <Badge variant="light" color="orange" size="sm" leftSection={<IconShieldOff size={12} />}>
        Bypass
      </Badge>
    );
  }
  if (status === "no_access") {
    return (
      <Badge variant="light" color="gray" size="sm">
        No access
      </Badge>
    );
  }
  return (
    <Badge variant="light" color="green" size="sm">
      Allowlisted
    </Badge>
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
  status: "bypass" | "no_access" | "allowlisted";
}) {
  if (status === "bypass") {
    return (
      <Group gap={4} wrap="nowrap">
        <IconShield size={14} color="var(--mantine-color-orange-5)" />
        <Text size="sm" c="orange.5" style={{ whiteSpace: "nowrap" }}>
          Any host
        </Text>
      </Group>
    );
  }

  if (status === "no_access") {
    return (
      <Tooltip label="User has no enabled IPs in the cache" withArrow>
        <Text size="sm" c="dimmed" style={{ cursor: "default" }}>
          —
        </Text>
      </Tooltip>
    );
  }

  const pct = totalHosts > 0 ? (user.allowed_host_count / totalHosts) * 100 : 0;
  return (
    <Group gap="xs" wrap="nowrap" style={{ minWidth: 90 }}>
      <Text size="sm" style={{ whiteSpace: "nowrap" }}>
        {user.allowed_host_count} / {totalHosts}
      </Text>
      <Progress value={pct} size="xs" style={{ flex: 1, minWidth: 40 }} color="indigo" />
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
  const status = getUserStatus(user);

  return (
    <Table.Tr style={{ cursor: "pointer" }} onClick={onSelect}>
      <Table.Td>
        <Group gap="sm" wrap="nowrap">
          <Avatar size="md" color="indigo" variant="filled" radius="xl">
            {user.user_name[0]?.toUpperCase() ?? "?"}
          </Avatar>
          <Group gap="xs" wrap="nowrap">
            <Text size="sm" fw={500}>
              {user.user_name}
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
        <Text size="sm" c={user.device_count > 0 ? undefined : "dimmed"}>
          {user.device_count > 0 ? user.device_count : "—"}
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

  const counts = useMemo(
    () => ({
      all: data.users.length,
      allowlisted: data.users.filter((u) => !u.bypass_allowlist && u.ips.length > 0).length,
      bypass: data.users.filter((u) => u.bypass_allowlist).length,
      no_access: data.users.filter((u) => u.ips.length === 0).length,
    }),
    [data.users],
  );

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    return data.users.filter((user) => {
      if (statusFilter !== "all" && getUserStatus(user) !== statusFilter) return false;
      if (sharedOnly && !user.on_shared_ip) return false;
      return matchesSearch(user, q);
    });
  }, [data.users, search, statusFilter, sharedOnly]);

  return (
    <Stack gap="sm">
      <Group justify="space-between" wrap="nowrap">
        <TextInput
          placeholder="Search by IP, user, or device..."
          value={search}
          onChange={(e) => setSearch(e.currentTarget.value)}
          leftSection={<IconSearch size={14} />}
          size="sm"
          style={{ width: 280 }}
        />
        <Group gap="sm" wrap="nowrap">
          <SegmentedControl
            size="xs"
            value={statusFilter}
            onChange={(v) => setStatusFilter(v as StatusFilter)}
            data={[
              { label: `All (${counts.all})`, value: "all" },
              { label: `Allowlisted (${counts.allowlisted})`, value: "allowlisted" },
              { label: `Bypass (${counts.bypass})`, value: "bypass" },
              { label: `No access (${counts.no_access})`, value: "no_access" },
            ]}
          />
          <Checkbox
            label="Shared IPs only"
            size="sm"
            checked={sharedOnly}
            onChange={(e) => setSharedOnly(e.currentTarget.checked)}
          />
          <Text size="xs" c="dimmed" style={{ whiteSpace: "nowrap" }}>
            {filtered.length} of {data.users.length}
          </Text>
        </Group>
      </Group>

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
                <Table.Th />
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
