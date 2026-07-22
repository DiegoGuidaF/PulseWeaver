import { useQuery } from "@tanstack/react-query";
import { getDeviceFleetOptions } from "@/lib/api/@tanstack/react-query.gen";
import { FLEET_STALE_TIME } from "../fleetCache";

/**
 * The top-level devices page: every owner with their whole fleet, each device
 * carrying its own rules and pairing. One request, nothing composed client-side.
 */
export function useOwnerGroups() {
  return useQuery({ ...getDeviceFleetOptions(), staleTime: FLEET_STALE_TIME });
}
