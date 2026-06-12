import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  disableAddressMutation,
  getDeviceAddressesQueryKey,
  getDevicesQueryKey,
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
      // Device lists carry enabled-address counts and derived device state
      queryClient.invalidateQueries({ queryKey: getDevicesQueryKey() });
      queryClient.invalidateQueries({ queryKey: [{ _id: "getDevicesByUser" }] });
    },
  });
}
