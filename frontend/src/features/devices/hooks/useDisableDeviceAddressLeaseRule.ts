import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  disableDeviceAddressLeaseRuleMutation,
  getDeviceAddressLeaseRuleQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useDisableDeviceAddressLeaseRule(deviceId: number) {
  const queryClient = useQueryClient();

  return useMutation({
    ...disableDeviceAddressLeaseRuleMutation({
      path: { device_id: deviceId },
    }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: getDeviceAddressLeaseRuleQueryKey({
          path: { device_id: deviceId },
        }),
      });
    },
  });
}
