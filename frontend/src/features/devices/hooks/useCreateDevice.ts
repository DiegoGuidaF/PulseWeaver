import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  createDeviceMutation,
  listDeviceRefsQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";
import type { CreateDeviceResponse } from "@/lib/api";
import { invalidateOwnerFleet } from "../fleetCache";

export function useCreateDevice(options?: {
  onSuccess?: (data: CreateDeviceResponse) => void;
}) {
  const queryClient = useQueryClient();

  return useMutation({
    ...createDeviceMutation(),
    onSuccess: (data) => {
      // The owner's fleet group carries both the new device row and the aggregates
      // (device_count, live_address_count) that move with it.
      invalidateOwnerFleet(queryClient, data.device.owner_id);
      // Refs gain the new device.
      queryClient.invalidateQueries({ queryKey: listDeviceRefsQueryKey() });
      options?.onSuccess?.(data);
    },
  });
}
