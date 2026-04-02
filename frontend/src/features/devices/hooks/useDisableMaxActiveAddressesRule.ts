import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  disableMaxActiveAddressesRuleMutation,
  getMaxActiveAddressesRuleQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useDisableMaxActiveAddressesRule(deviceId: number) {
  const queryClient = useQueryClient();

  return useMutation({
    ...disableMaxActiveAddressesRuleMutation({
      path: { device_id: deviceId },
    }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: getMaxActiveAddressesRuleQueryKey({
          path: { device_id: deviceId },
        }),
      });
    },
  });
}
