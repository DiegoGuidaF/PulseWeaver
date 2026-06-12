import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  deviceHeartbeatMutation,
  getDeviceAddressesQueryKey,
  getDevicesQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useDeviceHeartbeat() {
  const queryClient = useQueryClient();
  return useMutation({
    ...deviceHeartbeatMutation(),
    onSuccess: (_address, variables) => {
      queryClient.invalidateQueries({
        queryKey: getDeviceAddressesQueryKey({ path: { device_id: variables.path.device_id } }),
      });
      // Device lists carry enabled-address counts and derived device state
      queryClient.invalidateQueries({ queryKey: getDevicesQueryKey() });
      queryClient.invalidateQueries({ queryKey: [{ _id: "getDevicesByUser" }] });
    },
  });
}
