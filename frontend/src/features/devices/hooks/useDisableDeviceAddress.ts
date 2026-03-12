import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  disableAddressMutation,
  getDeviceAddressesQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useDisableDeviceAddress() {
  const queryClient = useQueryClient();

  return useMutation({
    ...disableAddressMutation(),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: getDeviceAddressesQueryKey({
          path: { device_id: variables.path.device_id },
        }),
      });
    },
  });
}
