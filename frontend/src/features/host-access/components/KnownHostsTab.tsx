import React, { useMemo, useState } from "react";
import {
  ActionIcon,
  Badge,
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
  Tooltip,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import {
  IconArrowBackUp,
  IconPlus,
  IconSearch,
  IconTrash,
} from "@tabler/icons-react";
import type { Id, KnownHostWithStats, HostGroupWithMembers } from "@/lib/api";
import { useCreateKnownHosts } from "@/features/host-access/hooks/useCreateKnownHosts";
import { useUpdateKnownHost } from "@/features/host-access/hooks/useUpdateKnownHost";
import { useDeleteKnownHost } from "@/features/host-access/hooks/useDeleteKnownHost";
import { useUpdateHostGroup } from "@/features/host-access/hooks/useUpdateHostGroup";
import { AddHostModal } from "@/features/host-access/components/AddHostModal";
import { IconPicker } from "@/features/host-access/components/IconPicker";
import { GroupBadgeList } from "@/features/host-access/components/GroupBadgeList";
import { StagedChangesBar } from "@/features/host-access/components/StagedChangesBar";
import { resolveHostIcon } from "@/features/host-access/hostIconConfig";
import {
  diffHosts,
  isDirtyHosts,
  type DraftHost,
  type DraftHostId,
  type HostsDraftAction,
  type HostsDraftState,
} from "@/features/host-access/drafts/knownHostsDraft";
import { saveKnownHostsDraft } from "@/features/host-access/drafts/saveKnownHostsDraft";
import { toErrorMessage } from "@/lib/api-client";

interface Props {
  state: HostsDraftState;
  dispatch: React.Dispatch<HostsDraftAction>;
  serverGroups: HostGroupWithMembers[];
}

const UNGROUPED = "__ungrouped__";

export function KnownHostsTab({ state, dispatch, serverGroups }: Props) {
  const createKnownHosts = useCreateKnownHosts();
  const updateKnownHost = useUpdateKnownHost();
  const deleteKnownHost = useDeleteKnownHost();
  const updateHostGroup = useUpdateHostGroup();

  const [search, setSearch] = useState("");
  const [groupFilter, setGroupFilter] = useState<string[]>([]);
  const [addOpen, setAddOpen] = useState(false);
  const [iconTargetId, setIconTargetId] = useState<DraftHostId | null>(null);
  const [iconDraftValue, setIconDraftValue] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<DraftHost | null>(null);
  const [saving, setSaving] = useState(false);

  const drafts = useMemo(() => Array.from(state.draft.values()), [state]);
  const tombstoned = useMemo(
    () =>
      Array.from(state.tombstoned)
        .map((id) => state.original.get(id))
        .filter((h): h is KnownHostWithStats => h !== undefined),
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
        const groupsA = a.groupIds.map((id) => groupName(id, serverGroups, state.draft)).sort();
        const groupsB = b.groupIds.map((id) => groupName(id, serverGroups, state.draft)).sort();
        const ga = groupsA[0] ?? "￿";
        const gb = groupsB[0] ?? "￿";
        if (ga !== gb) return ga < gb ? -1 : 1;
        return a.fqdn.localeCompare(b.fqdn);
      }),
    [filtered, serverGroups, state.draft],
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

  function handleConfirmDelete() {
    if (!deleteTarget) return;
    dispatch({ type: "remove", id: deleteTarget.id });
    setDeleteTarget(null);
  }

  function handleAdd(values: { fqdn: string; icon: string | null; groupIds: Id[] }) {
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
      const result = await saveKnownHostsDraft(state, serverGroups, {
        createKnownHostsAsync: async (input) =>
          (await createKnownHosts.mutateAsync({ body: input.body })) ?? [],
        updateKnownHostAsync: (input) => updateKnownHost.mutateAsync(input),
        deleteKnownHostAsync: (input) => deleteKnownHost.mutateAsync(input),
        updateHostGroupAsync: (input) => updateHostGroup.mutateAsync(input),
      });
      if (result.failed.length === 0) {
        notifications.show({
          color: "green",
          message: `Saved ${result.succeeded} change${result.succeeded === 1 ? "" : "s"}`,
        });
      } else {
        notifications.show({
          color: "red",
          title: `Saved ${result.succeeded}, failed ${result.failed.length}`,
          message: result.failed.map((f) => `${f.label}: ${f.error}`).join("\n"),
        });
      }
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
          groups={Array.from(state.draft.values()).length > 0 ? [] : []}
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

  const draftGroups = Array.from(serverGroups).map((g) => ({
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

      <Modal
        opened={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        title="Stage host removal?"
        closeOnClickOutside={false}
        closeOnEscape={false}
        withCloseButton={false}
      >
        <Text size="sm">
          Mark{" "}
          <Text component="span" fw={600} ff="monospace">
            {deleteTarget?.fqdn}
          </Text>{" "}
          for removal? It will be deleted when you save.
        </Text>
        <Group justify="flex-end" mt="md" gap="xs">
          <Button variant="outline" onClick={() => setDeleteTarget(null)}>
            Cancel
          </Button>
          <Button color="red" onClick={handleConfirmDelete}>
            Stage delete
          </Button>
        </Group>
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
                      onDelete={() => setDeleteTarget(d)}
                    />
                  ))}
                  {tombstoned.map((h) => (
                    <TombstonedRow
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

interface HostRowProps {
  draft: DraftHost;
  diff: ReturnType<typeof diffHosts>;
  serverGroups: HostGroupWithMembers[];
  onIconClick: () => void;
  onDelete: () => void;
}

function HostRow({ draft, diff, serverGroups, onIconClick, onDelete }: HostRowProps) {
  const resolved = resolveHostIcon(draft.icon);
  const isNew = typeof draft.id !== "number";
  const isIconChanged = diff.iconChanged.some((d) => d.id === draft.id);
  const isGroupsChanged = diff.groupsChanged.some((d) => d.id === draft.id);
  const dirty = isNew || isIconChanged || isGroupsChanged;

  const groupRefs = draft.groupIds
    .map((id) => serverGroups.find((g) => g.id === id))
    .filter((g): g is HostGroupWithMembers => g !== undefined)
    .map((g) => ({ id: g.id, name: g.name, icon: g.icon ?? null }));

  return (
    <Table.Tr>
      <Table.Td>
        <Group gap="xs" wrap="nowrap">
          <Tooltip label="Change icon" withArrow>
            <ActionIcon
              variant="subtle"
              size="sm"
              color="gray"
              onClick={onIconClick}
              aria-label={`Change icon for ${draft.fqdn}`}
            >
              {resolved.kind === "tabler" ? (
                React.createElement(resolved.icon, { size: 14, stroke: 1.5 })
              ) : (
                <Text size="sm" span>
                  {resolved.value}
                </Text>
              )}
            </ActionIcon>
          </Tooltip>
          <Text size="sm" fw={500} ff="monospace">
            {draft.fqdn}
          </Text>
          {isNew && (
            <Badge size="xs" color="teal" variant="light">
              New
            </Badge>
          )}
          {dirty && !isNew && (
            <Badge size="xs" color="orange" variant="light">
              Edited
            </Badge>
          )}
        </Group>
      </Table.Td>
      <Table.Td>
        {groupRefs.length === 0 ? (
          <Text size="sm" c="dimmed">
            —
          </Text>
        ) : (
          <GroupBadgeList groups={groupRefs} />
        )}
      </Table.Td>
      <Table.Td>
        <UserCount draft={draft} />
      </Table.Td>
      <Table.Td>
        <Group gap={4} justify="flex-end">
          <Tooltip label="Stage delete" withArrow>
            <ActionIcon
              variant="subtle"
              color="red"
              size="sm"
              onClick={onDelete}
              aria-label={`Delete ${draft.fqdn}`}
            >
              <IconTrash size={14} stroke={1.5} />
            </ActionIcon>
          </Tooltip>
        </Group>
      </Table.Td>
    </Table.Tr>
  );
}

function UserCount({ draft }: { draft: DraftHost }) {
  // Server stat is only available for persisted hosts.
  const count = typeof draft.id === "number" ? null : null;
  return (
    <Text size="sm" c={count && count > 0 ? "indigo" : "dimmed"}>
      {count == null ? "—" : `${count} ${count === 1 ? "user" : "users"}`}
    </Text>
  );
}

interface TombstonedRowProps {
  host: KnownHostWithStats;
  onRestore: () => void;
}

function TombstonedRow({ host, onRestore }: TombstonedRowProps) {
  return (
    <Table.Tr style={{ opacity: 0.55 }}>
      <Table.Td>
        <Group gap="xs" wrap="nowrap">
          <Text size="sm" fw={500} ff="monospace" td="line-through">
            {host.fqdn}
          </Text>
          <Badge size="xs" color="red" variant="light">
            Will delete
          </Badge>
        </Group>
      </Table.Td>
      <Table.Td colSpan={2}>
        <Text size="xs" c="dimmed">
          {host.user_count} {host.user_count === 1 ? "user" : "users"} will lose access
        </Text>
      </Table.Td>
      <Table.Td>
        <Group gap={4} justify="flex-end">
          <Tooltip label="Undo delete" withArrow>
            <ActionIcon
              variant="subtle"
              size="sm"
              onClick={onRestore}
              aria-label={`Restore ${host.fqdn}`}
            >
              <IconArrowBackUp size={14} stroke={1.5} />
            </ActionIcon>
          </Tooltip>
        </Group>
      </Table.Td>
    </Table.Tr>
  );
}

function summarizeHosts(diff: ReturnType<typeof diffHosts>): string {
  const parts: string[] = [];
  if (diff.added.length) parts.push(`${diff.added.length} added`);
  if (diff.removed.length) parts.push(`${diff.removed.length} removed`);
  if (diff.iconChanged.length) parts.push(`${diff.iconChanged.length} icon`);
  if (diff.groupsChanged.length) parts.push(`${diff.groupsChanged.length} re-grouped`);
  return parts.length === 0 ? "No staged changes" : parts.join(" · ");
}

function hostUserImpact(diff: ReturnType<typeof diffHosts>): string | null {
  const total = diff.removed.reduce((acc, h) => acc + h.user_count, 0);
  if (total === 0) return null;
  return `${total} user${total === 1 ? "" : "s"} will lose access on save`;
}

function groupName(
  id: Id,
  serverGroups: HostGroupWithMembers[],
  drafts: HostsDraftState["draft"],
): string {
  const fromServer = serverGroups.find((g) => g.id === id);
  if (fromServer) return fromServer.name;
  // Drafts here are hosts not groups, so we can't resolve unsaved group names — fall back.
  void drafts;
  return "";
}
