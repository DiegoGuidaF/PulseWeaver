import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  disableDeviceAddressLeaseRuleMutation,
  getDeviceAddressesQueryKey,
  getDeviceAddressLeaseRuleQueryKey,
  getDevicesQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useDisableDeviceAddressLeaseRule(deviceId: number) {
  const queryClient = useQueryClient();

  return useMutation({
    ...disableDeviceAddressLeaseRuleMutation({ path: { device_id: deviceId } }),
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
