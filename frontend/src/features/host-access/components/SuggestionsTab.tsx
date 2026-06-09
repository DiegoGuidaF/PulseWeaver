import { useMemo, useState } from "react";
import {
  Button,
  Card,
  Group,
  Stack,
  Table,
  Text,
  Title,
  UnstyledButton,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { IconArrowDown, IconArrowUp, IconArrowsSort, IconRefresh } from "@tabler/icons-react";
import type { HostSuggestionsPage } from "@/lib/api";
import { useIgnoreSuggestion } from "@/features/host-access/hooks/useIgnoreSuggestion";
import { useUnignoreSuggestion } from "@/features/host-access/hooks/useUnignoreSuggestion";
import { useDateFormatter } from "@/contexts/useDateTimePrefs";
import { toErrorMessage } from "@/lib/api-client";

interface Props {
  data: HostSuggestionsPage;
  onRefresh: () => void;
  onStageHosts: (fqdns: string[]) => void;
}

type SortCol = "allowed_hits" | "denied_hits";
type SortDir = "asc" | "desc";

export function SuggestionsTab({ data, onRefresh, onStageHosts }: Props) {
  const formatDateTime = useDateFormatter();
  const ignoreSuggestion = useIgnoreSuggestion();
  const unignoreSuggestion = useUnignoreSuggestion();

  const [pendingFqdns, setPendingFqdns] = useState<Set<string>>(new Set());
  const [sort, setSort] = useState<{ col: SortCol; dir: SortDir } | null>(null);

  function toggleSort(col: SortCol) {
    setSort((prev) => {
      if (prev?.col === col) return prev.dir === "desc" ? null : { col, dir: "desc" };
      return { col, dir: "desc" };
    });
  }

  const sortedSuggestions = useMemo(() => {
    if (!sort) return data.suggestions;
    return [...data.suggestions].sort((a, b) =>
      sort.dir === "desc" ? b[sort.col] - a[sort.col] : a[sort.col] - b[sort.col],
    );
  }, [data.suggestions, sort]);

  function addPending(fqdn: string) {
    setPendingFqdns((prev) => new Set([...prev, fqdn]));
  }

  function removePending(fqdn: string) {
    setPendingFqdns((prev) => { const next = new Set(prev); next.delete(fqdn); return next; });
  }

  async function handleIgnore(fqdn: string) {
    addPending(fqdn);
    try {
      await ignoreSuggestion.mutateAsync({ body: { fqdn } });
      notifications.show({ color: "gray", message: `${fqdn} ignored` });
    } catch (err) {
      notifications.show({ color: "red", title: "Failed to ignore", message: toErrorMessage(err) });
    } finally {
      removePending(fqdn);
    }
  }

  async function handleUnignore(fqdn: string) {
    addPending(fqdn);
    try {
      await unignoreSuggestion.mutateAsync({ path: { fqdn } });
      notifications.show({ color: "green", message: `${fqdn} removed from ignore list` });
    } catch (err) {
      notifications.show({ color: "red", title: "Failed to unignore", message: toErrorMessage(err) });
    } finally {
      removePending(fqdn);
    }
  }

  if (data.suggestions.length === 0 && data.ignored.length === 0) {
    return (
      <Card withBorder>
        <Stack gap="md" align="center" py="xl">
          <Text fz={48}>🔍</Text>
          <Title order={3}>Nothing to review</Title>
          <Text c="dimmed" size="sm">
            No unknown hosts in recent traffic.
          </Text>
          <Button variant="subtle" leftSection={<IconRefresh size={14} />} onClick={onRefresh}>
            Refresh
          </Button>
        </Stack>
      </Card>
    );
  }

  return (
    <Stack gap="md">
      <Card withBorder padding="md">
        {sortedSuggestions.length === 0 ? (
          <Text size="sm" c="dimmed">
            No unknown hosts in recent traffic.
          </Text>
        ) : (<>


          <Table.ScrollContainer minWidth={600}>
            <Table>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th>Hostname</Table.Th>
                  <Table.Th>First seen</Table.Th>
                  <Table.Th>
                    <SortableHeader label="Allowed hits" col="allowed_hits" sort={sort} onToggle={toggleSort} />
                  </Table.Th>
                  <Table.Th>
                    <SortableHeader label="Denied hits" col="denied_hits" sort={sort} onToggle={toggleSort} />
                  </Table.Th>
                  <Table.Th style={{ textAlign: "right" }}>Action</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {sortedSuggestions.map((s) => {
                  const isPending = pendingFqdns.has(s.fqdn);
                  return (
                    <Table.Tr key={s.fqdn}>
                      <Table.Td>
                        <Text size="sm" fw={500} ff="monospace">{s.fqdn}</Text>
                      </Table.Td>
                      <Table.Td>
                        <Text size="sm" c="dimmed">{formatDateTime(s.first_seen)}</Text>
                      </Table.Td>
                      <Table.Td>
                        <Text
                          size="sm"
                          c={s.allowed_hits > 100 ? "green" : s.allowed_hits === 0 ? "dimmed" : undefined}
                          fw={s.allowed_hits > 100 ? 500 : 400}
                        >
                          {s.allowed_hits.toLocaleString()}
                        </Text>
                      </Table.Td>
                      <Table.Td>
                        <Text
                          size="sm"
                          c={s.denied_hits > 50 ? "var(--pw-amber-text)" : s.denied_hits === 0 ? "dimmed" : undefined}
                          fw={s.denied_hits > 50 ? 500 : 400}
                        >
                          {s.denied_hits.toLocaleString()}
                        </Text>
                      </Table.Td>
                      <Table.Td>
                        <Group gap="xs" justify="flex-end">
                          <Button
                            size="xs"
                            variant="outline"
                            onClick={() => handleIgnore(s.fqdn)}
                            loading={isPending}
                            disabled={isPending}
                          >
                            Ignore
                          </Button>
                          <Button
                            size="xs"
                            onClick={() => onStageHosts([s.fqdn])}
                            disabled={isPending}
                          >
                            Add to known
                          </Button>
                        </Group>
                      </Table.Td>
                    </Table.Tr>
                  );
                })}
              </Table.Tbody>
            </Table>
          </Table.ScrollContainer>
        </>)}
      </Card>

      {data.ignored.length > 0 && (
        <Card withBorder padding="md">
          <Text fw={600} mb={4}>
            Ignored
          </Text>
          <Text size="sm" c="dimmed" mb="md">
            Won't appear in suggestions again.
          </Text>
          <Table.ScrollContainer minWidth={400}>
            <Table>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th>Hostname</Table.Th>
                  <Table.Th>Ignored at</Table.Th>
                  <Table.Th />
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {data.ignored.map((s) => {
                  const isPending = pendingFqdns.has(s.fqdn);
                  return (
                    <Table.Tr key={s.fqdn}>
                      <Table.Td>
                        <Text size="sm" ff="monospace" c="dimmed">{s.fqdn}</Text>
                      </Table.Td>
                      <Table.Td>
                        <Text size="sm" c="dimmed">{formatDateTime(s.created_at)}</Text>
                      </Table.Td>
                      <Table.Td>
                        <Group justify="flex-end">
                          <Button
                            size="xs"
                            variant="subtle"
                            onClick={() => handleUnignore(s.fqdn)}
                            loading={isPending}
                            disabled={isPending}
                          >
                            Unignore
                          </Button>
                        </Group>
                      </Table.Td>
                    </Table.Tr>
                  );
                })}
              </Table.Tbody>
            </Table>
          </Table.ScrollContainer>
        </Card>
      )}
    </Stack>
  );
}

interface SortableHeaderProps {
  label: string;
  col: SortCol;
  sort: { col: SortCol; dir: SortDir } | null;
  onToggle: (col: SortCol) => void;
}

function SortableHeader({ label, col, sort, onToggle }: SortableHeaderProps) {
  const active = sort?.col === col;
  const Icon = active ? (sort!.dir === "desc" ? IconArrowDown : IconArrowUp) : IconArrowsSort;
  return (
    <UnstyledButton
      onClick={() => onToggle(col)}
      style={{ display: "flex", alignItems: "center", gap: 4, fontWeight: 600 }}
    >
      {label}
      <Icon size={13} stroke={active ? 2 : 1.2} style={{ opacity: active ? 1 : 0.4 }} />
    </UnstyledButton>
  );
}
