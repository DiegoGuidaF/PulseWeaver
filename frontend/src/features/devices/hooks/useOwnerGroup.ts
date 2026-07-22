import { useEffect } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { getDeviceFleetOptions } from "@/lib/api/@tanstack/react-query.gen";
import { FLEET_STALE_TIME, spliceOwnerGroup } from "../fleetCache";

/**
 * One owner's workspace data — the same group the list page renders, narrowed to
 * this owner. An owner id that resolves to nothing yields an empty array, so
 * `data` is undefined for both "not found" and "not loaded yet".
 */
export function useOwnerGroup(ownerId: number) {
  const queryClient = useQueryClient();

  const query = useQuery({
    ...getDeviceFleetOptions({ query: { owner_id: ownerId } }),
    staleTime: FLEET_STALE_TIME,
    enabled: !Number.isNaN(ownerId),
  });

  const group = query.data?.[0];

  // The response *is* one element of the list response, so it doubles as a patch
  // for the cached fleet list. Structural sharing keeps `group` stable between
  // fetches, so within a mount this only fires when the owner's fleet changed;
  // on mount it also fires once with whatever is cached, which can briefly be
  // older than the list entry it replaces until the refetch lands.
  useEffect(() => {
    if (group) spliceOwnerGroup(queryClient, group);
  }, [group, queryClient]);

  return { ...query, data: group };
}
