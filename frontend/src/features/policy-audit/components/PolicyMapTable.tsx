import { useMemo, useState } from "react";
import {
  Badge,
  Card,
  Code,
  Group,
  Stack,
  Table,
  Text,
  TextInput,
  Tooltip,
} from "@mantine/core";
import { IconAlertTriangle, IconChevronDown, IconChevronRight } from "@tabler/icons-react";
import type { PolicyMapAudit, PolicyMapEntry } from "@/lib/api";
import { useDateFormatter } from "@/contexts/useDateTimePrefs";
import { ContributorCard } from "./ContributorCard";

interface PolicyMapTableProps {
  data: PolicyMapAudit;
  onSelectIp: (ip: string) => void;
}

function EntryRow({
  entry,
  expanded,
  onToggle,
  onSelectIp,
}: {
  entry: PolicyMapEntry;
  expanded: boolean;
  onToggle: () => void;
  onSelectIp: (ip: string) => void;
}) {
  return (
    <>
      <Table.Tr
        style={{ cursor: "pointer" }}
        onClick={onToggle}
      >
        <Table.Td>
          <Group gap="xs" wrap="nowrap">
            {expanded ? (
              <IconChevronDown size={14} style={{ flexShrink: 0, color: "var(--mantine-color-dimmed)" }} />
            ) : (
              <IconChevronRight size={14} style={{ flexShrink: 0, color: "var(--mantine-color-dimmed)" }} />
            )}
            <Code
              onClick={(e) => {
                e.stopPropagation();
                onSelectIp(entry.ip);
              }}
              style={{ cursor: "text" }}
            >
              {entry.ip}
            </Code>
          </Group>
        </Table.Td>
        <Table.Td>
          {entry.bypass_allowlist ? (
            <Badge variant="light" color="green" size="sm">
              bypass
            </Badge>
          ) : (
            <Badge variant="light" color="gray" size="sm">
              restricted
            </Badge>
          )}
        </Table.Td>
        <Table.Td>
          {entry.bypass_allowlist ? (
            <Text size="sm" c="dimmed">
              —
            </Text>
          ) : (
            <Text size="sm">{entry.allowed_hosts.length}</Text>
          )}
        </Table.Td>
        <Table.Td>
          <Text size="sm">{entry.contributors.length}</Text>
        </Table.Td>
        <Table.Td>
          {entry.intersection_applied && (
            <Tooltip label="Deny-wins intersection trimmed at least one contributor's grants" withArrow>
              <IconAlertTriangle size={16} color="var(--mantine-color-yellow-6)" />
            </Tooltip>
          )}
        </Table.Td>
      </Table.Tr>
      {expanded && (
        <Table.Tr>
          <Table.Td colSpan={5} p="md" style={{ background: "var(--mantine-color-default-hover)" }}>
                  <Stack gap="sm">
                {!entry.bypass_allowlist && entry.allowed_hosts.length > 0 && (
                  <div>
                    <Text size="xs" c="dimmed" mb={6} tt="uppercase" fw={500}>
                      Effective hosts
                    </Text>
                    <Group gap={4} wrap="wrap">
                      {entry.allowed_hosts.map((h) => (
                        <Badge key={h} variant="light" color="blue" size="sm">
                          {h}
                        </Badge>
                      ))}
                    </Group>
                  </div>
                )}

                <div>
                  <Text size="xs" c="dimmed" mb={6} tt="uppercase" fw={500}>
                    Contributors
                  </Text>
                  <Stack gap="xs">
                    {entry.contributors.map((c) => (
                      <ContributorCard key={`${c.user_id}-${c.device_id}`} contributor={c} />
                    ))}
                  </Stack>
                </div>

                {entry.intersection_applied && (
                  <Group gap={4}>
                    <IconAlertTriangle size={14} color="var(--mantine-color-yellow-6)" />
                    <Text size="xs" c="yellow.7">
                      Deny-wins intersection reduced the effective host set — at least one contributor
                      had hosts that others do not allow.
                    </Text>
                  </Group>
                )}
              </Stack>
          </Table.Td>
        </Table.Tr>
      )}
    </>
  );
}

export function PolicyMapTable({ data, onSelectIp }: PolicyMapTableProps) {
  const formatDateTime = useDateFormatter();
  const [filter, setFilter] = useState("");
  const [expandedIp, setExpandedIp] = useState<string | null>(null);

  const filtered = useMemo(() => {
    const q = filter.trim().toLowerCase();
    if (!q) return data.entries;
    return data.entries.filter(
      (e) =>
        e.ip.toLowerCase().includes(q) ||
        e.allowed_hosts.some((h) => h.toLowerCase().includes(q)),
    );
  }, [data.entries, filter]);

  function toggleRow(ip: string) {
    setExpandedIp((prev) => (prev === ip ? null : ip));
  }

  return (
    <Stack gap="sm">
      <Group justify="space-between" align="flex-end">
        <div>
          <Text size="xs" c="dimmed">
            Snapshot taken {formatDateTime(data.refreshed_at)} · computed in{" "}
            {data.refresh_duration_ms}ms
          </Text>
        </div>
        <TextInput
          placeholder="Filter by IP or host…"
          value={filter}
          onChange={(e) => setFilter(e.currentTarget.value)}
          size="sm"
          style={{ width: 260 }}
        />
      </Group>

      <Card withBorder p={0}>
        <Table.ScrollContainer minWidth={600}>
          <Table>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>IP</Table.Th>
                <Table.Th>Status</Table.Th>
                <Table.Th>Eff. hosts</Table.Th>
                <Table.Th>Contributors</Table.Th>
                <Table.Th />
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {filtered.length === 0 ? (
                <Table.Tr>
                  <Table.Td colSpan={5}>
                    <Text size="sm" c="dimmed" ta="center" py="xl">
                      {filter ? "No entries match the filter." : "No entries in the policy cache."}
                    </Text>
                  </Table.Td>
                </Table.Tr>
              ) : (
                filtered.map((entry) => (
                  <EntryRow
                    key={entry.ip}
                    entry={entry}
                    expanded={expandedIp === entry.ip}
                    onToggle={() => toggleRow(entry.ip)}
                    onSelectIp={onSelectIp}
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
