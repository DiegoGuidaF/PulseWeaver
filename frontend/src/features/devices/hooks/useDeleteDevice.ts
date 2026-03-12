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
      options?.onSuccess?.();
    },
  });
}
