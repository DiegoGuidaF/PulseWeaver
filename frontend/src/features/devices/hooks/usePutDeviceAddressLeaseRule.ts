import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  getDeviceAddressLeaseRuleQueryKey,
  putDeviceAddressLeaseRuleMutation,
} from "@/lib/api/@tanstack/react-query.gen";

export function usePutDeviceAddressLeaseRule(deviceId: number) {
  const queryClient = useQueryClient();

  return useMutation({
    ...putDeviceAddressLeaseRuleMutation({ path: { device_id: deviceId } }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: getDeviceAddressLeaseRuleQueryKey({
          path: { device_id: deviceId },
        }),
      });
    },
  });
}
