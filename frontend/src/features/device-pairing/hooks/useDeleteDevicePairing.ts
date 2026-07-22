import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  deleteDevicePairingMutation,
  listDevicePairingsQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";
import { invalidateOwnerFleet } from "@/features/devices/fleetCache";

export function useDeleteDevicePairing(deviceId: number, ownerId: number) {
  const queryClient = useQueryClient();
  return useMutation({
    ...deleteDevicePairingMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: listDevicePairingsQueryKey({ path: { id: deviceId } }),
      });
      // The pairing chip is nested on the device row in the owner's fleet group.
      invalidateOwnerFleet(queryClient, ownerId);
    },
  });
}
