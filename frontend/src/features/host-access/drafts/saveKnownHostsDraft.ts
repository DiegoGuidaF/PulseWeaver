import type {
  HostGroupWithMembers,
  Id,
  KnownHost,
  KnownHostWithStats,
} from "@/lib/api";
import type { HostsDraftState } from "./knownHostsDraft";
import { diffHosts } from "./knownHostsDraft";

export interface SaveResultEntry {
  label: string;
  error: string;
}

export interface SaveResult {
  succeeded: number;
  failed: SaveResultEntry[];
}

export interface SaveHostsDeps {
  createKnownHostsAsync: (input: {
    body: { fqdns: string[] };
  }) => Promise<KnownHost[]>;
  updateKnownHostAsync: (input: {
    path: { host_id: Id };
    body: { icon?: string | null };
  }) => Promise<unknown>;
  deleteKnownHostAsync: (input: { path: { host_id: Id } }) => Promise<unknown>;
  updateHostGroupAsync: (input: {
    path: { group_id: Id };
    body: {
      name: string;
      description?: string | null;
      icon?: string | null;
      host_ids?: Id[];
    };
  }) => Promise<unknown>;
}

// Hosts can't have their groups set in one call — the API only exposes group→hosts.
// To realise host-side group membership changes we patch each affected group's host_ids.
// If the user has unsaved edits in the groups tab, those are clobbered after save: the
// cache invalidation triggers a reset there. Tabs are saved independently by design.
export async function saveKnownHostsDraft(
  state: HostsDraftState,
  serverGroups: HostGroupWithMembers[],
  deps: SaveHostsDeps,
): Promise<SaveResult> {
  const diff = diffHosts(state);
  const failed: SaveResultEntry[] = [];
  let succeeded = 0;

  // Step 1 — create new hosts in one batch so we get the persisted IDs back.
  const newDrafts = diff.added;
  const draftIdToRealId = new Map<string, Id>();
  if (newDrafts.length > 0) {
    try {
      const created = await deps.createKnownHostsAsync({
        body: { fqdns: newDrafts.map((d) => d.fqdn) },
      });
      // Created order matches request order (server contract).
      created.forEach((host, i) => {
        const draft = newDrafts[i];
        if (draft && typeof draft.id === "string") {
          draftIdToRealId.set(draft.id, host.id);
        }
      });
      succeeded += created.length;
    } catch (err) {
      failed.push({
        label: `Create ${newDrafts.length} host${newDrafts.length === 1 ? "" : "s"}`,
        error: errorMessage(err),
      });
    }
  }

  // Step 2 — apply icons on freshly-created hosts that asked for one.
  const newIconUpdates: Promise<void>[] = [];
  for (const draft of newDrafts) {
    if (typeof draft.id !== "string") continue;
    const realId = draftIdToRealId.get(draft.id);
    if (!realId || draft.icon == null) continue;
    newIconUpdates.push(
      deps
        .updateKnownHostAsync({ path: { host_id: realId }, body: { icon: draft.icon } })
        .then(
          () => {
            succeeded += 1;
          },
          (err: unknown) => {
            failed.push({ label: `Set icon on ${draft.fqdn}`, error: errorMessage(err) });
          },
        ),
    );
  }

  // Step 3 — icon changes on persisted hosts.
  const persistedIconUpdates = diff.iconChanged.map((draft) =>
    deps
      .updateKnownHostAsync({
        path: { host_id: draft.id as Id },
        body: { icon: draft.icon },
      })
      .then(
        () => {
          succeeded += 1;
        },
        (err: unknown) => {
          failed.push({ label: `Update ${draft.fqdn}`, error: errorMessage(err) });
        },
      ),
  );

  // Step 4 — deletions.
  const deletes = diff.removed.map((host) =>
    deps.deleteKnownHostAsync({ path: { host_id: host.id } }).then(
      () => {
        succeeded += 1;
      },
      (err: unknown) => {
        failed.push({ label: `Delete ${host.fqdn}`, error: errorMessage(err) });
      },
    ),
  );

  await Promise.all([...newIconUpdates, ...persistedIconUpdates, ...deletes]);

  // Step 5 — propagate host→group membership through group updates.
  const targetMembership = computeTargetGroupMembership(state, serverGroups, draftIdToRealId);
  const groupUpdates: Promise<void>[] = [];
  for (const [groupId, hostIds] of targetMembership) {
    const group = serverGroups.find((g) => g.id === groupId);
    if (!group) continue;
    const currentIds = new Set(group.hosts.map((h) => h.id));
    const targetIds = new Set(hostIds);
    if (sameSet(currentIds, targetIds)) continue;
    groupUpdates.push(
      deps
        .updateHostGroupAsync({
          path: { group_id: groupId },
          body: {
            name: group.name,
            description: group.description ?? null,
            icon: group.icon ?? null,
            host_ids: hostIds,
          },
        })
        .then(
          () => {
            succeeded += 1;
          },
          (err: unknown) => {
            failed.push({ label: `Update group ${group.name}`, error: errorMessage(err) });
          },
        ),
    );
  }
  await Promise.all(groupUpdates);

  return { succeeded, failed };
}

function computeTargetGroupMembership(
  state: HostsDraftState,
  serverGroups: HostGroupWithMembers[],
  draftIdToRealId: Map<string, Id>,
): Map<Id, Id[]> {
  const target = new Map<Id, Set<Id>>();
  for (const g of serverGroups) {
    target.set(g.id, new Set(g.hosts.map((h) => h.id)));
  }

  const tombstoned = state.tombstoned;
  for (const draft of state.draft.values()) {
    const realId =
      typeof draft.id === "number" ? draft.id : draftIdToRealId.get(draft.id);
    if (!realId) continue;
    const draftGroups = new Set(draft.groupIds);
    for (const [gid, members] of target) {
      if (draftGroups.has(gid)) members.add(realId);
      else members.delete(realId);
    }
  }
  for (const id of tombstoned) {
    for (const members of target.values()) members.delete(id);
  }

  // Only return groups whose membership changed vs. server.
  const result = new Map<Id, Id[]>();
  for (const g of serverGroups) {
    const wanted = target.get(g.id) ?? new Set<Id>();
    const current = new Set(g.hosts.map((h) => h.id));
    if (!sameSet(current, wanted)) {
      result.set(g.id, [...wanted].sort((a, b) => a - b));
    }
  }
  return result;
}

function sameSet(a: Set<Id>, b: Set<Id>): boolean {
  if (a.size !== b.size) return false;
  for (const v of a) if (!b.has(v)) return false;
  return true;
}

function errorMessage(err: unknown): string {
  if (err instanceof Error) return err.message;
  return String(err);
}

// Keep KnownHostWithStats import alive for callers needing the cache-shape.
export type { KnownHostWithStats };
