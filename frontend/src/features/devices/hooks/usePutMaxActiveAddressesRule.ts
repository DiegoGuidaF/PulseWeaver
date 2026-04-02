import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  getMaxActiveAddressesRuleQueryKey,
  putMaxActiveAddressesRuleMutation,
} from "@/lib/api/@tanstack/react-query.gen";

export function usePutMaxActiveAddressesRule(deviceId: number) {
  const queryClient = useQueryClient();

  return useMutation({
    ...putMaxActiveAddressesRuleMutation({ path: { device_id: deviceId } }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: getMaxActiveAddressesRuleQueryKey({
          path: { device_id: deviceId },
        }),
      });
    },
  });
}
