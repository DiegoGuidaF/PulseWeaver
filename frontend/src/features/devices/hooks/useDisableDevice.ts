import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  disableDeviceMutation,
  getDevicesQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useDisableDevice() {
  const queryClient = useQueryClient();

  return useMutation({
    ...disableDeviceMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: getDevicesQueryKey() });
      queryClient.invalidateQueries({ queryKey: [{ _id: "getDevicesByUser" }] });
    },
  });
}
