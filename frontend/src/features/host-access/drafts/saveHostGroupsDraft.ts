import type { DesiredHostGroup, HostGroupWithMembers, Id } from "@/lib/api";
import type { GroupsDraftState } from "./hostGroupsDraft";

export function buildReconcileGroupsBody(state: GroupsDraftState): DesiredHostGroup[] {
  // Tombstoned groups are simply absent → backend deletes them.
  return Array.from(state.draft.values()).map((g) => ({
    id: typeof g.id === "number" ? g.id : null,
    name: g.name,
    description: g.description ?? null,
    icon: g.icon ?? null,
    color: g.color ?? null,
    host_ids: g.hostIds,
  }));
}

export function groupsOriginalMatchesServer(
  original: Map<Id, HostGroupWithMembers>,
  current: HostGroupWithMembers[],
): boolean {
  if (original.size !== current.length) return false;
  return current.every((g) => original.has(g.id));
}
