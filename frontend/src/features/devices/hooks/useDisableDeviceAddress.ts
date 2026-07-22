import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  disableAddressMutation,
  getDeviceAddressesQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";
import { invalidateOwnerFleet } from "../fleetCache";

export function useDisableDeviceAddress(ownerId: number) {
  const queryClient = useQueryClient();

  return useMutation({
    ...disableAddressMutation(),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: getDeviceAddressesQueryKey({
          path: { device_id: variables.path.device_id },
        }),
      });
      // The owner's fleet group carries per-device live counts and derived state
      // alongside the fleet-wide live_address_count.
      invalidateOwnerFleet(queryClient, ownerId);
    },
  });
}
