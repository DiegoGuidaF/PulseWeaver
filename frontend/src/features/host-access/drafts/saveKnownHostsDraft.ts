import type { DesiredKnownHost, Id, KnownHostWithStats } from "@/lib/api";
import type { HostsDraftState } from "./knownHostsDraft";

export function buildReconcileHostsBody(state: HostsDraftState): DesiredKnownHost[] {
  // Tombstoned hosts are simply absent → backend deletes them.
  return Array.from(state.draft.values()).map((d) => ({
    id: typeof d.id === "number" ? d.id : null,
    fqdn: d.fqdn,
    icon: d.icon ?? null,
    group_ids: d.groupIds,
  }));
}

export function hostsOriginalMatchesServer(
  original: Map<Id, KnownHostWithStats>,
  current: KnownHostWithStats[],
): boolean {
  if (original.size !== current.length) return false;
  return current.every((h) => original.has(h.id));
}
