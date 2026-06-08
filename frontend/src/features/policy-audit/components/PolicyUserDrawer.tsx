import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import { useMemo, useState } from "react";
import {
  Alert,
  Avatar,
  Badge,
  Button,
  Card,
  Divider,
  Drawer,
  Group,
  Stack,
  Tabs,
  Text,
  TextInput,
  Tooltip,
} from "@mantine/core";
import {
  IconPlayerPlay,
  IconSearch,
  IconShield,
  IconShieldOff,
  IconUsers,
  IconWifi,
  IconWifiOff,
} from "@tabler/icons-react";
import type { PolicyUserEntry, PolicyUserIp } from "@/lib/api";
import { deriveUserStatus } from "../userStatus";

dayjs.extend(relativeTime);

// ─── helpers ───────────────────────────────────────────────────────────────

function heartbeatAgo(ip: PolicyUserIp): string {
  if (ip.addresses.length === 0) return "unknown";
  const latest = ip.addresses.reduce((max, a) =>
    dayjs(a.updated_at).isAfter(dayjs(max.updated_at)) ? a : max,
  );
  return dayjs(latest.updated_at).fromNow();
}

function buildIpReachSets(user: PolicyUserEntry) {
  return user.ips.map((ip) => ({
    bypass_at_ip: ip.bypass_at_ip,
    set: new Set(ip.effective_hosts),
  }));
}

function reachingIpCount(host: string, reachSets: ReturnType<typeof buildIpReachSets>): number {
  return reachSets.filter((s) => s.bypass_at_ip || s.set.has(host)).length;
}

interface DeviceInfo {
  device_id: number;
  device_name: string;
  last_seen_at: string;
  live_address_count: number;
}

function computeDevices(user: PolicyUserEntry): DeviceInfo[] {
  const map = new Map<number, DeviceInfo>();
  for (const ip of user.ips) {
    for (const addr of ip.addresses) {
      const existing = map.get(addr.device_id);
      if (!existing) {
        map.set(addr.device_id, {
          device_id: addr.device_id,
          device_name: addr.device_name,
          last_seen_at: addr.updated_at,
          live_address_count: 1,
        });
      } else {
        existing.live_address_count++;
        if (dayjs(addr.updated_at).isAfter(dayjs(existing.last_seen_at))) {
          existing.last_seen_at = addr.updated_at;
        }
      }
    }
  }
  return [...map.values()];
}

// ─── identity header ────────────────────────────────────────────────────────

function DrawerIdentity({ user }: { user: PolicyUserEntry }) {
  const status = deriveUserStatus(user);
  const hasLiveIps = status === "live_with_access" || status === "live_no_host_access";
  const hasHostAccess = status === "live_with_access" || status === "no_live_ips";

  return (
    <Group gap="sm" wrap="nowrap" mb="xs">
      <Avatar size="lg" color="indigo" variant="filled" radius="xl">
        {user.display_name[0]?.toUpperCase() ?? "?"}
      </Avatar>
      <Group gap="xs" align="center" wrap="wrap">
        <Text size="xl" fw={700}>
          {user.display_name}
        </Text>
        {status === "bypass" ? (
          <Badge variant="light" color="orange" leftSection={<IconShieldOff size={12} />}>
            Bypass
          </Badge>
        ) : (
          <>
            <Tooltip
              label={hasLiveIps ? "At least one live IP in the cache" : "No live IPs in the cache"}
              withArrow
            >
              <Badge
                variant="light"
                color={hasLiveIps ? "orange" : "gray"}
                leftSection={hasLiveIps ? <IconWifi size={12} /> : <IconWifiOff size={12} />}
              >
                {hasLiveIps ? "Live" : "Offline"}
              </Badge>
            </Tooltip>
            <Tooltip
              label={hasHostAccess ? "Has host grants in the allowlist" : "No host grants — all requests will be denied"}
              withArrow
            >
              <Badge
                variant="light"
                color={hasHostAccess ? "green" : "red"}
                leftSection={hasHostAccess ? <IconShield size={12} /> : <IconShieldOff size={12} />}
              >
                {hasHostAccess ? "Has access" : "No host access"}
              </Badge>
            </Tooltip>
          </>
        )}
        {user.on_shared_ip && (
          <Badge variant="light" color="yellow" leftSection={<IconUsers size={12} />}>
            Shared IP
          </Badge>
        )}
        {user.is_admin && (
          <Badge variant="light" color="indigo" size="xs">
            Admin
          </Badge>
        )}
      </Group>
    </Group>
  );
}

// ─── stats tiles ────────────────────────────────────────────────────────────

function StatTile({ label, value, sub }: { label: string; value: React.ReactNode; sub: string }) {
  return (
    <Stack gap={2} style={{ flex: 1, minWidth: 0 }}>
      <Text size="xs" c="dimmed" fw={500}>
        {label}
      </Text>
      <Text fw={700} lh={1}>
        {value}
      </Text>
      <Text size="xs" c="dimmed">
        {sub}
      </Text>
    </Stack>
  );
}

function StatsRow({ user, totalHosts }: { user: PolicyUserEntry; totalHosts: number }) {
  const { reachableCount, trimmedCount } = useMemo(() => {
    const reachable = new Set<string>();
    for (const ip of user.ips) {
      if (ip.bypass_at_ip) {
        user.user_allowed_hosts.forEach((h) => reachable.add(h));
      } else {
        ip.effective_hosts.forEach((h) => reachable.add(h));
      }
    }
    return {
      reachableCount: reachable.size,
      trimmedCount: user.bypass_allowlist ? 0 : user.allowed_host_count - reachable.size,
    };
  }, [user]);

  return (
    <Card withBorder p="sm">
      <Group gap={0} wrap="nowrap">
        <StatTile
          label="Configured"
          value={user.bypass_allowlist ? `All / ${totalHosts}` : `${user.allowed_host_count} / ${totalHosts}`}
          sub={user.bypass_allowlist ? "bypass enabled" : "in your allowlist"}
        />
        <Divider orientation="vertical" mx="md" />
        <StatTile
          label="Reachable"
          value={user.bypass_allowlist ? "All" : `${reachableCount} / ${user.allowed_host_count}`}
          sub="from at least one live IP"
        />
        <Divider orientation="vertical" mx="md" />
        <StatTile
          label="Trimmed"
          value={trimmedCount}
          sub="by IP intersection"
        />
      </Group>
    </Card>
  );
}

// ─── Hosts tab ──────────────────────────────────────────────────────────────

type HostFilter = "all" | "reachable" | "trimmed";

function HostsTab({ user }: { user: PolicyUserEntry }) {
  const [filter, setFilter] = useState<HostFilter>("all");
  const [search, setSearch] = useState("");

  const sorted = useMemo(() => [...user.user_allowed_hosts].sort(), [user.user_allowed_hosts]);
  const reachSets = useMemo(() => buildIpReachSets(user), [user]);
  const totalIps = user.ips.length;

  const { ipCounts, counts } = useMemo(() => {
    const ipCounts = new Map<string, number>();
    let reachable = 0;
    let trimmed = 0;
    for (const h of sorted) {
      const n = reachingIpCount(h, reachSets);
      ipCounts.set(h, n);
      if (n > 0) reachable++;
      else trimmed++;
    }
    return { ipCounts, counts: { all: sorted.length, reachable, trimmed } };
  }, [sorted, reachSets]);

  const filtered = useMemo(() => {
    const q = search.toLowerCase().trim();
    return sorted.filter((h) => {
      const n = ipCounts.get(h) ?? 0;
      if (filter === "reachable" && n === 0) return false;
      if (filter === "trimmed" && n > 0) return false;
      return !q || h.toLowerCase().includes(q);
    });
  }, [sorted, filter, search, ipCounts]);

  if (user.bypass_allowlist) {
    return (
      <Group gap="xs" mt="sm">
        <IconShield size={16} color="var(--mantine-color-orange-5)" />
        <Text c="var(--pw-amber-text)" size="sm">
          Bypass enabled — all system hosts accessible from any live IP
        </Text>
      </Group>
    );
  }

  return (
    <Stack gap="sm" mt="sm">
      <Group gap="xs" wrap="nowrap">
        <Group gap={4} wrap="nowrap">
          {(["all", "reachable", "trimmed"] as HostFilter[]).map((v) => (
            <Badge
              key={v}
              variant={filter === v ? "filled" : "light"}
              color={v === "trimmed" ? "yellow" : "indigo"}
              style={{ cursor: "pointer" }}
              onClick={() => setFilter(v)}
            >
              {v.charAt(0).toUpperCase() + v.slice(1)} · {counts[v]}
            </Badge>
          ))}
        </Group>
        <TextInput
          placeholder="Search hosts..."
          value={search}
          onChange={(e) => setSearch(e.currentTarget.value)}
          leftSection={<IconSearch size={13} />}
          size="xs"
          style={{ flex: 1 }}
        />
      </Group>

      <Stack gap={0}>
        {filtered.length === 0 ? (
          <Text size="sm" c="dimmed" ta="center" py="xl">
            No hosts match.
          </Text>
        ) : (
          filtered.map((host) => {
            const n = ipCounts.get(host) ?? 0;
            const allIps = n === totalIps && totalIps > 0;
            const partial = n > 0 && !allIps;
            return (
              <Group key={host} justify="space-between" px={8} py={6}>
                <Text size="sm" ff="monospace">
                  {host}
                </Text>
                <Text
                  size="xs"
                  c={allIps ? "green.5" : partial ? "yellow.5" : "dimmed"}
                  style={{ whiteSpace: "nowrap" }}
                >
                  {n === 0 ? "No IPs ○" : allIps ? "All IPs ●" : `${n} of ${totalIps} IPs ◐`}
                </Text>
              </Group>
            );
          })
        )}
      </Stack>
    </Stack>
  );
}

// ─── Live IPs tab ───────────────────────────────────────────────────────────

function IpCard({
  ip,
  userAllowedHostCount,
  onTestIp,
}: {
  ip: PolicyUserIp;
  userAllowedHostCount: number;
  onTestIp: (ip: string) => void;
}) {
  const deviceNames = ip.addresses.map((a) => a.device_name).join(", ");

  const reachableLabel = ip.bypass_at_ip
    ? "All hosts (bypass)"
    : `${ip.effective_hosts.length} / ${userAllowedHostCount} hosts reachable`;

  return (
    <Card withBorder p="sm">
      <Stack gap="xs">
        <Group justify="space-between" wrap="nowrap">
          <Stack gap={2}>
            <Group gap="sm" wrap="nowrap" align="center">
              <Text fw={700} ff="monospace">
                {ip.ip}
              </Text>
              <Text size="xs" c="var(--pw-amber-text)">
                heartbeat {heartbeatAgo(ip)}
              </Text>
            </Group>
            {deviceNames && (
              <Text size="xs" c="dimmed">
                {deviceNames}
              </Text>
            )}
          </Stack>
          <Button
            size="xs"
            variant="subtle"
            color="indigo"
            leftSection={<IconPlayerPlay size={12} />}
            onClick={() => onTestIp(ip.ip)}
          >
            Test from this IP
          </Button>
        </Group>

        <Text size="sm">{reachableLabel}</Text>

        {ip.trimmed_hosts.length > 0 && (
          <Alert color="yellow" p="xs" title={`${ip.trimmed_hosts.length} host${ip.trimmed_hosts.length !== 1 ? "s" : ""} trimmed at this IP`}>
            <Text size="xs" c="dimmed" ff="monospace">
              {ip.trimmed_hosts.join(", ")}
            </Text>
          </Alert>
        )}

        {ip.shared_with_users.length > 0 && (
          <Card withBorder p="xs">
            <Text size="xs" c="dimmed" mb={4}>
              Also at this IP ({ip.shared_with_users.length} other user
              {ip.shared_with_users.length !== 1 ? "s" : ""})
            </Text>
            <Stack gap={4}>
              {ip.shared_with_users.map((u) => (
                <Group key={u.user_id} justify="space-between" wrap="nowrap">
                  <Text size="sm">
                    {u.user_name}
                    {u.devices.length > 0 && (
                      <Text span c="dimmed">
                        {" "}
                        · {u.devices.map((d) => d.device_name).join(", ")}
                      </Text>
                    )}
                  </Text>
                  <Text size="xs" c="dimmed">
                    {ip.effective_hosts.length} hosts
                  </Text>
                </Group>
              ))}
            </Stack>
          </Card>
        )}
      </Stack>
    </Card>
  );
}

function LiveIpsTab({
  user,
  onTestIp,
}: {
  user: PolicyUserEntry;
  onTestIp: (ip: string) => void;
}) {
  if (user.ips.length === 0) {
    return (
      <Text size="sm" c="dimmed" ta="center" py="xl" mt="sm">
        No live IPs in the cache.
      </Text>
    );
  }
  return (
    <Stack gap="sm" mt="sm">
      {user.ips.map((ip) => (
        <IpCard
          key={ip.ip}
          ip={ip}
          userAllowedHostCount={user.allowed_host_count}
          onTestIp={onTestIp}
        />
      ))}
    </Stack>
  );
}

// ─── Devices tab ────────────────────────────────────────────────────────────

function DevicesTab({ user }: { user: PolicyUserEntry }) {
  const devices = useMemo(() => computeDevices(user), [user]);

  if (devices.length === 0) {
    return (
      <Text size="sm" c="dimmed" ta="center" py="xl" mt="sm">
        No devices in the cache.
      </Text>
    );
  }

  return (
    <Stack gap="sm" mt="sm">
      {devices.map((d) => (
        <Card key={d.device_id} withBorder p="sm">
          <Group justify="space-between" align="flex-start" wrap="nowrap">
            <Stack gap={2}>
              <Text fw={600}>{d.device_name}</Text>
              <Text size="xs" c="dimmed">
                last seen {dayjs(d.last_seen_at).fromNow()}
              </Text>
            </Stack>
            <Text size="sm" c="dimmed" style={{ whiteSpace: "nowrap" }}>
              {d.live_address_count} live IP{d.live_address_count !== 1 ? "s" : ""}
            </Text>
          </Group>
        </Card>
      ))}
    </Stack>
  );
}

// ─── drawer content ─────────────────────────────────────────────────────────

function DrawerContent({
  user,
  totalHosts,
  onSelectIp,
  onClose,
}: {
  user: PolicyUserEntry;
  totalHosts: number;
  onSelectIp: (ip: string) => void;
  onClose: () => void;
}) {
  function handleTestIp(ip: string) {
    onSelectIp(ip);
    onClose();
  }

  const hostsTabLabel = user.bypass_allowlist ? "Hosts" : `Hosts ${user.allowed_host_count}`;

  return (
    <Stack gap="sm">
      <DrawerIdentity user={user} />
      <StatsRow user={user} totalHosts={totalHosts} />

      {user.on_shared_ip && (
        <Alert color="yellow" icon={<IconUsers size={16} />} title="Shares an IP with another user">
          One or more live IPs are shared with other users. Hosts reachable from those IPs are the
          intersection of each user&apos;s allowed set.
        </Alert>
      )}

      <Tabs defaultValue="hosts">
        <Tabs.List>
          <Tabs.Tab value="hosts">{hostsTabLabel}</Tabs.Tab>
          <Tabs.Tab value="ips">Live IPs {user.ips.length}</Tabs.Tab>
          <Tabs.Tab value="devices">Devices {user.device_count}</Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="hosts">
          <HostsTab user={user} />
        </Tabs.Panel>
        <Tabs.Panel value="ips">
          <LiveIpsTab user={user} onTestIp={handleTestIp} />
        </Tabs.Panel>
        <Tabs.Panel value="devices">
          <DevicesTab user={user} />
        </Tabs.Panel>
      </Tabs>
    </Stack>
  );
}

// ─── exported component ─────────────────────────────────────────────────────

export interface PolicyUserDrawerProps {
  user: PolicyUserEntry | null;
  totalHosts: number;
  onClose: () => void;
  onSelectIp: (ip: string) => void;
}

export function PolicyUserDrawer({ user, totalHosts, onClose, onSelectIp }: PolicyUserDrawerProps) {
  return (
    <Drawer
      opened={user !== null}
      onClose={onClose}
      position="right"
      size="lg"
      title={
        <Text size="xs" c="dimmed" tt="uppercase" fw={600} style={{ letterSpacing: "0.05em" }}>
          User · Policy
        </Text>
      }
    >
      {user && (
        <DrawerContent
          user={user}
          totalHosts={totalHosts}
          onSelectIp={onSelectIp}
          onClose={onClose}
        />
      )}
    </Drawer>
  );
}
