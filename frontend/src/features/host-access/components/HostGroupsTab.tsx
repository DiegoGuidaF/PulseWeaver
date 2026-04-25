import React, { useMemo, useState } from "react";
import {
  Button,
  Card,
  Grid,
  Group,
  Modal,
  Stack,
  Text,
  Title,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { IconPlus } from "@tabler/icons-react";
import type { Id } from "@/lib/api";
import { useCreateHostGroup } from "@/features/host-access/hooks/useCreateHostGroup";
import { useUpdateHostGroup } from "@/features/host-access/hooks/useUpdateHostGroup";
import { useDeleteHostGroup } from "@/features/host-access/hooks/useDeleteHostGroup";
import { GroupMasterList } from "@/features/host-access/components/GroupMasterList";
import { GroupDetailPanel } from "@/features/host-access/components/GroupDetailPanel";
import { GroupMetadataModal } from "@/features/host-access/components/GroupMetadataModal";
import { StagedChangesBar } from "@/features/host-access/components/StagedChangesBar";
import {
  diffGroups,
  isDirtyGroups,
  type DraftGroup,
  type DraftGroupId,
  type GroupsDraftAction,
  type GroupsDraftState,
} from "@/features/host-access/drafts/hostGroupsDraft";
import {
  type DraftHost,
  type HostsDraftAction,
  type HostsDraftState,
} from "@/features/host-access/drafts/knownHostsDraft";
import { saveHostGroupsDraft } from "@/features/host-access/drafts/saveHostGroupsDraft";
import { toErrorMessage } from "@/lib/api-client";

interface Props {
  state: GroupsDraftState;
  dispatch: React.Dispatch<GroupsDraftAction>;
  hostsState: HostsDraftState;
  hostsDispatch: React.Dispatch<HostsDraftAction>;
}

export function HostGroupsTab({ state, dispatch, hostsState, hostsDispatch }: Props) {
  const createHostGroup = useCreateHostGroup();
  const updateHostGroup = useUpdateHostGroup();
  const deleteHostGroup = useDeleteHostGroup();

  const [createOpen, setCreateOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<DraftGroup | null>(null);
  const [saving, setSaving] = useState(false);

  const groups = useMemo(() => Array.from(state.draft.values()), [state]);
  const tombstoned = useMemo(
    () =>
      Array.from(state.tombstoned)
        .map((id) => state.original.get(id))
        .filter((g): g is NonNullable<typeof g> => g !== undefined),
    [state],
  );

  const selected = state.selectedId !== null ? state.draft.get(state.selectedId) ?? null : null;
  const tombstonedSelected =
    state.selectedId !== null && typeof state.selectedId === "number"
      ? state.tombstoned.has(state.selectedId)
      : false;
  const tombstonedAsDraft =
    tombstonedSelected && typeof state.selectedId === "number"
      ? toDraftFromOriginal(state, state.selectedId)
      : null;

  const diff = diffGroups(state);
  const dirty = isDirtyGroups(state);

  const existingNames = groups.map((g) => g.name);

  const hosts: DraftHost[] = useMemo(
    () => Array.from(hostsState.draft.values()),
    [hostsState],
  );

  function handleCreate(values: {
    name: string;
    description: string | null;
    icon: string | null;
    color: DraftGroup["color"];
  }) {
    const id: `new-${string}` = `new-${crypto.randomUUID()}`;
    dispatch({
      type: "add",
      id,
      group: { ...values, hostIds: [] },
    });
  }

  function handleEdit(values: {
    name: string;
    description: string | null;
    icon: string | null;
    color: DraftGroup["color"];
  }) {
    if (!selected) return;
    dispatch({ type: "update", id: selected.id, patch: values });
  }

  function handleConfirmDelete() {
    if (!deleteTarget) return;
    dispatch({ type: "remove", id: deleteTarget.id });
    setDeleteTarget(null);
  }

  function handleToggleHost(hostId: Id) {
    if (!selected) return;
    dispatch({ type: "toggleHost", id: selected.id, hostId });
    // Mirror the change on the host draft so both views stay in sync.
    const host = hostsState.draft.get(hostId);
    if (!host) return;
    const inGroup = host.groupIds.includes(typeof selected.id === "number" ? selected.id : -1);
    const groupIdNum = typeof selected.id === "number" ? selected.id : null;
    if (groupIdNum === null) return;
    const nextGroupIds = inGroup
      ? host.groupIds.filter((g) => g !== groupIdNum)
      : [...host.groupIds, groupIdNum];
    hostsDispatch({ type: "update", id: hostId, patch: { groupIds: nextGroupIds } });
  }

  async function handleSave() {
    setSaving(true);
    try {
      const result = await saveHostGroupsDraft(state, {
        createHostGroupAsync: (input) => createHostGroup.mutateAsync(input),
        updateHostGroupAsync: (input) => updateHostGroup.mutateAsync(input),
        deleteHostGroupAsync: (input) => deleteHostGroup.mutateAsync(input),
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

  if (groups.length === 0 && tombstoned.length === 0) {
    return (
      <>
        <Card withBorder>
          <Stack gap="md" align="center" py="xl">
            <Text fz={48}>🗂</Text>
            <Title order={3}>No groups yet</Title>
            <Text c="dimmed" size="sm" maw={440} ta="center">
              Bundle related hosts so you can grant access in one click. Groups are a UX
              convenience, not an authz concept.
            </Text>
            <Button leftSection={<IconPlus size={16} />} onClick={() => setCreateOpen(true)}>
              New group
            </Button>
          </Stack>
        </Card>
        <GroupMetadataModal
          opened={createOpen}
          onClose={() => setCreateOpen(false)}
          initial={null}
          existingNames={existingNames}
          onSubmit={handleCreate}
        />
        <StagedChangesBar
          visible={dirty}
          summary={summarizeGroups(diff)}
          saving={saving}
          onSave={handleSave}
          onDiscard={() => dispatch({ type: "discard" })}
        />
      </>
    );
  }

  return (
    <>
      <Modal
        opened={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        title="Stage group removal?"
        closeOnClickOutside={false}
        closeOnEscape={false}
        withCloseButton={false}
      >
        <Text size="sm">
          Mark{" "}
          <Text component="span" fw={600}>
            {deleteTarget?.name}
          </Text>{" "}
          for deletion? It will be removed when you save. New groups disappear immediately.
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

      <GroupMetadataModal
        opened={createOpen}
        onClose={() => setCreateOpen(false)}
        initial={null}
        existingNames={existingNames}
        onSubmit={handleCreate}
      />
      <GroupMetadataModal
        opened={editOpen}
        onClose={() => setEditOpen(false)}
        initial={selected}
        existingNames={existingNames}
        onSubmit={handleEdit}
      />

      <Grid>
        <Grid.Col span={{ base: 12, md: 4 }}>
          <GroupMasterList
            groups={groups}
            selectedId={state.selectedId}
            diff={diff}
            onSelect={(id) => dispatch({ type: "select", id })}
            onCreate={() => setCreateOpen(true)}
          />
        </Grid.Col>
        <Grid.Col span={{ base: 12, md: 8 }}>
          <GroupDetailPanel
            group={selected ?? tombstonedAsDraft}
            isTombstoned={tombstonedSelected}
            diff={diff}
            hosts={hosts}
            onEdit={() => setEditOpen(true)}
            onDelete={() => selected && setDeleteTarget(selected)}
            onRestore={() => {
              if (typeof state.selectedId === "number") {
                dispatch({ type: "restore", id: state.selectedId });
              }
            }}
            onToggleHost={handleToggleHost}
          />
        </Grid.Col>
      </Grid>

      <StagedChangesBar
        visible={dirty}
        summary={summarizeGroups(diff)}
        saving={saving}
        onSave={handleSave}
        onDiscard={() => dispatch({ type: "discard" })}
      />
    </>
  );
}

function toDraftFromOriginal(state: GroupsDraftState, id: Id): DraftGroup | null {
  const original = state.original.get(id);
  if (!original) return null;
  return {
    id,
    name: original.name,
    description: original.description ?? null,
    icon: original.icon ?? null,
    color: null,
    hostIds: original.hosts.map((h) => h.id),
  };
}

function summarizeGroups(diff: ReturnType<typeof diffGroups>): string {
  const parts: string[] = [];
  if (diff.added.length) parts.push(`${diff.added.length} added`);
  if (diff.removed.length) parts.push(`${diff.removed.length} removed`);
  if (diff.changed.length) parts.push(`${diff.changed.length} changed`);
  return parts.length === 0 ? "No staged changes" : parts.join(" · ");
}

// Discriminator for selectedId narrowing in the panel; not used externally.
export type _GroupsTabSelected = DraftGroupId | null;
