import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  createDevicePairingMutation,
  listDevicePairingsQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";
import { invalidateOwnerFleet } from "@/features/devices/fleetCache";

export function useCreateDevicePairing(deviceId: number, ownerId: number) {
  const queryClient = useQueryClient();
  return useMutation({
    ...createDevicePairingMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: listDevicePairingsQueryKey({ path: { id: deviceId } }),
      });
      // The pairing chip is nested on the device row in the owner's fleet group.
      invalidateOwnerFleet(queryClient, ownerId);
    },
  });
}
