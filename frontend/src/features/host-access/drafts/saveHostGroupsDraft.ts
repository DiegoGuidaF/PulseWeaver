import type { GroupDetailWithUsers, GroupListItem, GroupWrite, Id } from "@/lib/api";
import type { DraftGroup, GroupsDraftState } from "./hostGroupsDraft";

/**
 * Reconcile is a full-state replace: any existing group id absent from the request body
 * gets deleted server-side. Groups whose real host membership was never loaded (not
 * visited this session) must not be sent with a placeholder empty hostIds — the caller
 * is expected to backfill `freshDetails` for every id returned by
 * `unvisitedExistingGroupIds` before building the body.
 */
export function buildReconcileGroupsBody(
  state: GroupsDraftState,
  freshDetails: Map<Id, GroupDetailWithUsers>,
): GroupWrite[] {
  return Array.from(state.draft.values()).map((g) => ({
    id: typeof g.id === "number" ? g.id : null,
    name: g.name,
    description: g.description ?? null,
    icon: g.icon ?? "",
    color: g.color,
    host_ids: resolveHostIds(g, state.visited, freshDetails),
  }));
}

function resolveHostIds(
  g: DraftGroup,
  visited: Set<Id>,
  freshDetails: Map<Id, GroupDetailWithUsers>,
): Id[] {
  if (typeof g.id !== "number" || visited.has(g.id)) return g.hostIds;
  return freshDetails.get(g.id)?.hosts.map((h) => h.id) ?? g.hostIds;
}

/** Existing group ids in the draft whose real host membership hasn't been loaded this session — fetch their detail before Save so the full-state reconcile body doesn't wipe their membership. */
export function unvisitedExistingGroupIds(state: GroupsDraftState): Id[] {
  return Array.from(state.draft.values())
    .map((g) => g.id)
    .filter((id): id is Id => typeof id === "number" && !state.visited.has(id));
}

export function groupsOriginalMatchesServer(
  original: Map<Id, GroupListItem>,
  current: GroupListItem[],
): boolean {
  if (original.size !== current.length) return false;
  return current.every((g) => original.has(g.id));
}
