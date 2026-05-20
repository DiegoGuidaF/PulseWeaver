import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  getDeviceAddressesQueryKey,
  getDeviceAddressLeaseRuleQueryKey,
  getDevicesQueryKey,
  putDeviceAddressLeaseRuleMutation,
} from "@/lib/api/@tanstack/react-query.gen";

export function usePutDeviceAddressLeaseRule(deviceId: number) {
  const queryClient = useQueryClient();

  return useMutation({
    ...putDeviceAddressLeaseRuleMutation({ path: { device_id: deviceId } }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: getDeviceAddressLeaseRuleQueryKey({ path: { device_id: deviceId } }),
      });
      queryClient.invalidateQueries({
        queryKey: getDeviceAddressesQueryKey({ path: { device_id: deviceId } }),
      });
      queryClient.invalidateQueries({ queryKey: getDevicesQueryKey() });
    },
  });
}
