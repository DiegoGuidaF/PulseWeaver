import type { Host, HostInput, Id } from "@/lib/api";
import { sameGroupIds, type HostsDraftState } from "./knownHostsDraft";

export function buildReconcileHostsBody(state: HostsDraftState): HostInput[] {
  // Tombstoned hosts are simply absent → backend deletes them.
  return Array.from(state.draft.values()).map((d) => ({
    id: typeof d.id === "number" ? d.id : null,
    fqdn: d.fqdn,
    group_ids: d.groups.map((g) => g.id),
  }));
}

/**
 * Guards the full-state reconcile against clobbering concurrent edits: group
 * memberships changed externally since page load must reset the draft, not be
 * silently overwritten. FQDN is not compared — it is immutable server-side,
 * so id identity plus group membership covers every mutable field.
 */
export function hostsOriginalMatchesServer(
  original: Map<Id, Host>,
  current: Host[],
): boolean {
  if (original.size !== current.length) return false;
  return current.every((h) => {
    const o = original.get(h.id);
    return o !== undefined && sameGroupIds(o.groups, h.groups);
  });
}
