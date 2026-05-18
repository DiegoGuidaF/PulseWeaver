import type { GroupDetailWithUsers, GroupWrite, Id } from "@/lib/api";
import type { GroupsDraftState } from "./hostGroupsDraft";

export function buildReconcileGroupsBody(state: GroupsDraftState): GroupWrite[] {
  // Tombstoned groups are simply absent → backend deletes them.
  return Array.from(state.draft.values()).map((g) => ({
    id: typeof g.id === "number" ? g.id : null,
    name: g.name,
    description: g.description ?? null,
    icon: g.icon ?? "",
    color: g.color ?? "",
    host_ids: g.hostIds,
  }));
}

export function groupsOriginalMatchesServer(
  original: Map<Id, GroupDetailWithUsers>,
  current: GroupDetailWithUsers[],
): boolean {
  if (original.size !== current.length) return false;
  return current.every((g) => original.has(g.id));
}
