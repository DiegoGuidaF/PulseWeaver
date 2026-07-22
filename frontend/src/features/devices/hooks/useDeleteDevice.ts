import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  deleteDeviceMutation,
  listDeviceRefsQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";
import { invalidateOwnerFleet } from "../fleetCache";

export function useDeleteDevice(ownerId: number, options?: { onSuccess?: () => void }) {
  const queryClient = useQueryClient();

  return useMutation({
    ...deleteDeviceMutation(),
    onSuccess: () => {
      // The device drops out of the owner's group and out of its aggregates.
      invalidateOwnerFleet(queryClient, ownerId);
      queryClient.invalidateQueries({ queryKey: listDeviceRefsQueryKey() });
      options?.onSuccess?.();
    },
  });
}
