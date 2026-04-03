import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  deleteDeviceMutation,
  getDevicesQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useDeleteDevice(options?: { onSuccess?: () => void }) {
  const queryClient = useQueryClient();

  return useMutation({
    ...deleteDeviceMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: getDevicesQueryKey() });
      // Also invalidate per-user device caches so owner-filtered views stay fresh.
      queryClient.invalidateQueries({
        predicate: (query) =>
          (query.queryKey[0] as Record<string, unknown>)?._id === "getDevicesByUser",
      });
      options?.onSuccess?.();
    },
  });
}
