import { useMutation, useQueryClient } from "@tanstack/react-query";
import { disableDeviceMutation } from "@/lib/api/@tanstack/react-query.gen";
import { invalidateOwnerFleet } from "../fleetCache";

export function useDisableDevice(ownerId: number) {
  const queryClient = useQueryClient();

  return useMutation({
    ...disableDeviceMutation(),
    onSuccess: () => {
      // Disabling drops all of the device's live addresses; the owner's aggregate follows.
      invalidateOwnerFleet(queryClient, ownerId);
    },
  });
}
