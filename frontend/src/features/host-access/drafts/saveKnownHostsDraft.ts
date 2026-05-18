import type { Host, HostInput, Id } from "@/lib/api";
import type { HostsDraftState } from "./knownHostsDraft";

export function buildReconcileHostsBody(state: HostsDraftState): HostInput[] {
  // Tombstoned hosts are simply absent → backend deletes them.
  return Array.from(state.draft.values()).map((d) => ({
    id: typeof d.id === "number" ? d.id : null,
    fqdn: d.fqdn,
    group_ids: d.groupIds,
  }));
}

export function hostsOriginalMatchesServer(
  original: Map<Id, Host>,
  current: Host[],
): boolean {
  if (original.size !== current.length) return false;
  return current.every((h) => original.has(h.id));
}
