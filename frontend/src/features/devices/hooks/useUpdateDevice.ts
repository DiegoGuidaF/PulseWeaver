import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  updateDeviceMutation,
  getDeviceQueryKey,
  getDevicesQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useUpdateDevice(deviceId: number) {
  const queryClient = useQueryClient();

  return useMutation({
    ...updateDeviceMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: getDeviceQueryKey({ path: { device_id: deviceId } }),
      });
      queryClient.invalidateQueries({ queryKey: getDevicesQueryKey() });
    },
  });
}
