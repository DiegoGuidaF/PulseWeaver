import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  listDeviceRefsQueryKey,
  updateDeviceMutation,
} from "@/lib/api/@tanstack/react-query.gen";
import { invalidateFleetList, invalidateOwnerFleet } from "../fleetCache";

/**
 * `ownerId` is the device's owner at the time the hook is created (the page's current
 * context) — needed because a transfer moves the device out of that owner's fleet, and
 * the response only carries the resulting (possibly new) owner.
 */
export function useUpdateDevice(ownerId: number) {
  const queryClient = useQueryClient();

  return useMutation({
    ...updateDeviceMutation(),
    onSuccess: (data, variables) => {
      invalidateOwnerFleet(queryClient, ownerId);
      const transferred = data.owner_id !== ownerId;
      if (transferred) {
        invalidateOwnerFleet(queryClient, data.owner_id);
        // Only the source owner's workspace is mounted to refetch and patch the
        // cached list; the destination's group changed with nothing watching it.
        invalidateFleetList(queryClient);
      }
      // Refs carry name and owner_id; only refetch when either could have changed.
      if (transferred || variables.body.name !== undefined) {
        queryClient.invalidateQueries({ queryKey: listDeviceRefsQueryKey() });
      }
    },
  });
}
