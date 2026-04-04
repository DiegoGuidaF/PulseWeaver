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
      // Partial key match: invalidates all getDevicesByUser queries regardless of user_id.
      queryClient.invalidateQueries({ queryKey: [{ _id: "getDevicesByUser" }] });
      options?.onSuccess?.();
    },
  });
}
