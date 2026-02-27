import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  disableDeviceAddressLeaseRuleMutation,
  getDeviceAddressLeaseRuleQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";
import { toErrorMessage } from "@/lib/api-client";
import { toast } from "sonner";

export function useDisableDeviceAddressLeaseRule(deviceId: number) {
  const queryClient = useQueryClient();

  return useMutation({
    ...disableDeviceAddressLeaseRuleMutation({
      path: { device_id: deviceId },
    }),
    onSuccess: (_data, _variables) => {
      queryClient.invalidateQueries({
        queryKey: getDeviceAddressLeaseRuleQueryKey({
          path: { device_id: deviceId },
        }),
      });
      toast.success("Address lease rule disabled");
    },
    onError: (err) => {
      toast.error("Error disabling address lease rule", {
        description: toErrorMessage(err),
      });
    },
  });
}
