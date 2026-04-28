import React, { useMemo, useState } from "react";
import {
  Button,
  Card,
  Group,
  Modal,
  MultiSelect,
  Stack,
  Table,
  Text,
  TextInput,
  Title,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { IconPlus, IconSearch } from "@tabler/icons-react";
import { useQueryClient } from "@tanstack/react-query";
import type { HostGroupWithMembers } from "@/lib/api";
import { listKnownHostsOptions } from "@/lib/api/@tanstack/react-query.gen";
import { useReconcileKnownHosts } from "@/features/host-access/hooks/useReconcileKnownHosts";
import { AddHostModal } from "@/features/host-access/components/AddHostModal";
import { IconPicker } from "@/features/host-access/components/IconPicker";
import { StagedChangesBar } from "@/features/host-access/components/StagedChangesBar";
import { HostRow } from "@/features/host-access/components/HostRow";
import { TombstonedHostRow } from "@/features/host-access/components/TombstonedHostRow";
import { TabLockAlert } from "@/features/host-access/components/TabLockAlert";
import {
  diffHosts,
  isDirtyHosts,
  summarizeHosts,
  hostUserImpact,
  type DraftHost,
  type DraftHostId,
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
  serverGroups: HostGroupWithMembers[];
  locked: boolean;
  onDiscardLock: () => void;
}

const UNGROUPED = "__ungrouped__";

export function KnownHostsTab({ state, dispatch, serverGroups, locked, onDiscardLock }: Props) {
  const queryClient = useQueryClient();
  const reconcileKnownHosts = useReconcileKnownHosts();

  const [search, setSearch] = useState("");
  const [groupFilter, setGroupFilter] = useState<string[]>([]);
  const [addOpen, setAddOpen] = useState(false);
  const [iconTargetId, setIconTargetId] = useState<DraftHostId | null>(null);
  const [iconDraftValue, setIconDraftValue] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

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

  const sorted = useMemo(
    () =>
      [...filtered].sort((a, b) => {
        const ga = a.groupIds.map((id) => groupName(id, serverGroups)).sort()[0] ?? "￿";
        const gb = b.groupIds.map((id) => groupName(id, serverGroups)).sort()[0] ?? "￿";
        if (ga !== gb) return ga < gb ? -1 : 1;
        return a.fqdn.localeCompare(b.fqdn);
      }),
    [filtered, serverGroups],
  );

  const diff = diffHosts(state);
  const dirty = isDirtyHosts(state);
  const existingFqdns = drafts.map((d) => d.fqdn);

  const groupSelectOptions = [
    { value: UNGROUPED, label: "No group" },
    ...serverGroups.map((g) => ({ value: String(g.id), label: g.name })),
  ];

  function handleStartIconEdit(host: DraftHost) {
    setIconTargetId(host.id);
    setIconDraftValue(host.icon);
  }

  function handleApplyIcon() {
    if (iconTargetId === null) return;
    dispatch({ type: "update", id: iconTargetId, patch: { icon: iconDraftValue } });
    setIconTargetId(null);
    setIconDraftValue(null);
  }

  function handleAdd(values: { fqdn: string; icon: string | null; groupIds: number[] }) {
    const id: `new-${string}` = `new-${crypto.randomUUID()}`;
    dispatch({
      type: "add",
      id,
      host: { fqdn: values.fqdn, icon: values.icon, groupIds: values.groupIds },
    });
  }

  async function handleSave() {
    setSaving(true);
    try {
      const current = await queryClient.fetchQuery({
        ...listKnownHostsOptions(),
        staleTime: 0,
      });
      if (!hostsOriginalMatchesServer(state.original, current)) {
        notifications.show({
          color: "orange",
          title: "Server data changed",
          message: "The hosts list was modified externally. Your draft has been reset.",
        });
        dispatch({ type: "reset", hosts: current });
        return;
      }

      await reconcileKnownHosts.mutateAsync({ body: { hosts: buildReconcileHostsBody(state) } });
      notifications.show({ color: "green", message: "Hosts saved" });
    } catch (err) {
      notifications.show({ color: "red", message: toErrorMessage(err) });
    } finally {
      setSaving(false);
    }
  }

  if (locked) {
    return (
      <TabLockAlert
        title="Groups tab has unsaved changes"
        message="Save or discard your group changes before editing known hosts."
        discardLabel="Discard group changes"
        onDiscard={onDiscardLock}
      />
    );
  }

  if (drafts.length === 0 && tombstoned.length === 0) {
    return (
      <>
        <Card withBorder>
          <Stack gap="md" align="center" py="xl">
            <Text fz={48}>📡</Text>
            <Title order={3}>No known hosts yet</Title>
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
          groups={[]}
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

  const draftGroups = serverGroups.map((g) => ({
    id: g.id,
    name: g.name,
    icon: g.icon ?? null,
    description: g.description ?? null,
    color: null,
    hostIds: g.hosts.map((h) => h.id),
  }));

  return (
    <>
      <Modal
        opened={iconTargetId !== null}
        onClose={() => setIconTargetId(null)}
        title="Change host icon"
        size="md"
      >
        <Stack gap="md">
          <IconPicker value={iconDraftValue} onChange={setIconDraftValue} />
          <Group justify="flex-end" gap="xs">
            <Button variant="outline" onClick={() => setIconTargetId(null)}>
              Cancel
            </Button>
            <Button onClick={handleApplyIcon}>Apply</Button>
          </Group>
        </Stack>
      </Modal>

      <AddHostModal
        opened={addOpen}
        onClose={() => setAddOpen(false)}
        groups={draftGroups}
        existingFqdns={existingFqdns}
        onSubmit={handleAdd}
      />

      <Card withBorder>
        <Group justify="space-between" mb="sm" wrap="nowrap">
          <Group gap="xs" wrap="nowrap">
            <TextInput
              placeholder="Search hosts…"
              value={search}
              onChange={(e) => setSearch(e.currentTarget.value)}
              leftSection={<IconSearch size={14} />}
              w={240}
            />
            <MultiSelect
              placeholder="Filter by group"
              data={groupSelectOptions}
              value={groupFilter}
              onChange={setGroupFilter}
              clearable
              searchable
              w={260}
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

        <Table.ScrollContainer minWidth={560}>
          <Table verticalSpacing="xs">
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Hostname</Table.Th>
                <Table.Th>Groups</Table.Th>
                <Table.Th>Users</Table.Th>
                <Table.Th />
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {sorted.length === 0 && tombstoned.length === 0 ? (
                <Table.Tr>
                  <Table.Td colSpan={4}>
                    <Text size="sm" c="dimmed" ta="center" py="md">
                      No hosts match the current filter.
                    </Text>
                  </Table.Td>
                </Table.Tr>
              ) : (
                <>
                  {sorted.map((d) => (
                    <HostRow
                      key={String(d.id)}
                      draft={d}
                      diff={diff}
                      serverGroups={serverGroups}
                      onIconClick={() => handleStartIconEdit(d)}
                      onDelete={() => dispatch({ type: "remove", id: d.id })}
                    />
                  ))}
                  {tombstoned.map((h) => (
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
      </Card>

      <StagedChangesBar
        visible={dirty}
        summary={summarizeHosts(diff)}
        detail={hostUserImpact(diff)}
        saving={saving}
        onSave={handleSave}
        onDiscard={() => dispatch({ type: "discard" })}
      />
    </>
  );
}

function groupName(id: number, serverGroups: HostGroupWithMembers[]): string {
  return serverGroups.find((g) => g.id === id)?.name ?? "";
}
