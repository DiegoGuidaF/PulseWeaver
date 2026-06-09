import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  enableDeviceMutation,
  getDevicesQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useEnableDevice() {
  const queryClient = useQueryClient();

  return useMutation({
    ...enableDeviceMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: getDevicesQueryKey() });
      queryClient.invalidateQueries({ queryKey: [{ _id: "getDevicesByUser" }] });
    },
  });
}
