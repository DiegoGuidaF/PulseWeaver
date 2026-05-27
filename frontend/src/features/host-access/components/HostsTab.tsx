import React, { useMemo, useState } from "react";
import {
  Button,
  Card,
  Group,
  MultiSelect,
  Pagination,
  Stack,
  Table,
  Text,
  TextInput,
  Title,
  UnstyledButton,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { IconArrowDown, IconArrowUp, IconArrowsSort, IconPlus, IconSearch } from "@tabler/icons-react";
import { useQueryClient } from "@tanstack/react-query";
import type { GroupDetailWithUsers } from "@/lib/api";
import { listHostsOptions } from "@/lib/api/@tanstack/react-query.gen";
import { useReconcileHosts } from "@/features/host-access/hooks/useReconcileHosts";
import { AddHostModal } from "@/features/host-access/components/AddHostModal";
import { StagedChangesBar } from "@/features/host-access/components/StagedChangesBar";
import { HostRow } from "@/features/host-access/components/HostRow";
import { TombstonedHostRow } from "@/features/host-access/components/TombstonedHostRow";
import {
  diffHosts,
  isDirtyHosts,
  summarizeHosts,
  type DraftHost,
  type HostsDraftAction,
  type HostsDraftState,
} from "@/features/host-access/drafts/knownHostsDraft";
import {
  buildReconcileHostsBody,
  hostsOriginalMatchesServer,
} from "@/features/host-access/drafts/saveKnownHostsDraft";
import { toErrorMessage } from "@/lib/api-client";

interface Props {
  state: HostsDraftState;
  dispatch: React.Dispatch<HostsDraftAction>;
  serverGroups: GroupDetailWithUsers[];
}

const UNGROUPED = "__ungrouped__";
const PAGE_SIZE = 25;

type SortCol = "fqdn" | "groups";
type SortDir = "asc" | "desc";

export function HostsTab({ state, dispatch, serverGroups }: Props) {
  const queryClient = useQueryClient();
  const reconcileHosts = useReconcileHosts();

  const [search, setSearch] = useState("");
  const [groupFilter, setGroupFilter] = useState<string[]>([]);
  const [addOpen, setAddOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [page, setPage] = useState(1);
  const [sort, setSort] = useState<{ col: SortCol; dir: SortDir } | null>(null);

  const drafts = useMemo(() => Array.from(state.draft.values()), [state]);
  const tombstoned = useMemo(
    () =>
      Array.from(state.tombstoned)
        .map((id) => state.original.get(id))
        .filter((h): h is NonNullable<typeof h> => h !== undefined),
    [state],
  );

  const filtered = useMemo(() => {
    const term = search.toLowerCase().trim();
    return drafts.filter((d) => {
      if (term && !d.fqdn.toLowerCase().includes(term)) return false;
      if (groupFilter.length === 0) return true;
      if (groupFilter.includes(UNGROUPED) && d.groupIds.length === 0) return true;
      return d.groupIds.some((id) => groupFilter.includes(String(id)));
    });
  }, [drafts, search, groupFilter]);

  const sorted = useMemo(() => {
    const arr = [...filtered];
    if (!sort) {
      return arr.sort((a, b) => {
        const ga = firstGroupName(a, serverGroups);
        const gb = firstGroupName(b, serverGroups);
        if (ga !== gb) return ga < gb ? -1 : 1;
        return a.fqdn.localeCompare(b.fqdn);
      });
    }
    if (sort.col === "fqdn") {
      arr.sort((a, b) => {
        const cmp = a.fqdn.localeCompare(b.fqdn);
        return sort.dir === "asc" ? cmp : -cmp;
      });
    } else {
      arr.sort((a, b) => {
        const ga = firstGroupName(a, serverGroups);
        const gb = firstGroupName(b, serverGroups);
        const cmp = ga < gb ? -1 : ga > gb ? 1 : a.fqdn.localeCompare(b.fqdn);
        return sort.dir === "asc" ? cmp : -cmp;
      });
    }
    return arr;
  }, [filtered, sort, serverGroups]);

  const totalPages = Math.max(1, Math.ceil(sorted.length / PAGE_SIZE));
  const currentPage = Math.min(page, totalPages);
  const paginated = useMemo(
    () => sorted.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE),
    [sorted, currentPage],
  );

  function toggleSort(col: SortCol) {
    setSort((prev) => {
      if (prev?.col === col) return prev.dir === "asc" ? { col, dir: "desc" } : null;
      return { col, dir: "asc" };
    });
    setPage(1);
  }

  function handleGroupClick(groupId: number) {
    const key = String(groupId);
    setGroupFilter((prev) =>
      prev.includes(key) ? prev.filter((x) => x !== key) : [...prev, key],
    );
    setPage(1);
  }

  const diff = diffHosts(state);
  const dirty = isDirtyHosts(state);
  const existingFqdns = drafts.map((d) => d.fqdn);

  const groupSelectOptions = [
    { value: UNGROUPED, label: "No group (unassigned)" },
    ...serverGroups.map((g) => ({ value: String(g.id), label: g.name })),
  ];

  const addModalGroups = serverGroups.map((g) => ({ id: g.id, name: g.name }));

  function handleAdd(values: { fqdn: string; groupIds: number[] }) {
    const id: `new-${string}` = `new-${crypto.randomUUID()}`;
    dispatch({ type: "add", id, host: { fqdn: values.fqdn, groupIds: values.groupIds } });
  }

  async function handleSave() {
    setSaving(true);
    try {
      const current = await queryClient.fetchQuery({ ...listHostsOptions(), staleTime: 0 });
      if (!hostsOriginalMatchesServer(state.original, current.hosts)) {
        notifications.show({
          color: "orange",
          title: "Server data changed",
          message: "The hosts list was modified externally. Your draft has been reset.",
        });
        dispatch({ type: "reset", hosts: current.hosts });
        return;
      }

      await reconcileHosts.mutateAsync({ body: { hosts: buildReconcileHostsBody(state) } });
      notifications.show({ color: "green", message: "Hosts saved" });
    } catch (err) {
      notifications.show({ color: "red", message: toErrorMessage(err) });
    } finally {
      setSaving(false);
    }
  }

  if (drafts.length === 0 && tombstoned.length === 0) {
    return (
      <>
        <Card withBorder>
          <Stack gap="md" align="center" py="xl">
            <Text fz={48}>📡</Text>
            <Title order={2}>No hosts yet</Title>
            <Text c="dimmed" size="sm" maw={440} ta="center">
              Stage one or more hosts; nothing is sent to the server until you click Save.
            </Text>
            <Button leftSection={<IconPlus size={16} />} onClick={() => setAddOpen(true)}>
              Add host
            </Button>
          </Stack>
        </Card>
        <AddHostModal
          opened={addOpen}
          onClose={() => setAddOpen(false)}
          groups={addModalGroups}
          existingFqdns={existingFqdns}
          onSubmit={handleAdd}
        />
        <StagedChangesBar
          visible={dirty}
          summary={summarizeHosts(diff)}
          saving={saving}
          onSave={handleSave}
          onDiscard={() => dispatch({ type: "discard" })}
        />
      </>
    );
  }

  return (
    <>
      <AddHostModal
        opened={addOpen}
        onClose={() => setAddOpen(false)}
        groups={addModalGroups}
        existingFqdns={existingFqdns}
        onSubmit={handleAdd}
      />

      <Card withBorder>
        <Group justify="space-between" mb="sm" wrap="nowrap">
          <Group gap="xs" wrap="nowrap">
            <TextInput
              placeholder="Search hosts…"
              value={search}
              onChange={(e) => { setSearch(e.currentTarget.value); setPage(1); }}
              leftSection={<IconSearch size={14} />}
              w={240}
            />
            <MultiSelect
              placeholder="Filter by group"
              data={groupSelectOptions}
              value={groupFilter}
              onChange={(v) => { setGroupFilter(v); setPage(1); }}
              clearable
              searchable
              w={280}
            />
          </Group>
          <Button
            size="xs"
            leftSection={<IconPlus size={14} />}
            onClick={() => setAddOpen(true)}
          >
            New host
          </Button>
        </Group>

        <Table.ScrollContainer minWidth={480}>
          <Table verticalSpacing="xs">
            <Table.Thead>
              <Table.Tr>
                <Table.Th>
                  <SortableHeader label="Hostname" col="fqdn" sort={sort} onToggle={toggleSort} />
                </Table.Th>
                <Table.Th>
                  <SortableHeader label="Groups" col="groups" sort={sort} onToggle={toggleSort} />
                </Table.Th>
                <Table.Th aria-label="Actions" />
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {paginated.length === 0 && tombstoned.length === 0 ? (
                <Table.Tr>
                  <Table.Td colSpan={3}>
                    <Text size="sm" c="dimmed" ta="center" py="md">
                      No hosts match the current filter.
                    </Text>
                  </Table.Td>
                </Table.Tr>
              ) : (
                <>
                  {paginated.map((d) => (
                    <HostRow
                      key={String(d.id)}
                      draft={d}
                      diff={diff}
                      serverGroups={serverGroups}
                      onGroupClick={handleGroupClick}
                      onDelete={() => dispatch({ type: "remove", id: d.id })}
                    />
                  ))}
                  {page === totalPages &&
                    tombstoned.map((h) => (
                      <TombstonedHostRow
                        key={`tomb-${h.id}`}
                        host={h}
                        onRestore={() => dispatch({ type: "restore", id: h.id })}
                      />
                    ))}
                </>
              )}
            </Table.Tbody>
          </Table>
        </Table.ScrollContainer>

        {totalPages > 1 && (
          <Group justify="center" mt="sm">
            <Pagination
              value={currentPage}
              onChange={setPage}
              total={totalPages}
              size="sm"
            />
          </Group>
        )}
      </Card>

      <StagedChangesBar
        visible={dirty}
        summary={summarizeHosts(diff)}
        saving={saving}
        onSave={handleSave}
        onDiscard={() => dispatch({ type: "discard" })}
      />
    </>
  );
}

function firstGroupName(d: DraftHost, serverGroups: GroupDetailWithUsers[]): string {
  if (d.groupIds.length === 0) return "￿"; // sorts unassigned to end by default
  return d.groupIds
    .map((id) => serverGroups.find((g) => g.id === id)?.name ?? "")
    .sort()[0] ?? "￿";
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
