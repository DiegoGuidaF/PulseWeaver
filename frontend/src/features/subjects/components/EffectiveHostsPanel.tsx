import { useMemo, useState } from "react";
import { Badge, Group, Stack, Table, Text, TextInput } from "@mantine/core";
import { IconSearch } from "@tabler/icons-react";
import type { SubjectGroupDetail } from "@/lib/api";

interface EffectiveHost {
  id: number;
  fqdn: string;
  viaGroups: Array<{ id: number; name: string }>;
}

interface Props {
  groups: SubjectGroupDetail[];
  assignedGroupIds: Set<number>;
  bypassHostCheck: boolean;
}

export function EffectiveHostsPanel({ groups, assignedGroupIds, bypassHostCheck }: Props) {
  const [search, setSearch] = useState("");
  const [groupFilter, setGroupFilter] = useState<number | null>(null);

  const effectiveHosts = useMemo((): EffectiveHost[] => {
    const hostMap = new Map<number, EffectiveHost>();
    for (const group of groups) {
      if (!assignedGroupIds.has(group.id)) continue;
      for (const host of group.hosts) {
        const existing = hostMap.get(host.id);
        if (existing) {
          existing.viaGroups.push({ id: group.id, name: group.name });
        } else {
          hostMap.set(host.id, {
            id: host.id,
            fqdn: host.fqdn,
            viaGroups: [{ id: group.id, name: group.name }],
          });
        }
      }
    }
    return [...hostMap.values()].sort((a, b) => a.fqdn.localeCompare(b.fqdn));
  }, [groups, assignedGroupIds]);

  const totalKnownHosts = useMemo(() => {
    const ids = new Set<number>();
    for (const g of groups) {
      for (const h of g.hosts) ids.add(h.id);
    }
    return ids.size;
  }, [groups]);

  const assignedGroups = useMemo(
    () => groups.filter((g) => assignedGroupIds.has(g.id)),
    [groups, assignedGroupIds],
  );

  const filteredHosts = useMemo(() => {
    let list = effectiveHosts;
    if (groupFilter !== null) {
      list = list.filter((h) => h.viaGroups.some((g) => g.id === groupFilter));
    }
    const q = search.trim().toLowerCase();
    if (q) list = list.filter((h) => h.fqdn.toLowerCase().includes(q));
    return list;
  }, [effectiveHosts, groupFilter, search]);

  const header = (
    <Group justify="space-between" align="center" mb="xs">
      <Text size="sm" fw={600} c="dimmed" tt="uppercase" style={{ letterSpacing: "0.05em" }}>
        Effective hosts
      </Text>
      {!bypassHostCheck && (
        <Text size="xs" c="dimmed">
          {effectiveHosts.length} {effectiveHosts.length === 1 ? "host" : "hosts"} · read-only
        </Text>
      )}
      {bypassHostCheck && (
        <Text size="xs" c="dimmed">bypass active</Text>
      )}
    </Group>
  );

  if (bypassHostCheck) {
    return (
      <Stack gap="sm">
        {header}
        <div
          style={{
            padding: "16px",
            border: "1.5px dashed var(--mantine-color-default-border)",
            borderRadius: "var(--mantine-radius-sm)",
            textAlign: "center",
          }}
        >
          <Text size="sm" c="dimmed">All {totalKnownHosts} known hosts are reachable.</Text>
          <Text size="xs" c="dimmed" mt={4}>
            This subject also reaches hosts not yet in the catalog. Group filter has no effect
            while bypass is active.
          </Text>
        </div>
      </Stack>
    );
  }

  return (
    <Stack gap="sm">
      {header}

      <TextInput
        placeholder="Search hosts..."
        leftSection={<IconSearch size={14} />}
        value={search}
        onChange={(e) => setSearch(e.currentTarget.value)}
        size="sm"
      />

      {assignedGroups.length > 0 && (
        <Group gap="xs" wrap="wrap">
          <Text size="xs" c="dimmed">Group:</Text>
          <Badge
            variant={groupFilter === null ? "filled" : "outline"}
            color="indigo"
            size="sm"
            style={{ cursor: "pointer" }}
            onClick={() => setGroupFilter(null)}
          >
            All
          </Badge>
          {assignedGroups.map((g) => (
            <Badge
              key={g.id}
              variant={groupFilter === g.id ? "filled" : "outline"}
              color="indigo"
              size="sm"
              style={{ cursor: "pointer" }}
              onClick={() => setGroupFilter(groupFilter === g.id ? null : g.id)}
            >
              {g.name}
            </Badge>
          ))}
        </Group>
      )}

      {effectiveHosts.length === 0 ? (
        <Text size="sm" c="dimmed" ta="center" py="md">
          {assignedGroupIds.size === 0
            ? "No groups assigned — no hosts are accessible."
            : "No hosts match."}
        </Text>
      ) : filteredHosts.length === 0 ? (
        <Text size="sm" c="dimmed" ta="center" py="md">
          No hosts match.
        </Text>
      ) : (
        <Table fz="sm" withRowBorders>
          <Table.Thead>
            <Table.Tr>
              <Table.Th>FQDN</Table.Th>
              <Table.Th>Via group</Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {filteredHosts.map((host) => (
              <Table.Tr key={host.id}>
                <Table.Td ff="monospace" fz="xs">
                  {host.fqdn}
                </Table.Td>
                <Table.Td>
                  <Group gap={4} wrap="wrap">
                    {host.viaGroups.map((g) => (
                      <Badge key={g.id} size="xs" variant="light" color="indigo">
                        {g.name}
                      </Badge>
                    ))}
                  </Group>
                </Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>
      )}
    </Stack>
  );
}
