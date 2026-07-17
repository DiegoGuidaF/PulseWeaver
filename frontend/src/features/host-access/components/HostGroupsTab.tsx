import React, { useMemo, useState } from "react";
import {
  Button,
  Card,
  Grid,
  Stack,
  Text,
  Title,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { IconPlus } from "@tabler/icons-react";
import { useQueryClient } from "@tanstack/react-query";
import type { GroupDetailWithUsers, Host, Id } from "@/lib/api";
import { getHostGroupOptions, listHostGroupsOptions } from "@/lib/api/@tanstack/react-query.gen";
import { useReconcileHostGroups } from "@/features/host-access/hooks/useReconcileHostGroups";
import { GroupMasterList } from "@/features/host-access/components/GroupMasterList";
import { GroupDetailPanel } from "@/features/host-access/components/GroupDetailPanel";
import { GroupMetadataModal } from "@/features/host-access/components/GroupMetadataModal";
import { StagedChangesBar } from "@/features/host-access/components/StagedChangesBar";
import {
  diffGroups,
  isDirtyGroups,
  summarizeGroups,
  toDraftFromOriginal,
  type DraftGroup,
  type DraftGroupId,
  type GroupsDraftAction,
  type GroupsDraftState,
} from "@/features/host-access/drafts/hostGroupsDraft";
import {
  buildReconcileGroupsBody,
  groupsOriginalMatchesServer,
  unvisitedExistingGroupIds,
} from "@/features/host-access/drafts/saveHostGroupsDraft";
import { toErrorMessage } from "@/lib/api-client";

interface Props {
  state: GroupsDraftState;
  dispatch: React.Dispatch<GroupsDraftAction>;
  serverHosts: Host[];
  selectedDetailLoading?: boolean;
}

export function HostGroupsTab({ state, dispatch, serverHosts, selectedDetailLoading }: Props) {
  const queryClient = useQueryClient();
  const reconcileHostGroups = useReconcileHostGroups();

  const [createOpen, setCreateOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [saving, setSaving] = useState(false);

  const groups = useMemo(() => Array.from(state.draft.values()), [state]);
  const existingColors = useMemo(() => groups.map((g) => g.color), [groups]);
  const tombstonedDrafts = useMemo(
    () =>
      Array.from(state.tombstoned)
        .map((id) => toDraftFromOriginal(state, id))
        .filter((g): g is DraftGroup => g !== null),
    [state],
  );

  // Master-list badge count: unvisited groups only know host_count from the light list
  // (their draft.hostIds is a placeholder); visited/edited groups reflect the live draft.
  const hostCounts = useMemo(() => {
    const counts = new Map<DraftGroupId, number>();
    for (const g of groups) {
      if (typeof g.id === "number" && !state.visited.has(g.id)) {
        counts.set(g.id, state.listOriginal.get(g.id)?.host_count ?? g.hostIds.length);
      } else {
        counts.set(g.id, g.hostIds.length);
      }
    }
    return counts;
  }, [groups, state.visited, state.listOriginal]);

  const selected = state.selectedId !== null ? state.draft.get(state.selectedId) ?? null : null;
  const tombstonedSelected =
    state.selectedId !== null && typeof state.selectedId === "number"
      ? state.tombstoned.has(state.selectedId)
      : false;
  const tombstonedAsDraft = tombstonedSelected && typeof state.selectedId === "number"
    ? (tombstonedDrafts.find((g) => g.id === state.selectedId) ?? null)
    : null;

  // Resolve the server-side group for the access section (read-only users/policies)
  const selectedServerGroup: GroupDetailWithUsers | null =
    state.selectedId !== null && typeof state.selectedId === "number"
      ? (state.detailOriginal.get(state.selectedId) ?? null)
      : null;

  const diff = diffGroups(state);
  const dirty = isDirtyGroups(state);
  const existingNames = groups.map((g) => g.name);

  // Simple HostRef list for membership tables
  const hostRefs: { id: Id; fqdn: string }[] = useMemo(
    () => serverHosts.map((h) => ({ id: h.id, fqdn: h.fqdn })),
    [serverHosts],
  );

  function handleCreate(values: {
    name: string;
    description: string | null;
    icon: string | null;
    color: DraftGroup["color"];
  }) {
    const id: `new-${string}` = `new-${crypto.randomUUID()}`;
    dispatch({ type: "add", id, group: { ...values, hostIds: [] } });
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

  function handleToggleHost(hostId: Id) {
    if (!selected) return;
    dispatch({ type: "toggleHost", id: selected.id, hostId });
  }

  async function handleSave() {
    setSaving(true);
    try {
      const current = await queryClient.fetchQuery({ ...listHostGroupsOptions(), staleTime: 0 });
      if (!groupsOriginalMatchesServer(state.listOriginal, current.groups)) {
        notifications.show({
          color: "orange",
          title: "Server data changed",
          message: "The groups list was modified externally. Your draft has been reset.",
        });
        dispatch({ type: "reset", groups: current.groups });
        return;
      }

      // Reconcile is a full-state replace — any existing group not yet visited this
      // session needs its real host membership fetched first, or it would be sent
      // with an empty placeholder and wipe its members server-side.
      const unvisited = unvisitedExistingGroupIds(state);
      const freshDetails = new Map<Id, GroupDetailWithUsers>();
      if (unvisited.length > 0) {
        const details = await Promise.all(
          unvisited.map((id) =>
            queryClient.fetchQuery({
              ...getHostGroupOptions({ path: { group_id: id } }),
              staleTime: 0,
            }),
          ),
        );
        details.forEach((d) => freshDetails.set(d.id, d));
      }

      await reconcileHostGroups.mutateAsync({
        body: { groups: buildReconcileGroupsBody(state, freshDetails) },
      });
      notifications.show({ color: "green", message: "Groups saved" });
    } catch (err) {
      notifications.show({ color: "red", message: toErrorMessage(err) });
    } finally {
      setSaving(false);
    }
  }

  if (groups.length === 0 && tombstonedDrafts.length === 0) {
    return (
      <>
        <Card withBorder>
          <Stack gap="md" align="center" py="xl">
            <Text fz={48}>🗂</Text>
            <Title order={2}>No groups yet</Title>
            <Text c="dimmed" size="sm" maw={440} ta="center">
              Bundle related hosts so you can grant access in one click.
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
          existingColors={existingColors}
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
      <GroupMetadataModal
        opened={createOpen}
        onClose={() => setCreateOpen(false)}
        initial={null}
        existingNames={existingNames}
        existingColors={existingColors}
        onSubmit={handleCreate}
      />
      <GroupMetadataModal
        opened={editOpen}
        onClose={() => setEditOpen(false)}
        initial={selected}
        existingNames={existingNames}
        existingColors={existingColors}
        onSubmit={handleEdit}
      />

      <Grid>
        <Grid.Col span={{ base: 12, md: 4 }}>
          <GroupMasterList
            groups={groups}
            tombstoned={tombstonedDrafts}
            selectedId={state.selectedId}
            diff={diff}
            hostCounts={hostCounts}
            onSelect={(id) => dispatch({ type: "select", id })}
            onCreate={() => setCreateOpen(true)}
          />
        </Grid.Col>
        <Grid.Col span={{ base: 12, md: 8 }}>
          <GroupDetailPanel
            group={selected ?? tombstonedAsDraft}
            serverGroup={selectedServerGroup}
            isTombstoned={tombstonedSelected}
            detailLoading={selectedDetailLoading}
            hostCount={state.selectedId !== null ? hostCounts.get(state.selectedId) : undefined}
            diff={diff}
            hosts={hostRefs}
            onEdit={() => setEditOpen(true)}
            onDelete={() => selected && dispatch({ type: "remove", id: selected.id })}
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

export type _GroupsTabSelected = import("@/features/host-access/drafts/hostGroupsDraft").DraftGroupId | null;
