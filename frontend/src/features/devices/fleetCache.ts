import type { QueryClient } from "@tanstack/react-query";
import {
  getDeviceFleetQueryKey,
  listOwnerRefsQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";
import type { DevicePairingSummary, OwnerFleetGroup } from "@/lib/api";

/**
 * The fleet endpoint answers two screens from one shape: the unfiltered key backs
 * the `/devices` list, `?owner_id=` backs one owner's workspace. Keeping them as
 * distinct keys is what lets a workspace mutation refetch a single owner.
 */
export const fleetListKey = () => getDeviceFleetQueryKey();
export const fleetOwnerKey = (ownerId: number) =>
  getDeviceFleetQueryKey({ query: { owner_id: ownerId } });

/**
 * Both keys carry an owner's whole fleet, so re-reading either on navigation or on
 * window focus buys nothing a user would notice.
 */
export const FLEET_STALE_TIME = 30_000;

/** Invalidate one owner's fleet entry, leaving the list and every other owner alone. */
export function invalidateOwnerFleet(queryClient: QueryClient, ownerId: number) {
  queryClient.invalidateQueries({ queryKey: fleetOwnerKey(ownerId), exact: true });
}

/**
 * Invalidate the unfiltered fleet list, and only it.
 *
 * `exact` is load-bearing here: partial matching requires every field of the
 * filter to be present on the cached key, and the list key is an owner key minus
 * `query` — so it matches all of them. Without `exact` this refetches the whole
 * fleet, one request per cached owner.
 */
export function invalidateFleetList(queryClient: QueryClient) {
  queryClient.invalidateQueries({ queryKey: fleetListKey(), exact: true });
}

/**
 * Invalidate every fleet entry plus the ref list, after a change to who the
 * owners are or how they present.
 *
 * A group carries its owner's display name, role, bypass flag and host groups,
 * and the list and each workspace render them from their own cached copy. The
 * omitted `exact` is the point here: the list key partial-matches every
 * `?owner_id=` entry, so one call reaches all of them.
 */
export function invalidateOwnerIdentity(queryClient: QueryClient) {
  queryClient.invalidateQueries({ queryKey: fleetListKey() });
  queryClient.invalidateQueries({ queryKey: listOwnerRefsQueryKey() });
}

/**
 * Patch one owner's group into the cached fleet list, so a workspace refetch keeps
 * the list page current instead of forcing a whole-fleet reload.
 *
 * Only patches a list that is already cached — seeding the key from a single-owner
 * response would leave the list page rendering that owner as the entire fleet. The
 * list also keeps its own freshness timestamp: the other owners in it are no newer
 * than they were, and resetting the clock on every workspace visit would stop the
 * list ever refetching them.
 */
function samePairing(a: DevicePairingSummary | null, b: DevicePairingSummary | null) {
  if (a === null || b === null) return a === b;
  return a.status === b.status && a.expires_at === b.expires_at;
}

/**
 * Patch one device's pairing summary into the cached owner fleet.
 *
 * The fleet copy backs the pairing tab's dot, but only this app's own mutations
 * invalidate it — a code claimed on the end user's phone, or one that simply ran
 * out, moves the status with nothing here to notice. The pairing tab reads the
 * device's pairings directly and is therefore always the fresher of the two, so
 * it writes what it read back into the fleet instead of costing a second
 * round-trip. Writing only on a real change keeps this out of the render loop.
 */
export function patchDevicePairingSummary(
  queryClient: QueryClient,
  ownerId: number,
  deviceId: number,
  pairing: DevicePairingSummary | null,
) {
  queryClient.setQueryData<OwnerFleetGroup[]>(fleetOwnerKey(ownerId), (old) => {
    if (!old) return old;
    let changed = false;
    const next = old.map((group) => {
      let groupChanged = false;
      const devices = group.devices.map((device) => {
        if (device.id !== deviceId || samePairing(device.pairing ?? null, pairing)) return device;
        groupChanged = true;
        return { ...device, pairing };
      });
      if (!groupChanged) return group;
      changed = true;
      return { ...group, devices };
    });
    return changed ? next : old;
  });
}

export function spliceOwnerGroup(queryClient: QueryClient, group: OwnerFleetGroup) {
  const key = fleetListKey();
  const updatedAt = queryClient.getQueryState<OwnerFleetGroup[]>(key)?.dataUpdatedAt;
  queryClient.setQueryData<OwnerFleetGroup[]>(
    key,
    (old) => (old ? old.map((g) => (g.owner.id === group.owner.id ? group : g)) : old),
    { updatedAt },
  );
}
