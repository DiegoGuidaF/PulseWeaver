import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  getDeviceAddressLeaseRuleQueryKey,
  putDeviceAddressLeaseRuleMutation,
} from "@/lib/api/@tanstack/react-query.gen";
import { toErrorMessage } from "@/lib/api-client";
import { toast } from "sonner";

export function usePutDeviceAddressLeaseRule(deviceId: number) {
  const queryClient = useQueryClient();

  return useMutation({
    ...putDeviceAddressLeaseRuleMutation({ path: { device_id: deviceId } }),
    onSuccess: (_data, _variables) => {
      queryClient.invalidateQueries({
        queryKey: getDeviceAddressLeaseRuleQueryKey({
          path: { device_id: deviceId },
        }),
      });
      toast.success("Address lease rule saved");
    },
    onError: (err) => {
      toast.error("Error saving address lease rule", {
        description: toErrorMessage(err),
      });
    },
  });
}
